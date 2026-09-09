package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withModelRateLimitDisabled forces the in-memory backend (no Redis) and
// isolates a unique model so global window state from other tests cannot leak.
func withInMemoryModelRateLimit(t *testing.T, model string) {
	t.Helper()
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		delete(inMemoryModelRateLimitWindows, model)
	})
}

func TestReserveModelRateLimitUnconfigured(t *testing.T) {
	model := "unconfigured-model"
	withInMemoryModelRateLimit(t, model)
	require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(`{}`))

	// No entry: neither RPM nor TPM is enforced.
	assert.NoError(t, ReserveModelRateLimit(model))
	// Recording for an unconfigured model is a safe no-op.
	RecordModelTokens(model, 1000)
}

func TestReserveModelRateLimitRPM(t *testing.T) {
	model := "rpm-model"
	withInMemoryModelRateLimit(t, model)
	require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(
		`{"rpm-model": {"rpm": 3, "tpm": 0}}`,
	))

	for i := range 3 {
		assert.NoError(t, ReserveModelRateLimit(model), "request %d should be allowed", i+1)
	}
	// The fourth request exceeds the 3 RPM budget.
	err := ReserveModelRateLimit(model)
	assert.Error(t, err)
}

func TestReserveModelRateLimitTPM(t *testing.T) {
	model := "tpm-model"
	withInMemoryModelRateLimit(t, model)
	require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(
		`{"tpm-model": {"rpm": 0, "tpm": 100}}`,
	))

	// Window starts empty, so the first request passes the TPM pre-check.
	assert.NoError(t, ReserveModelRateLimit(model))

	// Record usage up to the limit; the next pre-check must reject.
	RecordModelTokens(model, 100)
	err := ReserveModelRateLimit(model)
	assert.Error(t, err)
}

func TestCheckModelRateLimitByModel(t *testing.T) {
	assert.NoError(t, setting.CheckModelRateLimitByModel(`{}`))
	assert.NoError(t, setting.CheckModelRateLimitByModel(`{"gpt-4o": {"rpm": 100, "tpm": 50000}}`))
	assert.Error(t, setting.CheckModelRateLimitByModel(`{"m": {"rpm": -1, "tpm": 0}}`))
	assert.Error(t, setting.CheckModelRateLimitByModel(`{"m": {"rpm": 0, "tpm": -5}}`))
	assert.Error(t, setting.CheckModelRateLimitByModel(`not-json`))

	// Round trip: update then read back.
	require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(`{"m": {"rpm": 7, "tpm": 42}}`))
	rpm, tpm, found := setting.GetModelRateLimit("m")
	require.True(t, found)
	assert.Equal(t, 7, rpm)
	assert.Equal(t, int64(42), tpm)
	_, _, found = setting.GetModelRateLimit("absent")
	assert.False(t, found)
}
