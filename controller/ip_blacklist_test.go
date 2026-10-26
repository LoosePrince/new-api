package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddLogIPToBlacklistAddsAndEnables(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}))
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[system_setting.IPBlacklistListOption] = `["198.51.100.4"]`
	common.OptionMap[system_setting.IPBlacklistEnabledOption] = "false"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		delete(common.OptionMap, system_setting.IPBlacklistListOption)
		delete(common.OptionMap, system_setting.IPBlacklistEnabledOption)
		common.OptionMapRWMutex.Unlock()
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/ip_blacklist", strings.NewReader(`{"ip":"203.0.113.9"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	AddLogIPToBlacklist(ctx)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			IP            string `json:"ip"`
			AlreadyListed bool   `json:"already_listed"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, "203.0.113.9", response.Data.IP)
	assert.False(t, response.Data.AlreadyListed)

	common.OptionMapRWMutex.RLock()
	enabled := common.OptionMap[system_setting.IPBlacklistEnabledOption]
	raw := common.OptionMap[system_setting.IPBlacklistListOption]
	common.OptionMapRWMutex.RUnlock()
	assert.Equal(t, "true", enabled)
	assert.JSONEq(t, `["198.51.100.4","203.0.113.9"]`, raw)

	secondRecorder := httptest.NewRecorder()
	secondContext, _ := gin.CreateTestContext(secondRecorder)
	secondContext.Request = httptest.NewRequest(http.MethodPost, "/api/log/ip_blacklist", strings.NewReader(`{"ip":"203.0.113.9"}`))
	secondContext.Request.Header.Set("Content-Type", "application/json")
	AddLogIPToBlacklist(secondContext)
	require.NoError(t, common.Unmarshal(secondRecorder.Body.Bytes(), &response))
	assert.True(t, response.Data.AlreadyListed)
}

func TestAddLogIPToBlacklistRequiresConfirmationForTheCurrentIP(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}))
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[system_setting.IPBlacklistListOption] = `[]`
	common.OptionMap[system_setting.IPBlacklistEnabledOption] = "false"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		delete(common.OptionMap, system_setting.IPBlacklistListOption)
		delete(common.OptionMap, system_setting.IPBlacklistEnabledOption)
		common.OptionMapRWMutex.Unlock()
	})

	postAsCurrentIP := func(body string) *httptest.ResponseRecorder {
		gin.SetMode(gin.TestMode)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/ip_blacklist", strings.NewReader(body))
		ctx.Request.RemoteAddr = "203.0.113.9:4321"
		ctx.Request.Header.Set("Content-Type", "application/json")
		AddLogIPToBlacklist(ctx)
		return recorder
	}

	recorder := postAsCurrentIP(`{"ip":"203.0.113.9"}`)
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"code":"current_ip_requires_confirmation"`)
	common.OptionMapRWMutex.RLock()
	assert.JSONEq(t, `[]`, common.OptionMap[system_setting.IPBlacklistListOption])
	assert.Equal(t, "false", common.OptionMap[system_setting.IPBlacklistEnabledOption])
	common.OptionMapRWMutex.RUnlock()

	confirmed := postAsCurrentIP(`{"ip":"203.0.113.9","confirm_current_ip":true}`)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(confirmed.Body.Bytes(), &response))
	assert.True(t, response.Success)
	common.OptionMapRWMutex.RLock()
	assert.JSONEq(t, `["203.0.113.9"]`, common.OptionMap[system_setting.IPBlacklistListOption])
	assert.Equal(t, "true", common.OptionMap[system_setting.IPBlacklistEnabledOption])
	common.OptionMapRWMutex.RUnlock()
}

func TestAddLogIPToBlacklistRejectsInvalidIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/ip_blacklist", strings.NewReader(`{"ip":"not-an-ip"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	AddLogIPToBlacklist(ctx)

	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.Contains(t, recorder.Body.String(), "Invalid IP address")
}
