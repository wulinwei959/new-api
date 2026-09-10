package setting

import (
	"fmt"
	"math"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

// ModelRateLimit 保存单个模型的每分钟请求数（RPM）与每分钟 Token 数（TPM）
// 限制。取值为 0 表示该维度不限。
type ModelRateLimit struct {
	// RPM 为每分钟最大请求数，0 表示不限。
	RPM int `json:"rpm"`
	// TPM 为每分钟最大 Token 数，0 表示不限。
	TPM int64 `json:"tpm"`
}

// maxModelRateLimitRPM、maxModelRateLimitTPM 是单个模型限速允许的上界，用于把
// 计数器的乘积与比较保持在安全的 int64 范围内。TPM 的上界远低于 math.MaxInt64，
// 避免窗口内累计大量响应时求和溢出。
const (
	maxModelRateLimitRPM = int(math.MaxInt32)
	maxModelRateLimitTPM = int64(math.MaxInt64) / 4
)

var (
	modelRateLimitByModel      = map[string]ModelRateLimit{}
	modelRateLimitByModelMutex sync.RWMutex
)

// ModelRateLimitByModel2JSONString 把按模型限速配置序列化为 JSON，供选项表存储。
func ModelRateLimitByModel2JSONString() string {
	modelRateLimitByModelMutex.RLock()
	defer modelRateLimitByModelMutex.RUnlock()

	jsonBytes, err := common.Marshal(modelRateLimitByModel)
	if err != nil {
		common.SysLog("error marshalling model rate limit config: " + err.Error())
	}
	return string(jsonBytes)
}

// UpdateModelRateLimitByModelByJSONString 把 JSON 配置解析到内存中的按模型限速表。
func UpdateModelRateLimitByModelByJSONString(jsonStr string) error {
	modelRateLimitByModelMutex.Lock()
	defer modelRateLimitByModelMutex.Unlock()

	modelRateLimitByModel = make(map[string]ModelRateLimit)
	return common.Unmarshal([]byte(jsonStr), &modelRateLimitByModel)
}

// GetModelRateLimit 返回模型已配置的 RPM/TPM。found 为 false 表示该模型没有配置项，
// 即不做按模型限速。
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

// CheckModelRateLimitByModel 在选项持久化前校验配置值：每个条目的 RPM/TPM 必须非负，
// 且不超过支持的上界。
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
