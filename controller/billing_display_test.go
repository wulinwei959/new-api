package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
)

// 回归:OpenAI 兼容 billing 接口的 *_USD 字段必须恒定为美元,
// 不随站点额度展示类型(USD/CNY/TOKENS/CUSTOM)漂移。
// 此前的实现按展示类型换算,CNY 模式下返回人民币数值,下游
// new-api/one-api 渠道余额查询会将其当作美元存储并层层放大。
func TestOpenAISubscriptionAmountIsAlwaysUSD(t *testing.T) {
	quota := 2 * int(common.QuotaPerUnit) // $2

	previousDisplay := operation_setting.GetQuotaDisplayType()
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = previousDisplay
	})

	for _, displayType := range []string{
		operation_setting.QuotaDisplayTypeUSD,
		operation_setting.QuotaDisplayTypeCNY,
		operation_setting.QuotaDisplayTypeTokens,
		operation_setting.QuotaDisplayTypeCustom,
	} {
		operation_setting.GetGeneralSetting().QuotaDisplayType = displayType
		assert.Equal(t, 2.0, openAISubscriptionAmount(int64(quota), false),
			"展示类型 %s 下必须仍返回美元数值", displayType)
	}

	assert.Equal(t, 100000000.0, openAISubscriptionAmount(int64(quota), true),
		"无限额度恒定返回契约上限值")
}

// 回归:total_usage 的单位恒为 0.01 美元,同样不随展示类型漂移。
func TestOpenAIUsageAmountIsAlwaysUSD(t *testing.T) {
	usedQuota := int(common.QuotaPerUnit) // $1 → total_usage = 100

	previousDisplay := operation_setting.GetQuotaDisplayType()
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = previousDisplay
	})

	for _, displayType := range []string{
		operation_setting.QuotaDisplayTypeUSD,
		operation_setting.QuotaDisplayTypeCNY,
		operation_setting.QuotaDisplayTypeTokens,
		operation_setting.QuotaDisplayTypeCustom,
	} {
		operation_setting.GetGeneralSetting().QuotaDisplayType = displayType
		assert.InDelta(t, 100.0, openAIUsageAmount(usedQuota), 0.0001,
			"展示类型 %s 下必须仍返回 0.01 美元口径", displayType)
	}
}

// 回归:人民币计价的渠道余额入库前统一换算为 USD。
// DeepSeek/硅基流动此前把人民币原值直接存入 channel.Balance,
// 前端按 USD 渲染,在 CNY 展示模式下余额会被二次放大。
func TestConvertCnyBalanceToUsd(t *testing.T) {
	previousRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		operation_setting.USDExchangeRate = previousRate
	})

	operation_setting.USDExchangeRate = 7.3
	assert.InDelta(t, 100.0, convertCnyBalanceToUsd(730), 0.0001)
	assert.InDelta(t, 0.0, convertCnyBalanceToUsd(0), 0.0001)

	// 汇率未配置(<=0)时跳过换算,保持原值并交由上层口径校正。
	operation_setting.USDExchangeRate = 0
	assert.Equal(t, 730.0, convertCnyBalanceToUsd(730))
}
