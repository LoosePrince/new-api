package model

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Request and error logs keep the client IP either when the user opted in or
// when an administrator forces IP recording for every account.
func TestRequestLogsRecordIPAccordingToTheForcedSetting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousForce := operation_setting.GetLogSetting().ForceRecordIP
	previousLogConsume := common.LogConsumeEnabled
	common.LogConsumeEnabled = true
	t.Cleanup(func() {
		operation_setting.GetLogSetting().ForceRecordIP = previousForce
		common.LogConsumeEnabled = previousLogConsume
	})

	for _, tc := range []struct {
		name        string
		force       bool
		userSetting bool
		wantIP      string
	}{
		{name: "user opted out and recording is not forced", wantIP: ""},
		{name: "user opted in", userSetting: true, wantIP: "203.0.113.9"},
		{name: "administrator forces recording for a user who opted out", force: true, wantIP: "203.0.113.9"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)
			operation_setting.GetLogSetting().ForceRecordIP = tc.force
			setting, err := common.Marshal(dto.UserSetting{RecordIpLog: tc.userSetting})
			require.NoError(t, err)
			user := User{Username: "ip-log-user", AffCode: "ip-log-user-aff", Setting: string(setting)}
			require.NoError(t, DB.Create(&user).Error)
			c := newRecordIPTestContext(t)

			RecordConsumeLog(c, user.Id, RecordConsumeLogParams{
				ModelName: "gpt-4o-mini",
				Quota:     1,
			})
			RecordErrorLog(c, user.Id, 1, "gpt-4o-mini", "token", "upstream failed", 0, 1, false, "default", NewLogOther())

			var logs []Log
			require.NoError(t, LOG_DB.Order("id asc").Find(&logs).Error)
			require.Len(t, logs, 2)
			assert.Equal(t, LogTypeConsume, logs[0].Type)
			assert.Equal(t, tc.wantIP, logs[0].Ip)
			assert.Equal(t, LogTypeError, logs[1].Type)
			assert.Equal(t, tc.wantIP, logs[1].Ip)
		})
	}
}

func newRecordIPTestContext(t *testing.T) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.RemoteAddr = "203.0.113.9:4321"
	c.Set("username", "ip-log-user")
	return c
}
