package unified_model_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateSettingsValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   Settings
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid single model",
			input: Settings{Models: []UnifiedModel{{
				Id:      "auto",
				Enabled: true,
				Channels: []ChannelMember{
					{ChannelId: 1, ModelName: "gpt-4o", Enabled: true},
				},
			}}},
		},
		{
			name:    "empty id rejected",
			input:   Settings{Models: []UnifiedModel{{Id: "  ", Enabled: true}}},
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name: "duplicate ids rejected",
			input: Settings{Models: []UnifiedModel{
				{Id: "auto"},
				{Id: "auto"},
			}},
			wantErr: true,
			errMsg:  "duplicate unified model id",
		},
		{
			name: "invalid channel id rejected",
			input: Settings{Models: []UnifiedModel{{
				Id:       "auto",
				Channels: []ChannelMember{{ChannelId: 0, ModelName: "gpt-4o", Enabled: true}},
			}}},
			wantErr: true,
			errMsg:  "invalid channel id",
		},
		{
			name: "empty member model name rejected",
			input: Settings{Models: []UnifiedModel{{
				Id:       "auto",
				Channels: []ChannelMember{{ChannelId: 1, ModelName: " ", Enabled: true}},
			}}},
			wantErr: true,
			errMsg:  "empty model name",
		},
		{
			name: "enabled model without enabled members rejected",
			input: Settings{Models: []UnifiedModel{{
				Id:       "auto",
				Enabled:  true,
				Channels: []ChannelMember{{ChannelId: 1, ModelName: "gpt-4o", Enabled: false}},
			}}},
			wantErr: true,
			errMsg:  "at least one enabled member",
		},
		{
			name: "disabled model may have zero enabled members",
			input: Settings{Models: []UnifiedModel{{
				Id:       "auto",
				Enabled:  false,
				Channels: []ChannelMember{{ChannelId: 1, ModelName: "gpt-4o", Enabled: false}},
			}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := UpdateSettings(test.input)
			if test.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.errMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateSettingsNormalizesMembers(t *testing.T) {
	input := Settings{Models: []UnifiedModel{{
		Id:      "auto",
		Enabled: true,
		Channels: []ChannelMember{
			{ChannelId: 1, ModelName: " gpt-4o ", Weight: 0, MinScore: -5, Enabled: true},
		},
	}}}
	require.NoError(t, UpdateSettings(input))

	stored, ok := GetUnifiedModel("auto")
	require.True(t, ok)
	require.Len(t, stored.Channels, 1)
	assert.Equal(t, "gpt-4o", stored.Channels[0].ModelName)
	assert.Equal(t, 100, stored.Channels[0].Weight, "zero weight must default to 100")
	assert.Equal(t, 0, stored.Channels[0].MinScore, "negative min score must clamp to 0")

	// 未配置调优值时使用默认值。
	assert.Equal(t, 1, GetScoreWindowHours())
	assert.Equal(t, 30, GetScoreCacheSeconds())

	_, ok = GetEnabledUnifiedModel("auto")
	assert.True(t, ok)
	assert.False(t, IsUnifiedModel("missing"))
}
