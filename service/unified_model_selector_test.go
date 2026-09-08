package service

import (
	"testing"

	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExcludeChannelIds(t *testing.T) {
	assert.Nil(t, ParseExcludeChannelIds(nil))
	assert.Nil(t, ParseExcludeChannelIds([]string{}))

	exclude := ParseExcludeChannelIds([]string{"3", "7", "bad"})
	require.NotNil(t, exclude)
	_, has3 := exclude[3]
	_, has7 := exclude[7]
	assert.True(t, has3)
	assert.True(t, has7)
	assert.Len(t, exclude, 2, "非数字条目应被跳过")
}

func TestScoreMemberUsesLiveStats(t *testing.T) {
	healthy := memberStat{
		Group:   "default",
		Enabled: true,
		Weight:  100,
		HasData: true,
		Stats: perfmetrics.ChannelStats{
			RequestCount: 100,
			SuccessRate:  100,
			AvgLatencyMs: 100,
			AvgTps:       40,
		},
	}
	score := scoreMember(healthy)
	assert.Greater(t, score, 0.9, "快且全成功的成员应得高分")
	assert.LessOrEqual(t, score, 1.0)

	degraded := memberStat{
		Group:   "default",
		Enabled: true,
		Weight:  100,
		HasData: true,
		Stats: perfmetrics.ChannelStats{
			RequestCount: 100,
			SuccessRate:  40,
			AvgLatencyMs: 10000,
			AvgTps:       1,
		},
	}
	assert.Less(t, scoreMember(degraded), 0.5, "慢且高失败的成员应得低分")
}

func TestScoreMemberNeutralWithoutData(t *testing.T) {
	noData := memberStat{Group: "default", Enabled: true, Weight: 100}
	assert.Equal(t, neutralScore, scoreMember(noData))

	zeroRequests := memberStat{Group: "default", Enabled: true, Weight: 100, HasData: true}
	assert.Equal(t, neutralScore, scoreMember(zeroRequests))
}

func TestPickWeightedCandidate(t *testing.T) {
	high := unifiedCandidate{ChannelId: 1, Score: 0.9, Weight: 100}
	low := unifiedCandidate{ChannelId: 2, Score: 0.1, Weight: 100}

	// 总权重为 0(全部 0 分)时必须回退到首个候选。
	zero := []unifiedCandidate{{ChannelId: 5, Score: 0, Weight: 100}}
	assert.Equal(t, 5, pickWeightedCandidate(zero).ChannelId)

	// 加权随机选取必须永远不出候选集合。
	candidates := []unifiedCandidate{high, low}
	for range 100 {
		picked := pickWeightedCandidate(candidates)
		require.Contains(t, []int{1, 2}, picked.ChannelId)
	}
}
