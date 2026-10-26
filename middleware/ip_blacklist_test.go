package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func requestThroughIPBlacklist(t *testing.T, enabled, list, clientIP string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previousEnabled, hadEnabled := common.OptionMap[system_setting.IPBlacklistEnabledOption]
	previousList, hadList := common.OptionMap[system_setting.IPBlacklistListOption]
	common.OptionMap[system_setting.IPBlacklistEnabledOption] = enabled
	common.OptionMap[system_setting.IPBlacklistListOption] = list
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadEnabled {
			common.OptionMap[system_setting.IPBlacklistEnabledOption] = previousEnabled
		} else {
			delete(common.OptionMap, system_setting.IPBlacklistEnabledOption)
		}
		if hadList {
			common.OptionMap[system_setting.IPBlacklistListOption] = previousList
		} else {
			delete(common.OptionMap, system_setting.IPBlacklistListOption)
		}
	})

	engine := gin.New()
	engine.Use(ClientIPBlacklist())
	engine.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = net.JoinHostPort(clientIP, "12345")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder.Code
}

func TestClientIPBlacklistBlocksListedIPs(t *testing.T) {
	list := `["203.0.113.9","10.0.0.0/8"]`

	assert.Equal(t, http.StatusForbidden, requestThroughIPBlacklist(t, "true", list, "203.0.113.9"))
	assert.Equal(t, http.StatusForbidden, requestThroughIPBlacklist(t, "true", list, "10.4.5.6"))
	assert.Equal(t, http.StatusOK, requestThroughIPBlacklist(t, "true", list, "198.51.100.4"))
}

func TestClientIPBlacklistIgnoresDisabledOrInvalidConfiguration(t *testing.T) {
	assert.Equal(t, http.StatusOK, requestThroughIPBlacklist(t, "false", `["203.0.113.9"]`, "203.0.113.9"))
	assert.Equal(t, http.StatusOK, requestThroughIPBlacklist(t, "true", `not-json`, "203.0.113.9"))
	assert.Equal(t, http.StatusOK, requestThroughIPBlacklist(t, "true", `[]`, "203.0.113.9"))
}
