package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withInMemoryModelRateLimit forces the in-memory backend (no Redis) and
// isolates unique models so global window state from other tests cannot leak.
func withInMemoryModelRateLimit(t *testing.T, models ...string) {
	t.Helper()
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		for _, model := range models {
			delete(inMemoryModelRateLimitWindows, model)
			modelRateLimitHeadroomCache.Delete(model)
		}
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

func TestModelRateLimitHasHeadroom(t *testing.T) {
	t.Run("unconfigured model always has headroom", func(t *testing.T) {
		withInMemoryModelRateLimit(t, "hr-unconfigured")
		require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(`{}`))
		assert.True(t, ModelRateLimitHasHeadroom("hr-unconfigured"))
	})

	t.Run("no headroom once the RPM window is full", func(t *testing.T) {
		model := "hr-rpm"
		withInMemoryModelRateLimit(t, model)
		require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(`{"hr-rpm": {"rpm": 2, "tpm": 0}}`))

		assert.True(t, ModelRateLimitHasHeadroom(model))
		require.NoError(t, ReserveModelRateLimit(model))
		require.NoError(t, ReserveModelRateLimit(model))
		assert.False(t, ModelRateLimitHasHeadroom(model), "the window is at the RPM cap")
	})

	t.Run("no headroom once the TPM window reaches the cap", func(t *testing.T) {
		model := "hr-tpm"
		withInMemoryModelRateLimit(t, model)
		require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(`{"hr-tpm": {"rpm": 0, "tpm": 50}}`))

		RecordModelTokens(model, 50)
		assert.False(t, ModelRateLimitHasHeadroom(model))
	})

	t.Run("headroom below the cap", func(t *testing.T) {
		model := "hr-partial"
		withInMemoryModelRateLimit(t, model)
		require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(`{"hr-partial": {"rpm": 5, "tpm": 0}}`))

		for range 3 {
			require.NoError(t, ReserveModelRateLimit(model))
		}
		assert.True(t, ModelRateLimitHasHeadroom(model), "3 of 5 RPM still leaves headroom")
	})
}

func TestFilterExhaustedMembers(t *testing.T) {
	t.Run("no limits keep every candidate", func(t *testing.T) {
		withInMemoryModelRateLimit(t, "fe-clean-a", "fe-clean-b")
		require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(`{}`))

		candidates := []unifiedCandidate{
			{ChannelId: 1, ModelName: "fe-clean-a", Score: 0.9, Weight: 100},
			{ChannelId: 2, ModelName: "fe-clean-b", Score: 0.4, Weight: 100},
		}
		kept, excluded, allExhausted := filterExhaustedMembers(candidates)
		assert.False(t, allExhausted)
		assert.Equal(t, 0, excluded)
		assert.Equal(t, candidates, kept)
	})

	t.Run("exhausted members are dropped", func(t *testing.T) {
		hot, cool := "fe-hot", "fe-cool"
		withInMemoryModelRateLimit(t, hot, cool)
		require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(`{"fe-hot": {"rpm": 1, "tpm": 0}}`))
		require.NoError(t, ReserveModelRateLimit(hot))

		candidates := []unifiedCandidate{
			{ChannelId: 1, ModelName: hot, Score: 0.9, Weight: 100},
			{ChannelId: 2, ModelName: cool, Score: 0.4, Weight: 100},
		}
		kept, excluded, allExhausted := filterExhaustedMembers(candidates)
		assert.False(t, allExhausted)
		assert.Equal(t, 1, excluded)
		require.Len(t, kept, 1)
		assert.Equal(t, cool, kept[0].ModelName, "only the member with headroom survives")
	})

	t.Run("fully exhausted pool falls back to the full set", func(t *testing.T) {
		a, b := "fe-full-a", "fe-full-b"
		withInMemoryModelRateLimit(t, a, b)
		require.NoError(t, setting.UpdateModelRateLimitByModelByJSONString(
			`{"fe-full-a": {"rpm": 1, "tpm": 0}, "fe-full-b": {"rpm": 1, "tpm": 0}}`,
		))
		require.NoError(t, ReserveModelRateLimit(a))
		require.NoError(t, ReserveModelRateLimit(b))

		candidates := []unifiedCandidate{
			{ChannelId: 1, ModelName: a, Score: 0.9, Weight: 100},
			{ChannelId: 2, ModelName: b, Score: 0.4, Weight: 100},
		}
		kept, excluded, allExhausted := filterExhaustedMembers(candidates)
		assert.True(t, allExhausted)
		assert.Equal(t, 2, excluded)
		assert.Equal(t, candidates, kept, "availability is preserved when the whole pool is exhausted")
	})
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
