package controller

import (
	"maps"
	"net"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

type addIPBlacklistRequest struct {
	IP string `json:"ip"`
	// ConfirmCurrentIP lets the caller block the IP it is connected from. The
	// web console asks for a second confirmation before setting it.
	ConfirmCurrentIP bool `json:"confirm_current_ip"`
}

// AddLogIPToBlacklist appends a client IP recorded in a log to the inbound IP
// blacklist and enables the blacklist so the block takes effect immediately.
func AddLogIPToBlacklist(c *gin.Context) {
	var request addIPBlacklistRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	ip := strings.TrimSpace(request.IP)
	targetIP := net.ParseIP(ip)
	if targetIP == nil {
		common.ApiErrorT(c, "Invalid IP address")
		return
	}
	// Blocking the caller's own address would lock it out immediately, so the
	// first request only asks for confirmation and changes nothing.
	if !request.ConfirmCurrentIP && targetIP.Equal(net.ParseIP(c.ClientIP())) {
		message := common.NewMessage("This is your current IP address")
		body := gin.H{
			"success": false,
			"message": message.Error(),
			"code":    "current_ip_requires_confirmation",
		}
		maps.Copy(body, message.Fields())
		c.AbortWithStatusJSON(http.StatusConflict, body)
		return
	}
	entries, err := system_setting.GetIPBlacklistEntries()
	if err != nil {
		common.ApiErrorT(c, "Invalid IP or CIDR range: {{entry}}", map[string]any{"entry": ip})
		return
	}
	alreadyListed := false
	for _, entry := range entries {
		if strings.TrimSpace(entry) == ip {
			alreadyListed = true
			break
		}
	}
	if !alreadyListed {
		entries = append(entries, ip)
	}
	encoded, err := common.Marshal(entries)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	values := map[string]string{
		system_setting.IPBlacklistListOption:    string(encoded),
		system_setting.IPBlacklistEnabledOption: "true",
	}
	if err := model.UpdateOptionsBulk(values); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"ip": ip, "already_listed": alreadyListed})
}
