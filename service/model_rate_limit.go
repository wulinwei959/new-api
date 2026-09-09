package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	redis "github.com/go-redis/redis/v8"
)

// modelRateLimitWindowSeconds is the fixed 1-minute window used for per-model
// RPM/TPM rate limiting. Counters are keyed by the aligned minute so the
// window rolls over on the wall clock and stale keys expire.
const modelRateLimitWindowSeconds = 60

// modelRateLimitKeyTTL keeps rate-limit counters alive long enough to span a
// window rollover, then lets Redis garbage-collect them.
const modelRateLimitKeyTTL = 2 * time.Minute

type modelRateLimitWindow struct {
	minute int64
	rpm    int
	tpm    int64
}

var (
	inMemoryModelRateLimitWindows = map[string]*modelRateLimitWindow{}
	inMemoryModelRateLimitMu      sync.Mutex
)

// ReserveModelRateLimit enforces the per-model rate limit for an incoming
// request to `model`: it pre-checks the TPM window and atomically reserves one
// RPM slot. It returns an error (to reject the request) when a limit is
// exceeded, and nil when the request may proceed.
func ReserveModelRateLimit(model string) error {
	if model == "" {
		return nil
	}
	rpm, tpm, found := setting.GetModelRateLimit(model)
	if !found || (rpm <= 0 && tpm <= 0) {
		return nil
	}
	if common.RedisEnabled {
		return reserveRedisModelRateLimit(model, rpm, tpm)
	}
	return reserveMemoryModelRateLimit(model, rpm, tpm)
}

// RecordModelTokens adds the actual response token count of a request to
// `model` into the TPM window. It is called after a successful relay so the
// next pre-checks observe the accumulated usage.
func RecordModelTokens(model string, tokens int64) {
	if model == "" || tokens <= 0 {
		return
	}
	if _, _, found := setting.GetModelRateLimit(model); !found {
		return
	}
	if common.RedisEnabled {
		recordRedisModelTokens(model, tokens)
		return
	}
	recordMemoryModelTokens(model, tokens)
}

func currentModelRateLimitMinute() int64 {
	return time.Now().Unix() / modelRateLimitWindowSeconds
}

// --- In-memory backend (single node) ---

func reserveMemoryModelRateLimit(model string, rpm int, tpm int64) error {
	inMemoryModelRateLimitMu.Lock()
	defer inMemoryModelRateLimitMu.Unlock()

	window := memoryModelRateLimitWindow(model)
	if tpm > 0 && window.tpm >= tpm {
		return fmt.Errorf("model %s has reached its token limit: %d tokens per minute", model, tpm)
	}
	window.rpm++
	if rpm > 0 && int64(window.rpm) > int64(rpm) {
		// The request was rejected, so do not count it against the window.
		window.rpm--
		return fmt.Errorf("model %s has reached its request limit: %d requests per minute", model, rpm)
	}
	return nil
}

func recordMemoryModelTokens(model string, tokens int64) {
	inMemoryModelRateLimitMu.Lock()
	defer inMemoryModelRateLimitMu.Unlock()

	memoryModelRateLimitWindow(model).tpm += tokens
}

func memoryModelRateLimitWindow(model string) *modelRateLimitWindow {
	now := currentModelRateLimitMinute()
	window, ok := inMemoryModelRateLimitWindows[model]
	if !ok || window.minute != now {
		window = &modelRateLimitWindow{minute: now}
		inMemoryModelRateLimitWindows[model] = window
	}
	return window
}

// --- Redis backend (multi node) ---

func reserveRedisModelRateLimit(model string, rpm int, tpm int64) error {
	ctx := context.Background()
	rdb := common.RDB
	minute := currentModelRateLimitMinute()

	if tpm > 0 {
		// Best-effort pre-check: reject only when the window has already
		// accumulated at least the limit. The in-flight request's own tokens
		// are recorded after the response.
		sum, err := rdb.Get(ctx, modelRateLimitRedisKey("tpm", model, minute)).Int64()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil
		}
		if sum >= tpm {
			return fmt.Errorf("model %s has reached its token limit: %d tokens per minute", model, tpm)
		}
	}

	if rpm > 0 {
		key := modelRateLimitRedisKey("rpm", model, minute)
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Fail open on Redis errors: a limiter outage must not take the
			// relay down.
			return nil
		}
		rdb.Expire(ctx, key, modelRateLimitKeyTTL)
		if count > int64(rpm) {
			return fmt.Errorf("model %s has reached its request limit: %d requests per minute", model, rpm)
		}
	}

	return nil
}

func recordRedisModelTokens(model string, tokens int64) {
	ctx := context.Background()
	rdb := common.RDB
	key := modelRateLimitRedisKey("tpm", model, currentModelRateLimitMinute())
	rdb.IncrBy(ctx, key, tokens)
	rdb.Expire(ctx, key, modelRateLimitKeyTTL)
}

func modelRateLimitRedisKey(kind, model string, minute int64) string {
	return fmt.Sprintf("model_rate_limit:%s:%s:%d", kind, model, minute)
}
