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
