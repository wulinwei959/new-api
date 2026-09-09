package setting

import (
	"fmt"
	"math"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

// ModelRateLimit holds the per-model request-rate (RPM) and token-rate (TPM)
// limits for a single model. A zero value means that dimension is unlimited.
type ModelRateLimit struct {
	// RPM is the maximum number of requests per minute. 0 = unlimited.
	RPM int `json:"rpm"`
	// TPM is the maximum number of tokens per minute. 0 = unlimited.
	TPM int64 `json:"tpm"`
}

// maxModelRateLimitValues are the upper bounds a single per-model limit may take.
// They keep counter products and comparisons inside safe int64 ranges. TPM is
// capped well below math.MaxInt64 so accumulating many responses into a window
// sum cannot overflow.
const (
	maxModelRateLimitRPM = int(math.MaxInt32)
	maxModelRateLimitTPM = int64(math.MaxInt64) / 4
)

var (
	modelRateLimitByModel      = map[string]ModelRateLimit{}
	modelRateLimitByModelMutex sync.RWMutex
)

// ModelRateLimitByModel2JSONString serializes the per-model rate limits for the
// options table.
func ModelRateLimitByModel2JSONString() string {
	modelRateLimitByModelMutex.RLock()
	defer modelRateLimitByModelMutex.RUnlock()

	jsonBytes, err := common.Marshal(modelRateLimitByModel)
	if err != nil {
		common.SysLog("error marshalling model rate limit config: " + err.Error())
	}
	return string(jsonBytes)
}

// UpdateModelRateLimitByModelByJSONString parses the JSON value into the
// in-memory per-model rate limits.
func UpdateModelRateLimitByModelByJSONString(jsonStr string) error {
	modelRateLimitByModelMutex.Lock()
	defer modelRateLimitByModelMutex.Unlock()

	modelRateLimitByModel = make(map[string]ModelRateLimit)
	return common.Unmarshal([]byte(jsonStr), &modelRateLimitByModel)
}

// GetModelRateLimit returns the configured RPM/TPM for a model. found is false
// when the model has no entry, meaning no per-model limit is applied.
func GetModelRateLimit(model string) (rpm int, tpm int64, found bool) {
	modelRateLimitByModelMutex.RLock()
	defer modelRateLimitByModelMutex.RUnlock()

	if modelRateLimitByModel == nil {
		return 0, 0, false
	}
	limits, found := modelRateLimitByModel[model]
	if !found {
		return 0, 0, false
	}
	return limits.RPM, limits.TPM, true
}

// CheckModelRateLimitByModel validates an option value before it is persisted:
// each entry must have non-negative RPM/TPM within the supported upper bounds.
func CheckModelRateLimitByModel(jsonStr string) error {
	checkModelRateLimitByModel := make(map[string]ModelRateLimit)
	err := common.Unmarshal([]byte(jsonStr), &checkModelRateLimitByModel)
	if err != nil {
		return err
	}
	for model, limits := range checkModelRateLimitByModel {
		if limits.RPM < 0 || limits.TPM < 0 {
			return fmt.Errorf("model %s has negative rate limit values: rpm=%d, tpm=%d", model, limits.RPM, limits.TPM)
		}
		if limits.RPM > maxModelRateLimitRPM {
			return fmt.Errorf("model %s rpm %d exceeds max %d", model, limits.RPM, maxModelRateLimitRPM)
		}
		if limits.TPM > maxModelRateLimitTPM {
			return fmt.Errorf("model %s tpm %d exceeds max %d", model, limits.TPM, maxModelRateLimitTPM)
		}
	}
	return nil
}
