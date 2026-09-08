package service

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/unified_model_setting"

	"github.com/gin-gonic/gin"
)

// Unified model routing: a client-facing pool id (default "auto") resolves to
// one concrete upstream (channel, model) member scored by live performance
// metrics. Scores are cached briefly; channels already tried by earlier retry
// attempts are excluded so failures move to the next-best member.

const (
	// neutralScore substitutes for missing metric dimensions so freshly added
	// members stay routable before they accumulate statistics.
	neutralScore = 0.5

	// V1 fixed scoring weights; they sum to 1.
	successRateWeight  = 0.40
	latencyWeight      = 0.30
	throughputWeight   = 0.20
	availabilityWeight = 0.10

	defaultMemberWeight  = 100
	scoreLatencySmoothMs = 1000.0
)

// memberStat is one pool member's resolved group and live performance data.
type memberStat struct {
	Group   string
	Enabled bool
	Weight  int
	Stats   perfmetrics.ChannelStats
	HasData bool
}

type memberStatsCacheEntry struct {
	expiresAt time.Time
	version   int64
	stats     map[int]memberStat
}

var memberStatsCache sync.Map // key: unifiedId|tokenGroup|userGroup -> memberStatsCacheEntry

// collectMemberStats resolves each enabled member's serving group and gathers
// its performance stats over the scoring window. Members whose channel cannot
// serve the request's groups are dropped. Results are cached briefly; the
// cache is invalidated whenever the unified model settings change.
func collectMemberStats(c *gin.Context, unified unified_model_setting.UnifiedModel, tokenGroup string, userGroup string) map[int]memberStat {
	cacheKey := unified.Id + "|" + tokenGroup + "|" + userGroup
	version := unified_model_setting.Version()
	ttl := time.Duration(unified_model_setting.GetScoreCacheSeconds()) * time.Second
	if entry, ok := memberStatsCache.Load(cacheKey); ok {
		cached := entry.(memberStatsCacheEntry)
		if cached.version == version && time.Now().Before(cached.expiresAt) {
			return cached.stats
		}
	}
	stats := collectMemberStatsUncached(c, unified, tokenGroup, userGroup)
	memberStatsCache.Store(cacheKey, memberStatsCacheEntry{
		expiresAt: time.Now().Add(ttl),
		version:   version,
		stats:     stats,
	})
	return stats
}

// collectMemberStatsUncached is the live computation behind collectMemberStats.
func collectMemberStatsUncached(c *gin.Context, unified unified_model_setting.UnifiedModel, tokenGroup string, userGroup string) map[int]memberStat {
	windowHours := unified_model_setting.GetScoreWindowHours()
	statsByGroupCache := make(map[string]map[int]perfmetrics.ChannelStats)
	result := make(map[int]memberStat, len(unified.Channels))
	for _, member := range unified.Channels {
		entry := memberStat{
			Group:   "",
			Enabled: member.Enabled,
			Weight:  member.Weight,
		}
		if entry.Weight <= 0 {
			entry.Weight = defaultMemberWeight
		}
		if !member.Enabled {
			result[member.ChannelId] = entry
			continue
		}
		group, ok := resolveUnifiedMemberGroup(c, member.ChannelId, tokenGroup, userGroup)
		if !ok {
			continue
		}
		entry.Group = group
		groupStats, cached := statsByGroupCache[group]
		if !cached {
			var err error
			groupStats, err = perfmetrics.QueryChannelStats(perfmetrics.QueryParams{
				Model: unified.Id,
				Group: group,
				Hours: windowHours,
			})
			if err != nil {
				groupStats = map[int]perfmetrics.ChannelStats{}
			}
			statsByGroupCache[group] = groupStats
		}
		if stat, ok := groupStats[member.ChannelId]; ok && stat.RequestCount > 0 {
			entry.Stats = stat
			entry.HasData = true
		}
		result[member.ChannelId] = entry
	}
	return result
}

// scoreMember folds the live stats into a single [0,1] score using the fixed
// V1 weights. Members without data get the neutral score.
func scoreMember(stat memberStat) float64 {
	if !stat.HasData || stat.Stats.RequestCount == 0 {
		return neutralScore
	}
	successRate := stat.Stats.SuccessRate / 100
	latencyScore := scoreLatencySmoothMs / (scoreLatencySmoothMs + float64(stat.Stats.AvgLatencyMs))
	throughputScore := neutralScore
	if stat.Stats.AvgTps > 0 {
		// Without a pool-wide max, anchor throughput at 20 tps = 1.0 so the
		// dimension stays comparable across refreshes.
		throughputScore = math.Min(stat.Stats.AvgTps/20.0, 1)
	}
	score := successRateWeight*successRate +
		latencyWeight*latencyScore +
		throughputWeight*throughputScore +
		availabilityWeight*successRate
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return neutralScore
	}
	return math.Min(math.Max(score, 0), 1)
}

type unifiedCandidate struct {
	ChannelId int
	ModelName string
	Group     string
	Score     float64
	Weight    int
}

// SelectUnifiedModelChannel resolves a unified model id to the best available
// (channel, upstream model) member for the request's groups.
func SelectUnifiedModelChannel(c *gin.Context, unifiedId string, tokenGroup string, excludeChannelIds map[int]struct{}) (*model.Channel, string, error) {
	unified, ok := unified_model_setting.GetEnabledUnifiedModel(unifiedId)
	if !ok {
		return nil, "", fmt.Errorf("unified model %s is not enabled or has no enabled members", unifiedId)
	}
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	memberStats := collectMemberStats(c, unified, tokenGroup, userGroup)

	candidates := make([]unifiedCandidate, 0, len(unified.Channels))
	for _, member := range unified.Channels {
		if !member.Enabled {
			continue
		}
		if _, excluded := excludeChannelIds[member.ChannelId]; excluded {
			continue
		}
		entry, ok := memberStats[member.ChannelId]
		if !ok || entry.Group == "" {
			continue
		}
		score := scoreMember(entry)
		if member.MinScore > 0 && score*100 < float64(member.MinScore) {
			continue
		}
		candidates = append(candidates, unifiedCandidate{
			ChannelId: member.ChannelId,
			ModelName: member.ModelName,
			Group:     entry.Group,
			Score:     score,
			Weight:    entry.Weight,
		})
	}
	if len(candidates) == 0 {
		return nil, "", fmt.Errorf("no available channel in unified model %s pool", unifiedId)
	}

	selected := pickWeightedCandidate(candidates)
	channel, err := model.CacheGetChannel(selected.ChannelId)
	if err != nil || channel == nil {
		return nil, "", fmt.Errorf("unified model %s member channel %d unavailable: %v", unifiedId, selected.ChannelId, err)
	}
	if channel.Status != common.ChannelStatusEnabled {
		return nil, "", fmt.Errorf("unified model %s member channel %d is disabled", unifiedId, selected.ChannelId)
	}
	common.SetContextKey(c, constant.ContextKeyUnifiedModelId, unifiedId)
	common.SetContextKey(c, constant.ContextKeyUnifiedModelTarget, selected.ModelName)
	return channel, selected.Group, nil
}

// resolveUnifiedMemberGroup checks the member channel serves at least one group
// the request may use, returning that group for ratio resolution and logging.
func resolveUnifiedMemberGroup(c *gin.Context, channelId int, tokenGroup string, userGroup string) (string, bool) {
	channel, err := model.CacheGetChannel(channelId)
	if err != nil || channel == nil || channel.Status != common.ChannelStatusEnabled {
		return "", false
	}
	channelGroups := channel.GetGroups()
	if tokenGroup == "auto" {
		for _, g := range GetRequestAutoGroups(c, userGroup) {
			for _, channelGroup := range channelGroups {
				if channelGroup == g {
					return g, true
				}
			}
		}
		return "", false
	}
	for _, channelGroup := range channelGroups {
		if channelGroup == tokenGroup {
			return tokenGroup, true
		}
	}
	return "", false
}

func pickWeightedCandidate(candidates []unifiedCandidate) unifiedCandidate {
	total := 0.0
	for _, candidate := range candidates {
		total += candidate.Score * float64(candidate.Weight)
	}
	if total <= 0 {
		return candidates[0]
	}
	threshold := rand.Float64() * total
	accumulator := 0.0
	for _, candidate := range candidates {
		accumulator += candidate.Score * float64(candidate.Weight)
		if accumulator >= threshold {
			return candidate
		}
	}
	return candidates[0]
}

// UnifiedModelHealth is the admin-facing health snapshot for one member.
type UnifiedModelHealth struct {
	ChannelId     int     `json:"channel_id"`
	ModelName     string  `json:"model_name"`
	Enabled       bool    `json:"enabled"`
	Group         string  `json:"group"`
	RequestCount  int64   `json:"request_count"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatencyMs  int64   `json:"avg_latency_ms"`
	AvgTps        float64 `json:"avg_tps"`
	Score         float64 `json:"score"`
	WeightedScore float64 `json:"weighted_score"`
}

// GetUnifiedModelHealth computes the live health snapshot for a unified model
// id without touching the score cache, so the admin UI always sees fresh data.
func GetUnifiedModelHealth(c *gin.Context, unifiedId string) ([]UnifiedModelHealth, bool) {
	unified, ok := unified_model_setting.GetUnifiedModel(unifiedId)
	if !ok {
		return nil, false
	}
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	// 管理端健康视图按 auto 分组解析成员：空 tokenGroup 无法命中任何渠道分组，
	// 会导致所有成员统计恒为 0。
	memberStats := collectMemberStatsUncached(c, unified, "auto", userGroup)
	healths := make([]UnifiedModelHealth, 0, len(unified.Channels))
	for _, member := range unified.Channels {
		entry, found := memberStats[member.ChannelId]
		health := UnifiedModelHealth{
			ChannelId:    member.ChannelId,
			ModelName:    member.ModelName,
			Enabled:      member.Enabled,
			Group:        entry.Group,
			RequestCount: entry.Stats.RequestCount,
			SuccessRate:  entry.Stats.SuccessRate,
			AvgLatencyMs: entry.Stats.AvgLatencyMs,
			AvgTps:       entry.Stats.AvgTps,
		}
		if found && entry.HasData {
			health.Score = math.Round(scoreMember(entry)*1000) / 1000
		} else if member.Enabled {
			health.Score = neutralScore
		}
		weight := member.Weight
		if weight <= 0 {
			weight = defaultMemberWeight
		}
		health.WeightedScore = math.Round(health.Score*float64(weight)*10) / 10
		healths = append(healths, health)
	}
	return healths, true
}

// ParseExcludeChannelIds converts the relay use_channel string slice into the
// exclusion set used by unified selection on retry attempts.
func ParseExcludeChannelIds(useChannel []string) map[int]struct{} {
	if len(useChannel) == 0 {
		return nil
	}
	exclude := make(map[int]struct{}, len(useChannel))
	for _, raw := range useChannel {
		if id, err := strconv.Atoi(raw); err == nil {
			exclude[id] = struct{}{}
		}
	}
	return exclude
}
