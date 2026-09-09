package setting

import (
	"sync"
)

// ModelRateLimitConfig holds the per-model RPM/TPM rate limit configuration.
type ModelRateLimitConfig struct {
	RPM int `json:"rpm"`
	TPM int `json:"tpm"`
}

var (
	modelRateLimitConfig map[string]ModelRateLimitConfig
	modelRateLimitMu     sync.RWMutex
)

func init() {
	modelRateLimitConfig = make(map[string]ModelRateLimitConfig)
}

// GetModelRateLimit returns the rate limit config for a model.
func GetModelRateLimit(model string) (rpm, tpm int, found bool) {
	modelRateLimitMu.RLock()
	defer modelRateLimitMu.RUnlock()
	cfg, ok := modelRateLimitConfig[model]
	if !ok {
		return 0, 0, false
	}
	return cfg.RPM, cfg.TPM, true
}

// SetModelRateLimit updates the rate limit config for a model.
func SetModelRateLimit(model string, rpm, tpm int) {
	modelRateLimitMu.Lock()
	defer modelRateLimitMu.Unlock()
	if modelRateLimitConfig == nil {
		modelRateLimitConfig = make(map[string]ModelRateLimitConfig)
	}
	modelRateLimitConfig[model] = ModelRateLimitConfig{RPM: rpm, TPM: tpm}
}
