package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderRequestCapturesExpandsJSONAndEventStreams(t *testing.T) {
	items := []*model.RequestCapture{
		{
			RequestId:           "req-1",
			Attempt:             1,
			CreatedAt:           1700000000,
			ModelName:           "gpt-4o-mini",
			ChannelId:           9,
			ChannelName:         "primary",
			ResponseStatus:      200,
			ResponseContentType: "text/event-stream",
			RequestBody:         `{"prompt":"hello"}`,
			UpstreamRequestBody: `{"input":"hello"}`,
			ResponseBody:        "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n",
			ResponseTruncated:   true,
		},
	}

	rendered := renderRequestCaptures("req-1", items)

	assert.Contains(t, rendered, "Request ID: req-1")
	assert.Contains(t, rendered, "===== Attempt 1 =====")
	assert.Contains(t, rendered, "Model: gpt-4o-mini")
	assert.Contains(t, rendered, "Channel: #9 (primary)")
	assert.Contains(t, rendered, "[Original request]\n{\n  \"prompt\": \"hello\"\n}")
	assert.Contains(t, rendered, "--- event 1 ---")
	assert.Contains(t, rendered, "\"content\": \"hi\"")
	assert.Contains(t, rendered, "[DONE]")
	assert.Contains(t, rendered, "[Upstream response (truncated)]")
}

func TestRenderRequestCapturesHandlesEmptyAndNonJSONBodies(t *testing.T) {
	assert.Contains(t, renderRequestCaptures("req-empty", nil), "No request records.")

	rendered := renderRequestCaptures("req-plain", []*model.RequestCapture{
		{Attempt: 2, RequestBody: "plain text body"},
	})
	assert.Contains(t, rendered, "[Original request]\nplain text body")
}

func TestExportRequestCapturesSupportsRenderedFormat(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.RequestCapture{}))
	require.NoError(t, db.Create(&model.RequestCapture{
		Id:          1,
		RequestId:   "req-rendered",
		Attempt:     1,
		RequestBody: `{"prompt":"hello"}`,
	}).Error)
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/request-captures/export?request_id=req-rendered&format=rendered", nil)
	ExportRequestCaptures(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, recorder.Body.String(), "Request ID: req-rendered")
	assert.Contains(t, recorder.Body.String(), "\"prompt\": \"hello\"")

	invalidRecorder := httptest.NewRecorder()
	invalidContext, _ := gin.CreateTestContext(invalidRecorder)
	invalidContext.Request = httptest.NewRequest(http.MethodGet, "/api/request-captures/export?request_id=req-rendered&format=unknown", nil)
	ExportRequestCaptures(invalidContext)
	assert.Contains(t, invalidRecorder.Body.String(), `"success":false`)
}
