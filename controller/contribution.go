package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// contributorScopeID returns the contributor filter for the current request.
// Administrators use zero, which means every contributor-owned channel.
func contributorScopeID(c *gin.Context) int {
	if c.GetInt("role") >= common.RoleAdminUser {
		return 0
	}
	return c.GetInt("id")
}

func requireContributorAccess(c *gin.Context) bool {
	if c.GetInt("role") >= common.RoleAdminUser {
		return true
	}
	if common.GetContextKeyBool(c, constant.ContextKeyIsGuest) {
		return true
	}
	common.ApiErrorStatus(c, http.StatusForbidden, common.NewMessage("Unauthorized, insufficient privileges"))
	return false
}

func GetContributorOverview(c *gin.Context) {
	if !requireContributorAccess(c) {
		return
	}
	overview, err := model.GetContributorOverview(contributorScopeID(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, overview)
}

func ListContributorChannels(c *gin.Context) {
	if !requireContributorAccess(c) {
		return
	}
	scopeID := contributorScopeID(c)
	pageInfo := common.GetPageQuery(c)
	channels, total, err := model.GetContributorChannels(scopeID, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, channel := range channels {
		clearChannelInfo(channel)
		if scopeID == 0 {
			annotateContributor(channel)
		}
	}
	common.ApiSuccess(c, gin.H{
		"items":     channels,
		"total":     total,
		"page":      pageInfo.GetPage(),
		"page_size": pageInfo.GetPageSize(),
	})
}

func GetContributorChannel(c *gin.Context) {
	if !requireContributorAccess(c) {
		return
	}
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	channel, err := model.GetContributorChannel(contributorScopeID(c), channelID, false)
	if err != nil {
		common.ApiErrorT(c, "Channel does not exist")
		return
	}
	clearChannelInfo(channel)
	annotateContributor(channel)
	common.ApiSuccess(c, channel)
}

func FetchContributorUpstreamModels(c *gin.Context) {
	if !requireContributorAccess(c) {
		return
	}
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	channel, err := model.GetContributorChannel(contributorScopeID(c), channelID, true)
	if err != nil {
		common.ApiErrorT(c, "Channel does not exist")
		return
	}
	if channel.Type == constant.ChannelTypeTaskPlugin {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	fetchModelsResponse(c, channel)
}

func FetchContributorModels(c *gin.Context) {
	if !requireContributorAccess(c) {
		return
	}
	var req fetchModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Type == constant.ChannelTypeTaskPlugin {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	var (
		savedChannel *model.Channel
		err          error
	)
	if req.ChannelID > 0 {
		savedChannel, err = model.GetContributorChannel(contributorScopeID(c), req.ChannelID, true)
		if err != nil || savedChannel.Type != constant.ChannelTypeAdvancedCustom {
			common.ApiErrorT(c, "Channel does not exist")
			return
		}
	}
	channel, err := buildFetchModelsPreviewChannelFromSaved(req, savedChannel)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	fetchModelsResponse(c, channel)
}

func CreateContributorChannel(c *gin.Context) {
	if !requireContributorAccess(c) {
		return
	}
	c.Set("contributor_channel", true)
	AddChannel(c)
}

func UpdateContributorChannel(c *gin.Context) {
	if !requireContributorAccess(c) {
		return
	}
	c.Set("contributor_channel", true)
	UpdateChannel(c)
}

func DisableContributorChannel(c *gin.Context) {
	if !requireContributorAccess(c) {
		return
	}
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	changed, err := model.DisableContributorChannel(contributorScopeID(c), channelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorT(c, "Channel does not exist")
			return
		}
		common.ApiError(c, err)
		return
	}
	if changed {
		model.InitChannelCache()
		closeActiveChannelWebSockets([]int{channelID})
	}
	common.ApiSuccess(c, changed)
}

func DeleteContributorChannel(c *gin.Context) {
	if !requireContributorAccess(c) {
		return
	}
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	if err := model.DeleteContributorChannel(contributorScopeID(c), channelID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorT(c, "Channel does not exist")
			return
		}
		common.ApiError(c, err)
		return
	}
	model.InitChannelCache()
	closeActiveChannelWebSockets([]int{channelID})
	common.ApiSuccess(c, true)
}
