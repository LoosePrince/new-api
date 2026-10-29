package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

// setupPublicChannelStatusTest prepares a throwaway database holding one log
// row per model and channel, so the public page has real availability figures.
func setupPublicChannelStatusTest(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedis := common.RedisEnabled
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Log{}))
	model.DB, model.LOG_DB = db, db

	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedis
		require.NoError(t, db.Migrator().DropTable(&model.Log{}, &model.Channel{}))
		require.NoError(t, sqlDB.Close())
	})
	return db
}

func callPublicChannelStatus(t *testing.T, query string) map[string]any {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/public/channel-status"+query, nil)
	GetPublicChannelStatus(c)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	return response.Data
}

func TestPublicChannelStatusStaysHiddenWhenDisabled(t *testing.T) {
	setupPublicChannelStatusTest(t)
	require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusEnabledOption, "false"))

	data := callPublicChannelStatus(t, "")

	require.Len(t, data, 1)
	assert.Equal(t, false, data["enabled"])
	assert.NotContains(t, data, "models")
}

func TestPublicChannelStatusReportsPerModelAvailability(t *testing.T) {
	db := setupPublicChannelStatusTest(t)
	require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusEnabledOption, "true"))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusEnabledOption, "false"))
	})

	now := common.GetTimestamp()
	dayStart := now - now%channelStatusDaySeconds
	channels := []model.Channel{
		{Id: 1, Name: "primary", Status: common.ChannelStatusEnabled},
		{Id: 2, Name: "backup", Status: common.ChannelStatusEnabled},
		{Id: 3, Name: "retired", Status: common.ChannelStatusManuallyDisabled},
	}
	for i := range channels {
		require.NoError(t, db.Create(&channels[i]).Error)
	}

	logs := []model.Log{
		// Healthy model on the primary channel.
		{ChannelId: 1, ModelName: "gpt-4o", Type: model.LogTypeConsume, CreatedAt: dayStart + 10, UseTime: 2, RequestId: "a"},
		{ChannelId: 1, ModelName: "gpt-4o", Type: model.LogTypeConsume, CreatedAt: dayStart + 20, UseTime: 4, RequestId: "b"},
		// Failing model on the backup channel.
		{ChannelId: 2, ModelName: "claude", Type: model.LogTypeError, CreatedAt: dayStart + 30, UseTime: 0, RequestId: "c"},
		// Disabled channel: intentionally out of the public report.
		{ChannelId: 3, ModelName: "gemini", Type: model.LogTypeConsume, CreatedAt: dayStart + 40, UseTime: 1, RequestId: "d"},
	}
	for i := range logs {
		require.NoError(t, db.Create(&logs[i]).Error)
	}

	data := callPublicChannelStatus(t, "")
	require.Equal(t, true, data["enabled"])

	summary, ok := data["summary"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "degraded", summary["status"], "two successes out of three attempts on enabled channels")
	assert.EqualValues(t, 2, summary["channel_count"])
	assert.EqualValues(t, 2, summary["model_count"])

	models, ok := data["models"].([]any)
	require.True(t, ok)
	require.Len(t, models, 2)

	byName := make(map[string]map[string]any, len(models))
	for _, entry := range models {
		modelEntry, ok := entry.(map[string]any)
		require.True(t, ok)
		name, ok := modelEntry["name"].(string)
		require.True(t, ok)
		byName[name] = modelEntry
	}
	assert.NotContains(t, byName, "gemini", "a disabled channel is not part of the public report")

	healthy := byName["gpt-4o"]
	assert.Equal(t, "normal", healthy["status"])
	assert.EqualValues(t, 100, healthy["availability_7d"])
	servedBy, ok := healthy["channels"].([]any)
	require.True(t, ok)
	require.Len(t, servedBy, 1)
	channelEntry, ok := servedBy[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "primary", channelEntry["name"])
	assert.NotContains(t, channelEntry, "id", "the public report never exposes channel ids")

	broken := byName["claude"]
	assert.Equal(t, "unavailable", broken["status"])
	assert.EqualValues(t, 0, broken["availability_7d"])
}

func TestPublicChannelStatusSummarySkipsModelPayload(t *testing.T) {
	db := setupPublicChannelStatusTest(t)
	require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusEnabledOption, "true"))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusEnabledOption, "false"))
	})

	channel := model.Channel{Id: 1, Name: "primary", Status: common.ChannelStatusEnabled}
	require.NoError(t, db.Create(&channel).Error)
	now := common.GetTimestamp()
	dayStart := now - now%channelStatusDaySeconds
	require.NoError(t, db.Create(&model.Log{
		ChannelId: 1, ModelName: "gpt-4o", Type: model.LogTypeConsume,
		CreatedAt: dayStart + 10, UseTime: 2, RequestId: "a",
	}).Error)

	data := callPublicChannelStatus(t, "?summary=1")

	require.Equal(t, true, data["enabled"])
	assert.Contains(t, data, "summary")
	assert.Empty(t, data["models"], "the home page bar never receives the per-model payload")
}

// TestPublicChannelStatusTodayCellsCoverElapsedHours keeps the last third of the
// strip honest: today is reported hour by hour, so a request from a few hours
// ago lands in its own cell instead of a single daily block.
func TestPublicChannelStatusTodayCellsCoverElapsedHours(t *testing.T) {
	db := setupPublicChannelStatusTest(t)
	require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusEnabledOption, "true"))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusEnabledOption, "false"))
	})

	channel := model.Channel{Id: 1, Name: "primary", Status: common.ChannelStatusEnabled}
	require.NoError(t, db.Create(&channel).Error)

	now := common.GetTimestamp()
	today := now - now%channelStatusDaySeconds
	hourStart := today + channelStatusHourSeconds
	// One success in the second hour of today and one failure right after it.
	require.NoError(t, db.Create(&model.Log{
		ChannelId: 1, ModelName: "gpt-4o", Type: model.LogTypeConsume,
		CreatedAt: hourStart + 10, UseTime: 2, RequestId: "a",
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		ChannelId: 1, ModelName: "gpt-4o", Type: model.LogTypeError,
		CreatedAt: hourStart + 20, UseTime: 1, RequestId: "b",
	}).Error)

	data := callPublicChannelStatus(t, "")
	summary, ok := data["summary"].(map[string]any)
	require.True(t, ok)

	history, ok := summary["history"].([]any)
	require.True(t, ok)
	assert.Len(t, history, channelStatusHistoryDays-1, "the strip keeps the 29 completed days")

	todayCells, ok := summary["today"].([]any)
	require.True(t, ok)
	// Cells run from midnight to the current hour, never a full day early on.
	elapsedHours := (now-today)/channelStatusHourSeconds + 1
	assert.Len(t, todayCells, int(elapsedHours))
	require.GreaterOrEqual(t, len(todayCells), 2)

	covered, ok := todayCells[1].(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, hourStart, covered["hour"])
	assert.Equal(t, "abnormal", covered["status"], "one success out of two attempts in that hour")
	assert.EqualValues(t, 50, covered["availability"])

	// The first hour saw no traffic at all, which is not the same as a failure.
	idle, ok := todayCells[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, channelStatusNoTraffic, idle["status"])
	assert.Nil(t, idle["availability"])
}

// TestPublicStatusTodayClampsToAFullDay keeps the hourly grid bounded: a full
// day is 24 cells, never more.
func TestPublicStatusTodayClampsToAFullDay(t *testing.T) {
	today := int64(200 * channelStatusDaySeconds)

	cells := publicStatusToday(today, today, nil)
	require.Len(t, cells, 1, "midnight still shows the hour in progress")
	assert.Equal(t, today, cells[0].Hour)

	cells = publicStatusToday(today, today+channelStatusDaySeconds-channelStatusHourSeconds, nil)
	assert.Len(t, cells, channelStatusHoursPerDay)

	cells = publicStatusToday(today, today+channelStatusDaySeconds+5, nil)
	assert.Len(t, cells, channelStatusHoursPerDay, "a clock past midnight never grows the grid")
}

// publishPublicChannelStatusTest enables the feature and restores the previous
// switches, so each test can pick its own layout without leaking into others.
func publishPublicChannelStatusTest(t *testing.T) {
	t.Helper()
	require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusEnabledOption, "true"))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusEnabledOption, "false"))
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusHomeModeOption, setting.PublicChannelStatusHomeModeCoarse))
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusChannelModeOption, setting.PublicChannelStatusChannelModeAll))
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusChannelIDsOption, "[]"))
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusModelEnabledOption, "true"))
	})
}

// TestPublicChannelStatusChannelModes keeps the channel selector honest: the
// site and model figures never change, only which channel names are published.
func TestPublicChannelStatusChannelModes(t *testing.T) {
	db := setupPublicChannelStatusTest(t)
	publishPublicChannelStatusTest(t)

	now := common.GetTimestamp()
	dayStart := now - now%channelStatusDaySeconds
	for _, channel := range []model.Channel{
		{Id: 1, Name: "primary", Status: common.ChannelStatusEnabled},
		{Id: 2, Name: "backup", Status: common.ChannelStatusEnabled},
	} {
		require.NoError(t, db.Create(&channel).Error)
	}
	for _, entry := range []model.Log{
		{ChannelId: 1, ModelName: "gpt-4o", Type: model.LogTypeConsume, CreatedAt: dayStart + 10, UseTime: 2, RequestId: "a"},
		{ChannelId: 2, ModelName: "gpt-4o", Type: model.LogTypeConsume, CreatedAt: dayStart + 20, UseTime: 4, RequestId: "b"},
	} {
		require.NoError(t, db.Create(&entry).Error)
	}

	channelNames := func(data map[string]any) []string {
		models, ok := data["models"].([]any)
		require.True(t, ok)
		require.Len(t, models, 1)
		entry, ok := models[0].(map[string]any)
		require.True(t, ok)
		served, ok := entry["channels"].([]any)
		require.True(t, ok)
		names := make([]string, 0, len(served))
		for _, raw := range served {
			channelEntry, ok := raw.(map[string]any)
			require.True(t, ok)
			names = append(names, channelEntry["name"].(string))
		}
		return names
	}

	t.Run("all channels are named by default", func(t *testing.T) {
		data := callPublicChannelStatus(t, "")
		assert.Equal(t, []string{"backup", "primary"}, channelNames(data))
		summary := data["summary"].(map[string]any)
		assert.EqualValues(t, 2, summary["channel_count"])
		assert.EqualValues(t, 100, summary["availability_7d"])
	})

	t.Run("hiding channel details keeps the headline figures", func(t *testing.T) {
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusChannelModeOption, setting.PublicChannelStatusChannelModeNone))

		data := callPublicChannelStatus(t, "")
		assert.Empty(t, channelNames(data))
		assert.Equal(t, false, data["channels_visible"])
		summary := data["summary"].(map[string]any)
		// Hiding names must not shrink the site totals, otherwise the published
		// availability would silently disagree with the request logs.
		assert.EqualValues(t, 0, summary["channel_count"], "a hidden channel is not counted in the header")
		assert.EqualValues(t, 1, summary["model_count"])
		assert.EqualValues(t, 100, summary["availability_7d"])
		assert.EqualValues(t, 2, summary["attempts"])
	})

	t.Run("only the selected channels are named", func(t *testing.T) {
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusChannelModeOption, setting.PublicChannelStatusChannelModeSelected))
		require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusChannelIDsOption, "[2]"))

		data := callPublicChannelStatus(t, "")
		assert.Equal(t, []string{"backup"}, channelNames(data))
		assert.Equal(t, true, data["channels_visible"])
		summary := data["summary"].(map[string]any)
		assert.EqualValues(t, 1, summary["channel_count"], "the header counts the published channel only")
		assert.EqualValues(t, 1, summary["model_count"])
		assert.EqualValues(t, 100, summary["availability_7d"])
		assert.EqualValues(t, 2, summary["attempts"])
	})
}

// TestPublicChannelStatusHidesModelSection covers the independent model switch.
func TestPublicChannelStatusHidesModelSection(t *testing.T) {
	db := setupPublicChannelStatusTest(t)
	publishPublicChannelStatusTest(t)

	channel := model.Channel{Id: 1, Name: "primary", Status: common.ChannelStatusEnabled}
	require.NoError(t, db.Create(&channel).Error)
	now := common.GetTimestamp()
	dayStart := now - now%channelStatusDaySeconds
	require.NoError(t, db.Create(&model.Log{
		ChannelId: 1, ModelName: "gpt-4o", Type: model.LogTypeConsume,
		CreatedAt: dayStart + 10, UseTime: 2, RequestId: "a",
	}).Error)

	data := callPublicChannelStatus(t, "")
	models, ok := data["models"].([]any)
	require.True(t, ok)
	require.Len(t, models, 1)

	require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusModelEnabledOption, "false"))

	data = callPublicChannelStatus(t, "")
	assert.Empty(t, data["models"], "the model section is an independent publication switch")
	summary, ok := data["summary"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "normal", summary["status"], "the site summary stays published")
}

// TestPublicHomeModeDrivesHomeLayoutAndPageAccess pins the four published
// combinations, because they are the whole point of a single home mode.
func TestPublicHomeModeDrivesHomeLayoutAndPageAccess(t *testing.T) {
	setupPublicChannelStatusTest(t)
	publishPublicChannelStatusTest(t)

	cases := []struct {
		mode         string
		wantDetailed bool
		wantPage     bool
	}{
		{mode: setting.PublicChannelStatusHomeModeCoarse, wantDetailed: false, wantPage: true},
		{mode: setting.PublicChannelStatusHomeModeCoarseNoPage, wantDetailed: false, wantPage: false},
		{mode: setting.PublicChannelStatusHomeModeDetailed, wantDetailed: true, wantPage: true},
		{mode: setting.PublicChannelStatusHomeModeDetailedNoPage, wantDetailed: true, wantPage: false},
	}

	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusHomeModeOption, tc.mode))

			data := callPublicChannelStatus(t, "?summary=1")
			assert.Equal(t, tc.wantDetailed, data["home_detailed"])
			assert.Equal(t, tc.wantPage, data["page_enabled"])
			assert.Equal(t, true, data["enabled"])
		})
	}
}

// TestPublicChannelStatusChannelsWithoutIDsNeverLeakNames guards the hand-picked
// list against stale ids: a channel that is gone simply disappears.
func TestPublicChannelStatusChannelsWithoutIDsNeverLeakNames(t *testing.T) {
	setupPublicChannelStatusTest(t)
	publishPublicChannelStatusTest(t)

	require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusChannelModeOption, setting.PublicChannelStatusChannelModeSelected))
	require.NoError(t, setting.UpdatePublicHomeOption(setting.PublicChannelStatusChannelIDsOption, "[999]"))

	data := callPublicChannelStatus(t, "")
	models, ok := data["models"].([]any)
	require.True(t, ok)
	assert.Empty(t, models, "a stale channel selection publishes nothing rather than guessing")
}
