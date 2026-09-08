package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/unified_model_setting"

	"github.com/gin-gonic/gin"
)

type unifiedModelSettingsRequest struct {
	Models            []unified_model_setting.UnifiedModel `json:"models"`
	ScoreWindowHours  int                                  `json:"score_window_hours"`
	ScoreCacheSeconds int                                  `json:"score_cache_seconds"`
}

type unifiedModelWithHealth struct {
	unified_model_setting.UnifiedModel
	Health      []service.UnifiedModelHealth `json:"health"`
	MemberCount int                          `json:"member_count"`
}

// GetUnifiedModels 返回配置的统一模型列表及其实时健康度快照。
func GetUnifiedModels(c *gin.Context) {
	settings := unified_model_setting.GetSettings()
	result := make([]unifiedModelWithHealth, 0, len(settings.Models))
	for _, unifiedModel := range settings.Models {
		healths, ok := service.GetUnifiedModelHealth(c, unifiedModel.Id)
		if !ok {
			healths = []service.UnifiedModelHealth{}
		}
		result = append(result, unifiedModelWithHealth{
			UnifiedModel: unifiedModel,
			Health:       healths,
			MemberCount:  len(unifiedModel.Channels),
		})
	}
	common.ApiSuccess(c, gin.H{
		"models":              result,
		"score_window_hours":  settings.ScoreWindowHours,
		"score_cache_seconds": settings.ScoreCacheSeconds,
	})
}

// UpdateUnifiedModels 校验并整体替换统一模型配置,经选项表持久化。
func UpdateUnifiedModels(c *gin.Context) {
	var request unifiedModelSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	next := unified_model_setting.Settings{
		Models:            request.Models,
		ScoreWindowHours:  request.ScoreWindowHours,
		ScoreCacheSeconds: request.ScoreCacheSeconds,
	}
	if err := unified_model_setting.UpdateSettings(next); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	modelsJson, err := common.Marshal(next.Models)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateOption("unified_model_setting.models", string(modelsJson)); err != nil {
		common.ApiError(c, err)
		return
	}
	if request.ScoreWindowHours > 0 {
		if err := model.UpdateOption("unified_model_setting.score_window_hours", strconv.Itoa(request.ScoreWindowHours)); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if request.ScoreCacheSeconds > 0 {
		if err := model.UpdateOption("unified_model_setting.score_cache_seconds", strconv.Itoa(request.ScoreCacheSeconds)); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	common.ApiSuccess(c, nil)
}

// GetUnifiedModelHealth 返回单个统一模型的实时成员健康度。
func GetUnifiedModelHealth(c *gin.Context) {
	unifiedId := c.Query("id")
	if unifiedId == "" {
		common.ApiErrorMsg(c, "id is required")
		return
	}
	healths, ok := service.GetUnifiedModelHealth(c, unifiedId)
	if !ok {
		common.ApiErrorMsg(c, "unified model not found")
		return
	}
	common.ApiSuccess(c, healths)
}

type unifiedModelChannelOption struct {
	Id     int      `json:"id"`
	Name   string   `json:"name"`
	Type   int      `json:"type"`
	Status int      `json:"status"`
	Models []string `json:"models"`
	Groups []string `json:"groups"`
}

// GetUnifiedModelChannels 返回成员编辑器的渠道下拉数据。
func GetUnifiedModelChannels(c *gin.Context) {
	// selectAll=true 返回全部渠道；否则 Limit(0) 查不到任何数据。
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	options := make([]unifiedModelChannelOption, 0, len(channels))
	for _, channel := range channels {
		options = append(options, unifiedModelChannelOption{
			Id:     channel.Id,
			Name:   channel.Name,
			Type:   channel.Type,
			Status: channel.Status,
			Models: channel.GetModels(),
			Groups: channel.GetGroups(),
		})
	}
	common.ApiSuccess(c, options)
}
