package middleware

import (
	"maps"
	"net"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

// ClientIPBlacklist rejects requests from client IPs in the administrator's
// blacklist before any route handler or authentication runs.
func ClientIPBlacklist() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !system_setting.IsIPBlacklisted(net.ParseIP(c.ClientIP())) {
			c.Next()
			return
		}
		message := common.NewMessage("Access from this IP address is blocked")
		body := gin.H{
			"success": false,
			"message": message.Error(),
			"error": gin.H{
				"message": message.Error(),
				"type":    "ip_blacklist",
			},
		}
		maps.Copy(body, message.Fields())
		c.AbortWithStatusJSON(http.StatusForbidden, body)
	}
}
