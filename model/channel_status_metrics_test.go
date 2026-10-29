package model

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openChannelStatusLogDB opens a throwaway log database per dialect so the
// aggregation SQL is exercised on every supported backend.
func openChannelStatusLogDB(t *testing.T, dialect string) *gorm.DB {
	t.Helper()
	var driver gorm.Dialector
	switch dialect {
	case "sqlite":
		driver = sqlite.Open(":memory:")
	case "mysql":
		dsn := os.Getenv("TEST_MYSQL_DSN")
		if dsn == "" {
			t.Skip("TEST_MYSQL_DSN is not configured")
		}
		driver = mysql.Open(dsn)
	case "postgres":
		dsn := os.Getenv("TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("TEST_POSTGRES_DSN is not configured")
		}
		driver = postgres.Open(dsn)
	default:
		t.Fatalf("unknown dialect %q", dialect)
	}
	db, err := gorm.Open(driver, &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.Migrator().DropTable(&Log{}))
	require.NoError(t, db.AutoMigrate(&Log{}))
	t.Cleanup(func() {
		require.NoError(t, db.Migrator().DropTable(&Log{}))
		require.NoError(t, sqlDB.Close())
	})
	return db
}

func TestGetChannelStatusMetricsGroupsPerDayAndType(t *testing.T) {
	const day = int64(86400)
	dayStart := int64(20000 * day)
	previousDay := dayStart - day

	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db := openChannelStatusLogDB(t, dialect)
			previousLogDB := LOG_DB
			LOG_DB = db
			t.Cleanup(func() { LOG_DB = previousLogDB })

			logs := []Log{
				{ChannelId: 7, ModelName: "gpt-4o", Type: LogTypeConsume, CreatedAt: dayStart + 10, UseTime: 2},
				{ChannelId: 7, ModelName: "gpt-4o", Type: LogTypeConsume, CreatedAt: dayStart + 20, UseTime: 4},
				{ChannelId: 7, ModelName: "gpt-4o", Type: LogTypeError, CreatedAt: dayStart + 30, UseTime: 0},
				{ChannelId: 7, ModelName: "claude", Type: LogTypeConsume, CreatedAt: dayStart + 40, UseTime: 1},
				{ChannelId: 7, ModelName: "gpt-4o", Type: LogTypeConsume, CreatedAt: previousDay + 5, UseTime: 3},
				// Ignored: other log types and logs without a channel.
				{ChannelId: 7, ModelName: "gpt-4o", Type: LogTypeTopup, CreatedAt: dayStart + 50},
				{ChannelId: 0, ModelName: "gpt-4o", Type: LogTypeConsume, CreatedAt: dayStart + 60},
			}
			for i := range logs {
				require.NoError(t, db.Create(&logs[i]).Error)
			}

			metrics, err := GetChannelStatusMetrics(previousDay, dayStart+day-1)
			require.NoError(t, err)

			type key struct {
				day   int64
				model string
				typ   int
			}
			got := make(map[key]ChannelStatusMetric, len(metrics))
			for _, metric := range metrics {
				got[key{metric.Day, metric.ModelName, metric.Type}] = metric
			}

			require.Len(t, got, 4)
			today := got[key{dayStart, "gpt-4o", LogTypeConsume}]
			assert.EqualValues(t, 2, today.Count)
			assert.EqualValues(t, 6, today.UseTime)
			assert.EqualValues(t, dayStart+20, today.LastActivity)
			assert.EqualValues(t, 1, got[key{dayStart, "gpt-4o", LogTypeError}].Count)
			assert.EqualValues(t, 1, got[key{dayStart, "claude", LogTypeConsume}].Count)
			assert.EqualValues(t, 1, got[key{previousDay, "gpt-4o", LogTypeConsume}].Count)
		})
	}
}

// TestGetChannelStatusHourlyMetricsGroupsByHour covers the current-day buckets
// of the public status strip: requests within the same hour aggregate together,
// and the next hour stays a separate cell even though it is the same day.
func TestGetChannelStatusHourlyMetricsGroupsByHour(t *testing.T) {
	const hour = int64(3600)
	dayStart := int64(20000 * 86400)
	hourStart := dayStart + 5*hour

	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db := openChannelStatusLogDB(t, dialect)
			previousLogDB := LOG_DB
			LOG_DB = db
			t.Cleanup(func() { LOG_DB = previousLogDB })

			logs := []Log{
				{ChannelId: 7, ModelName: "gpt-4o", Type: LogTypeConsume, CreatedAt: hourStart + 10, UseTime: 2},
				{ChannelId: 7, ModelName: "gpt-4o", Type: LogTypeError, CreatedAt: hourStart + 20, UseTime: 1},
				{ChannelId: 7, ModelName: "gpt-4o", Type: LogTypeConsume, CreatedAt: hourStart + hour + 30, UseTime: 4},
			}
			for i := range logs {
				require.NoError(t, db.Create(&logs[i]).Error)
			}

			metrics, err := GetChannelStatusHourlyMetrics(dayStart, dayStart+86399)
			require.NoError(t, err)

			type key struct {
				hour int64
				typ  int
			}
			got := make(map[key]ChannelStatusMetric, len(metrics))
			for _, metric := range metrics {
				got[key{metric.Day, metric.Type}] = metric
			}

			require.Len(t, got, 3)
			assert.EqualValues(t, 1, got[key{hourStart, LogTypeConsume}].Count)
			assert.EqualValues(t, 1, got[key{hourStart, LogTypeError}].Count)
			// The next hour is its own cell, with its own latency.
			nextHour := got[key{hourStart + hour, LogTypeConsume}]
			assert.EqualValues(t, 1, nextHour.Count)
			assert.EqualValues(t, 4, nextHour.UseTime)
		})
	}
}

// fallbackLogOther builds the `other` payload a log row carries for one request,
// holding the request policy events recorded so far.
func fallbackLogOther(t *testing.T, events ...map[string]any) string {
	t.Helper()
	payload := map[string]any{"admin_info": map[string]any{"request_policy": events}}
	encoded, err := common.Marshal(payload)
	require.NoError(t, err)
	return string(encoded)
}

func fallbackAttemptEvent(attempt int, channelID int, modelName string, elapsedMS int64) map[string]any {
	return map[string]any{
		"attempt":       attempt,
		"channel_id":    channelID,
		"attempt_model": modelName,
		"elapsed_ms":    elapsedMS,
		"decision":      map[string]any{"action": "attempt"},
	}
}

func fallbackOutcomeEvent(attempt int, channelID int, modelName string, action string, elapsedMS int64) map[string]any {
	decision := map[string]any{"action": action}
	if action == "failure" {
		decision["reason"] = "upstream_failure"
	}
	return map[string]any{
		"attempt":       attempt,
		"channel_id":    channelID,
		"attempt_model": modelName,
		"elapsed_ms":    elapsedMS,
		"decision":      decision,
	}
}

func fallbackRetryEvent(attempt int, channelID int, elapsedMS int64) map[string]any {
	return map[string]any{
		"attempt":    attempt,
		"channel_id": channelID,
		"elapsed_ms": elapsedMS,
		"decision":   map[string]any{"action": "retry", "reason": "retry_status_matched"},
	}
}

// TestGetChannelStatusMetricsCountsEveryFallbackAttempt covers the fallback
// chain of a single request: three models failed before the fourth answered. Each
// failed model must count as failed on its own channel, and the winner must count
// as a success with its own latency instead of the latency of the whole request.
func TestGetChannelStatusMetricsCountsEveryFallbackAttempt(t *testing.T) {
	const day = int64(86400)
	dayStart := int64(21000 * day)

	// Attempt 1 failed after 60s, attempt 2 after 100s, attempt 3 after 90s, and
	// attempt 4 answered in 5s: 255s end to end.
	attempt1 := []map[string]any{
		fallbackAttemptEvent(1, 2, "gpt-4o", 0),
		fallbackOutcomeEvent(1, 2, "gpt-4o", "failure", 60_000),
		fallbackRetryEvent(1, 2, 60_000),
	}
	attempt2 := append(append([]map[string]any{}, attempt1...),
		fallbackAttemptEvent(2, 3, "claude-3-7-sonnet", 60_000),
		fallbackOutcomeEvent(2, 3, "claude-3-7-sonnet", "failure", 160_000),
		fallbackRetryEvent(2, 3, 160_000),
	)
	attempt3 := append(append([]map[string]any{}, attempt2...),
		fallbackAttemptEvent(3, 4, "gemini-2.5-pro", 160_000),
		fallbackOutcomeEvent(3, 4, "gemini-2.5-pro", "failure", 250_000),
		fallbackRetryEvent(3, 4, 250_000),
	)
	attempt4 := append(append([]map[string]any{}, attempt3...),
		fallbackAttemptEvent(4, 5, "o3", 250_000),
		fallbackOutcomeEvent(4, 5, "o3", "success", 255_000),
	)

	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db := openChannelStatusLogDB(t, dialect)
			previousLogDB := LOG_DB
			LOG_DB = db
			t.Cleanup(func() { LOG_DB = previousLogDB })

			// One row per attempt: each row repeats the chain known at that time.
			logs := []Log{
				{ChannelId: 2, ModelName: "gpt-4o", Type: LogTypeError, CreatedAt: dayStart + 60, UseTime: 60, RequestId: "fallback-chain", Other: fallbackLogOther(t, attempt1...)},
				{ChannelId: 3, ModelName: "claude-3-7-sonnet", Type: LogTypeError, CreatedAt: dayStart + 160, UseTime: 100, RequestId: "fallback-chain", Other: fallbackLogOther(t, attempt2...)},
				{ChannelId: 4, ModelName: "gemini-2.5-pro", Type: LogTypeError, CreatedAt: dayStart + 250, UseTime: 90, RequestId: "fallback-chain", Other: fallbackLogOther(t, attempt3...)},
				{ChannelId: 5, ModelName: "o3", Type: LogTypeConsume, CreatedAt: dayStart + 255, UseTime: 255, RequestId: "fallback-chain", Other: fallbackLogOther(t, attempt4...)},
			}
			for i := range logs {
				require.NoError(t, db.Create(&logs[i]).Error)
			}

			metrics, err := GetChannelStatusMetrics(dayStart, dayStart+day-1)
			require.NoError(t, err)

			got := make(map[string]ChannelStatusMetric, len(metrics))
			for _, metric := range metrics {
				require.EqualValues(t, dayStart, metric.Day)
				got[metric.ModelName] = metric
			}

			// Four attempts on four models, never the four rows counted twice.
			require.Len(t, got, 4)
			failed := got["gpt-4o"]
			assert.Equal(t, 2, failed.ChannelID)
			assert.Equal(t, LogTypeError, failed.Type)
			assert.EqualValues(t, 1, failed.Count)
			assert.EqualValues(t, 60, failed.UseTime, "the failed attempt keeps its own latency")

			failed = got["claude-3-7-sonnet"]
			assert.Equal(t, 3, failed.ChannelID)
			assert.Equal(t, LogTypeError, failed.Type)
			assert.EqualValues(t, 1, failed.Count)
			assert.EqualValues(t, 100, failed.UseTime)

			failed = got["gemini-2.5-pro"]
			assert.Equal(t, 4, failed.ChannelID)
			assert.Equal(t, LogTypeError, failed.Type)
			assert.EqualValues(t, 1, failed.Count)
			assert.EqualValues(t, 90, failed.UseTime)

			success := got["o3"]
			assert.Equal(t, 5, success.ChannelID)
			assert.Equal(t, LogTypeConsume, success.Type)
			assert.EqualValues(t, 1, success.Count)
			assert.EqualValues(t, 5, success.UseTime, "a fallback success records its own latency, not the request's")
		})
	}
}

// TestGetChannelStatusMetricsSkipsUnattributableAttempts keeps attempts that
// never reached a known channel or model out of the status counters, and keeps
// the row's own outcome when the record never stated it.
func TestGetChannelStatusMetricsSkipsUnattributableAttempts(t *testing.T) {
	const day = int64(86400)
	dayStart := int64(22000 * day)

	// Attempt 1 has no channel and no model: nothing can be attributed to it.
	unknownAttempt := []map[string]any{
		fallbackAttemptEvent(1, 0, "", 0),
		fallbackOutcomeEvent(1, 0, "", "failure", 10_000),
		fallbackRetryEvent(1, 0, 10_000),
	}
	// Attempt 2 only opened; its outcome lives on the row itself.
	openAttempt := append(append([]map[string]any{}, unknownAttempt...),
		fallbackAttemptEvent(2, 9, "gpt-4o-mini", 10_000),
	)

	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db := openChannelStatusLogDB(t, dialect)
			previousLogDB := LOG_DB
			LOG_DB = db
			t.Cleanup(func() { LOG_DB = previousLogDB })

			logs := []Log{
				{ChannelId: 9, ModelName: "gpt-4o-mini", Type: LogTypeConsume, CreatedAt: dayStart + 15, UseTime: 5, RequestId: "unknown-first", Other: fallbackLogOther(t, openAttempt...)},
			}
			for i := range logs {
				require.NoError(t, db.Create(&logs[i]).Error)
			}

			metrics, err := GetChannelStatusMetrics(dayStart, dayStart+day-1)
			require.NoError(t, err)
			require.Len(t, metrics, 1)
			assert.Equal(t, 9, metrics[0].ChannelID)
			assert.Equal(t, "gpt-4o-mini", metrics[0].ModelName)
			assert.Equal(t, LogTypeConsume, metrics[0].Type)
			assert.EqualValues(t, 1, metrics[0].Count)
			assert.EqualValues(t, 5, metrics[0].UseTime)
		})
	}
}

func TestGetHistoricalChannelModelsHonorsWindow(t *testing.T) {
	const day = int64(86400)
	now := int64(30000 * day)

	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db := openChannelStatusLogDB(t, dialect)
			previousLogDB := LOG_DB
			LOG_DB = db
			t.Cleanup(func() { LOG_DB = previousLogDB })

			logs := []Log{
				{ChannelId: 3, ModelName: "recent", Type: LogTypeConsume, CreatedAt: now - day},
				{ChannelId: 3, ModelName: "ancient", Type: LogTypeConsume, CreatedAt: now - 90*day},
				{ChannelId: 4, ModelName: "other", Type: LogTypeConsume, CreatedAt: now - day},
			}
			for i := range logs {
				require.NoError(t, db.Create(&logs[i]).Error)
			}

			models, err := GetHistoricalChannelModels([]int{3}, now-30*day)
			require.NoError(t, err)
			assert.Equal(t, map[int]map[string]struct{}{3: {"recent": {}}}, models)

			historical, err := HasHistoricalChannelModel(3, "recent", now-30*day)
			require.NoError(t, err)
			assert.True(t, historical)

			expired, err := HasHistoricalChannelModel(3, "ancient", now-30*day)
			require.NoError(t, err)
			assert.False(t, expired)

			unknown, err := HasHistoricalChannelModel(3, "never-used", now-30*day)
			require.NoError(t, err)
			assert.False(t, unknown)

			none, err := GetHistoricalChannelModels(nil, now-30*day)
			require.NoError(t, err)
			assert.Empty(t, none)
		})
	}
}
