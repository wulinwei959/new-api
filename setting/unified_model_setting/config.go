package unified_model_setting

import (
	"errors"
	"strings"
	"sync/atomic"

	"github.com/QuantumNous/new-api/setting/config"
)

// DefaultUnifiedModelId is the reserved model name clients call to trigger
// unified-model routing.
const DefaultUnifiedModelId = "auto"

// ChannelMember is one pool entry of a unified model: a concrete upstream
// (channel, model) pair that requests may be routed to.
type ChannelMember struct {
	ChannelId int    `json:"channel_id"`
	ModelName string `json:"model_name"`
	Weight    int    `json:"weight"`    // score multiplier, 0 treated as default
	MinScore  int    `json:"min_score"` // 0 = no admission threshold
	Enabled   bool   `json:"enabled"`
}

// UnifiedModel is a named pool of upstream (channel, model) members exposed to
// clients as a single callable model id.
type UnifiedModel struct {
	Id       string          `json:"id"`
	Enabled  bool            `json:"enabled"`
	Channels []ChannelMember `json:"channels"`
}

// Settings is the admin-managed unified model configuration. It is persisted
// in the options table through config.GlobalConfig (key: unified_model_setting).
type Settings struct {
	Models            []UnifiedModel `json:"models"`
	ScoreWindowHours  int            `json:"score_window_hours"`
	ScoreCacheSeconds int            `json:"score_cache_seconds"`
}

const (
	defaultScoreWindowHours  = 1
	defaultScoreCacheSeconds = 30
)

var settings = Settings{
	Models:            []UnifiedModel{},
	ScoreWindowHours:  defaultScoreWindowHours,
	ScoreCacheSeconds: defaultScoreCacheSeconds,
}

func init() {
	config.GlobalConfig.Register("unified_model_setting", &settings)
}

func GetSettings() *Settings {
	return &settings
}

// GetUnifiedModel returns a copy of the named unified model.
func GetUnifiedModel(id string) (UnifiedModel, bool) {
	id = strings.TrimSpace(id)
	for _, m := range settings.Models {
		if m.Id == id {
			return m, true
		}
	}
	return UnifiedModel{}, false
}

// GetEnabledUnifiedModel returns the named unified model only when it is
// enabled and has at least one enabled member.
func GetEnabledUnifiedModel(id string) (UnifiedModel, bool) {
	m, ok := GetUnifiedModel(id)
	if !ok || !m.Enabled {
		return UnifiedModel{}, false
	}
	hasEnabled := false
	for _, member := range m.Channels {
		if member.Enabled {
			hasEnabled = true
			break
		}
	}
	if !hasEnabled {
		return UnifiedModel{}, false
	}
	return m, true
}

// IsUnifiedModel reports whether the given client model name resolves to a
// configured unified model (enabled or not), so relay validation can accept it.
func IsUnifiedModel(id string) bool {
	_, ok := GetUnifiedModel(id)
	return ok
}

// UnifiedModelIds lists all configured unified model ids.
func UnifiedModelIds() []string {
	ids := make([]string, 0, len(settings.Models))
	for _, m := range settings.Models {
		ids = append(ids, m.Id)
	}
	return ids
}

func GetScoreWindowHours() int {
	if settings.ScoreWindowHours > 0 {
		return settings.ScoreWindowHours
	}
	return defaultScoreWindowHours
}

func GetScoreCacheSeconds() int {
	if settings.ScoreCacheSeconds > 0 {
		return settings.ScoreCacheSeconds
	}
	return defaultScoreCacheSeconds
}

// UpdateSettings validates and replaces the whole configuration atomically.
func UpdateSettings(next Settings) error {
	seen := make(map[string]struct{}, len(next.Models))
	for i := range next.Models {
		model := &next.Models[i]
		model.Id = strings.TrimSpace(model.Id)
		if model.Id == "" {
			return errors.New("unified model id cannot be empty")
		}
		if _, dup := seen[model.Id]; dup {
			return errors.New("duplicate unified model id: " + model.Id)
		}
		seen[model.Id] = struct{}{}
		enabledMembers := 0
		for j := range model.Channels {
			member := &model.Channels[j]
			if member.ChannelId <= 0 {
				return errors.New("unified model " + model.Id + " has a member with invalid channel id")
			}
			if strings.TrimSpace(member.ModelName) == "" {
				return errors.New("unified model " + model.Id + " has a member with empty model name")
			}
			member.ModelName = strings.TrimSpace(member.ModelName)
			if member.Weight <= 0 {
				member.Weight = 100
			}
			if member.MinScore < 0 {
				member.MinScore = 0
			}
			if member.Enabled {
				enabledMembers++
			}
		}
		if model.Enabled && enabledMembers == 0 {
			return errors.New("enabled unified model " + model.Id + " needs at least one enabled member")
		}
	}
	settings.Models = next.Models
	if next.ScoreWindowHours > 0 {
		settings.ScoreWindowHours = next.ScoreWindowHours
	} else {
		settings.ScoreWindowHours = defaultScoreWindowHours
	}
	if next.ScoreCacheSeconds > 0 {
		settings.ScoreCacheSeconds = next.ScoreCacheSeconds
	} else {
		settings.ScoreCacheSeconds = defaultScoreCacheSeconds
	}
	return nil
}

// SettingsVersion increments whenever the configuration changes so the score
// cache can be invalidated cheaply.
var settingsVersion atomic.Int64

func BumpVersion() {
	settingsVersion.Add(1)
}

func Version() int64 {
	return settingsVersion.Load()
}
