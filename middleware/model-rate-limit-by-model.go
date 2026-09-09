package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// ModelRateLimitByModel enforces the per-model RPM/TPM limits configured in the
// "ModelRateLimitByModel" option. It is mounted after Distribute() on the text
// relay routes, where the requested model name is available on the context as
// original_model. Requests beyond a model's RPM or TPM budget are rejected with
// 429 before any relay work happens.
//
// For unified (auto) models the request is additionally reserved against the
// resolved member model (ContextKeyUnifiedModelTarget), so per-member limits
// also bound auto traffic. Unconfigured names are no-ops.
func ModelRateLimitByModel() func(c *gin.Context) {
	return func(c *gin.Context) {
		model := common.GetContextKeyString(c, constant.ContextKeyOriginalModel)
		target := common.GetContextKeyString(c, constant.ContextKeyUnifiedModelTarget)
		if model == "" && target == "" {
			c.Next()
			return
		}

		if err := service.ReserveModelRateLimit(model); err != nil {
			abortWithOpenAiMessage(c, http.StatusTooManyRequests, err.Error())
			return
		}
		if target != "" && target != model {
			if err := service.ReserveModelRateLimit(target); err != nil {
				abortWithOpenAiMessage(c, http.StatusTooManyRequests, err.Error())
				return
			}
		}
		c.Next()
	}
}
