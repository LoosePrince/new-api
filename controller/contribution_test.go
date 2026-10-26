package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeContributorChannelPatchPreservesOmittedConfiguration(t *testing.T) {
	baseURL := "https://upstream.example/v1"
	setting := `{"proxy":"http://proxy.example"}`
	origin := &model.Channel{
		Id:            42,
		ContributorId: 7,
		Key:           "secret-key",
		Status:        common.ChannelStatusManuallyDisabled,
		Name:          "original",
		Group:         "default",
		BaseURL:       &baseURL,
		Setting:       &setting,
		OtherSettings: `{"disable_store":true}`,
	}
	patch := &PatchChannel{}

	require.NoError(t, mergeContributorChannelPatch(patch, origin, map[string]any{
		"id":   float64(origin.Id),
		"name": "renamed",
	}))

	assert.Equal(t, origin.Id, patch.Id)
	assert.Equal(t, "renamed", patch.Name)
	assert.Equal(t, origin.Key, patch.Key)
	assert.Equal(t, origin.Status, patch.Status)
	assert.Equal(t, origin.Group, patch.Group)
	assert.Equal(t, origin.BaseURL, patch.BaseURL)
	assert.Equal(t, origin.Setting, patch.Setting)
	assert.Equal(t, origin.OtherSettings, patch.OtherSettings)
}

func TestContributorModelDiscoveryUsesOwnedChannelAndDraftSettings(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	gin.SetMode(gin.TestMode)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"draft-model"}]}`))
	}))
	t.Cleanup(server.Close)

	baseURL := server.URL
	owned := &model.Channel{
		ContributorId: 17,
		Type:          1,
		Name:          "owned channel",
		Key:           "owned-secret",
		Models:        "manual-model",
		Group:         "default",
		BaseURL:       &baseURL,
	}
	require.NoError(t, db.Create(owned).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(owned.Id)}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/contribution/channels/1/fetch_models", nil)
	ctx.Set("id", 17)
	ctx.Set(string(constant.ContextKeyIsGuest), true)
	FetchContributorUpstreamModels(ctx)

	var response struct {
		Success bool     `json:"success"`
		Data    []string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, []string{"draft-model"}, response.Data)
	assert.NotContains(t, recorder.Body.String(), "owned-secret")

	foreignRecorder := httptest.NewRecorder()
	foreignContext, _ := gin.CreateTestContext(foreignRecorder)
	foreignContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(owned.Id)}}
	foreignContext.Request = httptest.NewRequest(http.MethodGet, "/api/contribution/channels/1/fetch_models", nil)
	foreignContext.Set("id", 18)
	foreignContext.Set(string(constant.ContextKeyIsGuest), true)
	FetchContributorUpstreamModels(foreignContext)
	assert.Contains(t, foreignRecorder.Body.String(), "Channel does not exist")
}

func TestContributorModelPreviewRejectsArbitrarySavedChannelID(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	foreignChannel := &model.Channel{
		ContributorId: 18,
		Type:          1,
		Name:          "foreign channel",
		Key:           "foreign-secret",
		Models:        "foreign-model",
		Group:         "default",
	}
	require.NoError(t, db.Create(foreignChannel).Error)
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/contribution/channels/fetch_models", strings.NewReader(fmt.Sprintf(`{"type":1,"channel_id":%d}`, foreignChannel.Id)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 17)
	ctx.Set(string(constant.ContextKeyIsGuest), true)

	FetchContributorModels(ctx)

	assert.Contains(t, recorder.Body.String(), "Channel does not exist")
}

func TestAdministratorContributionScopeCoversEveryContributor(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.QuotaData{}))
	require.NoError(t, db.Create(&model.User{Id: 7, Username: "alice", Password: "secret", AffCode: "alicecode", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.User{Id: 17, Username: "bob", Password: "secret", AffCode: "bobcode", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}).Error)
	aliceChannel := &model.Channel{ContributorId: 7, Name: "alice-channel", Key: "alice-key", Status: common.ChannelStatusEnabled, Group: "default"}
	bobChannel := &model.Channel{ContributorId: 17, Name: "bob-channel", Key: "bob-key", Status: common.ChannelStatusEnabled, Group: "default"}
	privateChannel := &model.Channel{ContributorId: 0, Name: "admin-channel", Key: "admin-key", Status: common.ChannelStatusEnabled, Group: "default"}
	for _, channel := range []*model.Channel{aliceChannel, bobChannel, privateChannel} {
		require.NoError(t, db.Create(channel).Error)
	}
	require.NoError(t, db.Create(&model.QuotaData{ChannelID: aliceChannel.Id, ModelName: "gpt-a", Count: 3, TokenUsed: 30}).Error)
	require.NoError(t, db.Create(&model.QuotaData{ChannelID: bobChannel.Id, ModelName: "gpt-b", Count: 5, TokenUsed: 50}).Error)
	require.NoError(t, db.Create(&model.QuotaData{ChannelID: privateChannel.Id, ModelName: "gpt-private", Count: 7, TokenUsed: 70}).Error)

	adminList := httptest.NewRecorder()
	adminContext, _ := gin.CreateTestContext(adminList)
	adminContext.Request = httptest.NewRequest(http.MethodGet, "/api/contribution/channels?p=1&page_size=20", nil)
	adminContext.Set("id", 3)
	adminContext.Set("role", common.RoleAdminUser)
	ListContributorChannels(adminContext)

	var adminResponse struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.Channel `json:"items"`
			Total int64           `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(adminList.Body.Bytes(), &adminResponse))
	assert.True(t, adminResponse.Success)
	assert.EqualValues(t, 2, adminResponse.Data.Total)
	require.Len(t, adminResponse.Data.Items, 2)
	usernames := map[string]string{}
	for _, channel := range adminResponse.Data.Items {
		usernames[channel.Name] = channel.ContributorUsername
	}
	assert.Equal(t, "alice", usernames["alice-channel"])
	assert.Equal(t, "bob", usernames["bob-channel"])

	guestList := httptest.NewRecorder()
	guestContext, _ := gin.CreateTestContext(guestList)
	guestContext.Request = httptest.NewRequest(http.MethodGet, "/api/contribution/channels?p=1&page_size=20", nil)
	guestContext.Set("id", 7)
	guestContext.Set("role", common.RoleCommonUser)
	guestContext.Set(string(constant.ContextKeyIsGuest), true)
	ListContributorChannels(guestContext)

	var guestResponse struct {
		Data struct {
			Items []model.Channel `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(guestList.Body.Bytes(), &guestResponse))
	require.Len(t, guestResponse.Data.Items, 1)
	assert.Equal(t, "alice-channel", guestResponse.Data.Items[0].Name)

	adminOverview := httptest.NewRecorder()
	overviewContext, _ := gin.CreateTestContext(adminOverview)
	overviewContext.Request = httptest.NewRequest(http.MethodGet, "/api/contribution/overview", nil)
	overviewContext.Set("id", 3)
	overviewContext.Set("role", common.RoleAdminUser)
	GetContributorOverview(overviewContext)

	var overviewResponse struct {
		Success bool                      `json:"success"`
		Data    model.ContributorOverview `json:"data"`
	}
	require.NoError(t, common.Unmarshal(adminOverview.Body.Bytes(), &overviewResponse))
	assert.True(t, overviewResponse.Success)
	assert.EqualValues(t, 2, overviewResponse.Data.ChannelCount)
	assert.EqualValues(t, 2, overviewResponse.Data.EnabledChannelCount)
	assert.EqualValues(t, 8, overviewResponse.Data.RequestCount)
	assert.EqualValues(t, 80, overviewResponse.Data.TokenUsed)
}
