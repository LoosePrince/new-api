package model

import (
	"os"
	"testing"

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
