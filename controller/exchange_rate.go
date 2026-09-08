package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// 公开汇率源,依次回退。两者都返回 {"rates": {"CNY": ...}} 结构,
// 且无需 API Key(已实测确认响应格式)。源 URL 为硬编码常量,
// 不接收用户输入,无 SSRF 面。
var exchangeRateSourceURLs = []string{
	"https://open.er-api.com/v6/latest/USD",
	"https://api.frankfurter.dev/v1/latest?base=USD&symbols=CNY",
}

const exchangeRateFetchTimeout = 10 * time.Second

// RefreshExchangeRate 从公开汇率源拉取实时 USD→CNY 汇率,更新
// USDExchangeRate 选项并返回新值。管理员仍可在货币与展示页手动输入覆盖。
func RefreshExchangeRate(c *gin.Context) {
	rate, source, err := fetchLatestUsdToCnyRate()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateOption("USDExchangeRate", strconv.FormatFloat(rate, 'f', -1, 64)); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"rate":   rate,
		"source": source,
	})
}

type exchangeRatePayload struct {
	Rates map[string]float64 `json:"rates"`
}

func fetchLatestUsdToCnyRate() (float64, string, error) {
	var lastErr error
	for _, source := range exchangeRateSourceURLs {
		rate, err := fetchUsdToCnyFromSource(source)
		if err != nil {
			lastErr = err
			continue
		}
		return rate, source, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no exchange rate source configured")
	}
	return 0, "", fmt.Errorf("所有汇率源请求失败: %w", lastErr)
}

func fetchUsdToCnyFromSource(sourceURL string) (float64, error) {
	client, err := service.GetHttpClientWithProxy("")
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), exchangeRateFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("汇率源返回状态码 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	var payload exchangeRatePayload
	if err := common.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("解析汇率响应失败: %w", err)
	}
	rate := payload.Rates["CNY"]
	if rate <= 0 {
		return 0, errors.New("汇率响应中缺少有效的 CNY 汇率")
	}
	return rate, nil
}
