package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
)

// openAISubscriptionAmount 按 OpenAI 兼容契约返回 hard_limit_usd 等字段的值:
// 恒定为美元(额度 ÷ QuotaPerUnit),不随站点展示类型(USD/CNY/TOKENS)漂移。
// 展示层的币种换算由各客户端基于自身汇率配置完成;若在此处按展示类型换算,
// 下游 new-api/one-api 渠道余额查询会把人民币数值当作美元存储,导致层级放大。
func openAISubscriptionAmount(totalQuota int64, unlimited bool) float64 {
	if unlimited {
		return 100000000
	}
	return float64(totalQuota) / common.QuotaPerUnit
}

// openAIUsageAmount 返回 total_usage 的值,单位恒为 0.01 美元(OpenAI 契约)。
func openAIUsageAmount(usedQuota int) float64 {
	return float64(usedQuota) / common.QuotaPerUnit * 100
}

func GetSubscription(c *gin.Context) {
	var remainQuota int
	var usedQuota int
	var err error
	var token *model.Token
	var expiredTime int64
	if common.DisplayTokenStatEnabled {
		tokenId := c.GetInt("token_id")
		token, err = model.GetTokenById(tokenId)
		expiredTime = token.ExpiredTime
		remainQuota = token.RemainQuota
		usedQuota = token.UsedQuota
	} else {
		userId := c.GetInt("id")
		remainQuota, err = model.GetUserQuota(userId, false)
		usedQuota, err = model.GetUserUsedQuota(userId)
	}
	if expiredTime <= 0 {
		expiredTime = 0
	}
	if err != nil {
		openAIError := types.OpenAIError{
			Message: err.Error(),
			Type:    "upstream_error",
		}
		c.JSON(200, gin.H{
			"error": openAIError,
		})
		return
	}
	quota := remainQuota + usedQuota
	unlimited := token != nil && token.UnlimitedQuota
	amount := openAISubscriptionAmount(int64(quota), unlimited)
	subscription := OpenAISubscriptionResponse{
		Object:             "billing_subscription",
		HasPaymentMethod:   true,
		SoftLimitUSD:       amount,
		HardLimitUSD:       amount,
		SystemHardLimitUSD: amount,
		AccessUntil:        expiredTime,
	}
	c.JSON(200, subscription)
	return
}

func GetUsage(c *gin.Context) {
	var quota int
	var err error
	var token *model.Token
	if common.DisplayTokenStatEnabled {
		tokenId := c.GetInt("token_id")
		token, err = model.GetTokenById(tokenId)
		quota = token.UsedQuota
	} else {
		userId := c.GetInt("id")
		quota, err = model.GetUserUsedQuota(userId)
	}
	if err != nil {
		openAIError := types.OpenAIError{
			Message: err.Error(),
			Type:    "new_api_error",
		}
		c.JSON(200, gin.H{
			"error": openAIError,
		})
		return
	}
	usage := OpenAIUsageResponse{
		Object:     "list",
		TotalUsage: openAIUsageAmount(quota),
	}
	c.JSON(200, usage)
	return
}
