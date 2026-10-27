package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelHealthStatusThresholds(t *testing.T) {
	percent := func(value float64) *float64 { return &value }

	cases := []struct {
		name         string
		availability *float64
		want         string
	}{
		{name: "no logs", availability: nil, want: channelStatusNoTraffic},
		{name: "full availability", availability: percent(100), want: channelStatusNormal},
		{name: "normal boundary", availability: percent(95), want: channelStatusNormal},
		{name: "latency boundary", availability: percent(94.9), want: channelStatusLatency},
		{name: "latency floor", availability: percent(85), want: channelStatusLatency},
		{name: "degraded boundary", availability: percent(84.9), want: channelStatusDegraded},
		{name: "degraded floor", availability: percent(60), want: channelStatusDegraded},
		{name: "abnormal boundary", availability: percent(59.9), want: channelStatusAbnormal},
		{name: "abnormal floor", availability: percent(30), want: channelStatusAbnormal},
		{name: "unavailable boundary", availability: percent(29.9), want: channelStatusUnavailable},
		{name: "no successes", availability: percent(0), want: channelStatusUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, channelHealthStatus(tc.availability))
		})
	}
}

func TestChannelStatusAggregateAvailability(t *testing.T) {
	aggregate := channelStatusAggregate{}
	aggregate.add(model.ChannelStatusMetric{Type: model.LogTypeConsume, Count: 3, UseTime: 2, LastActivity: 100})
	aggregate.add(model.ChannelStatusMetric{Type: model.LogTypeError, Count: 1, UseTime: 0, LastActivity: 120})
	assert.Equal(t, int64(4), aggregate.Attempts)
	assert.Equal(t, int64(3), aggregate.Successes)
	assert.Equal(t, int64(120), aggregate.LastActivity)

	availability := aggregate.availability()
	require.NotNil(t, availability)
	assert.InDelta(t, 75.0, *availability, 0.001)
	// 2 seconds of elapsed time across 4 attempts.
	assert.InDelta(t, 500.0, aggregate.averageLatencyMS(), 0.001)

	empty := channelStatusAggregate{}
	assert.Nil(t, empty.availability())
	assert.Zero(t, empty.averageLatencyMS())
}

func TestChannelStatusHistoryKeepsDailyGaps(t *testing.T) {
	const day = channelStatusDaySeconds
	startDay := int64(100 * day)
	today := startDay + 2*day
	metrics := map[int64]channelStatusAggregate{
		startDay + day: {Attempts: 2, Successes: 2},
	}

	history := channelStatusHistory(startDay, today, metrics)

	require.Len(t, history, 3)
	assert.Equal(t, startDay, history[0].Day)
	assert.Equal(t, channelStatusNoTraffic, history[0].Status)
	assert.Equal(t, channelStatusNormal, history[1].Status)
	assert.Equal(t, today, history[2].Day)
	assert.Equal(t, channelStatusNoTraffic, history[2].Status)
}

func TestConfiguredChannelModelsTrimsAndDeduplicates(t *testing.T) {
	channel := &model.Channel{Models: " gpt-4o , gpt-4o-mini ,gpt-4o,,"}

	ordered, set := configuredChannelModels(channel)

	assert.Equal(t, []string{"gpt-4o", "gpt-4o-mini"}, ordered)
	assert.Len(t, set, 2)
}

func TestRemovedHistoricalModelsExcludesConfiguredOnes(t *testing.T) {
	historical := map[string]struct{}{
		"gpt-live":      {},
		"gpt-removed":   {},
		"claude-legacy": {},
	}
	configured := map[string]struct{}{"gpt-live": {}}

	removed := removedHistoricalModels(historical, configured)

	assert.Equal(t, []string{"claude-legacy", "gpt-removed"}, removed)
}
