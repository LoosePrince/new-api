package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetRequestCaptures(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	requestID := c.Query("request_id")
	if requestID == "" {
		common.ApiErrorT(c, "Request ID is required")
		return
	}
	items, total, err := model.GetRequestCaptures(requestID, pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetRequestCapture(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorT(c, "Invalid request capture ID")
		return
	}
	capture, err := model.GetRequestCapture(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, capture)
}

func DeleteRequestCapture(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorT(c, "Invalid request capture ID")
		return
	}
	if err := model.DeleteRequestCapture(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func DeleteRequestCapturesBefore(c *gin.Context) {
	timestamp, err := strconv.ParseInt(c.Query("before"), 10, 64)
	if err != nil || timestamp <= 0 {
		common.ApiErrorT(c, "Invalid cleanup timestamp")
		return
	}
	count, err := model.DeleteOldRequestCaptures(c.Request.Context(), timestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"deleted_count": count})
}

func DeleteAllRequestCaptures(c *gin.Context) {
	count, err := model.DeleteAllRequestCaptures(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"deleted_count": count})
}

func ExportRequestCaptures(c *gin.Context) {
	requestID := c.Query("request_id")
	if requestID == "" {
		common.ApiErrorT(c, "Request ID is required")
		return
	}
	items, _, err := model.GetRequestCaptures(requestID, 1, 1000)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	switch format := c.DefaultQuery("format", "ndjson"); format {
	case "ndjson":
	case "rendered", "text":
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Cache-Control", "no-store")
		c.Status(http.StatusOK)
		_, _ = c.Writer.WriteString(renderRequestCaptures(requestID, items))
		return
	default:
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	for _, item := range items {
		line, err := common.Marshal(item)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if _, err := c.Writer.Write(append(line, '\n')); err != nil {
			return
		}
	}
}

// renderRequestCaptures turns captured attempts into plain text a person can
// read without a JSON viewer. Streamed responses are expanded event by event.
func renderRequestCaptures(requestID string, items []*model.RequestCapture) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Request records\nRequest ID: %s\nAttempts: %d\n", requestID, len(items))
	if len(items) == 0 {
		builder.WriteString("\nNo request records.\n")
		return builder.String()
	}
	for _, item := range items {
		fmt.Fprintf(&builder, "\n===== Attempt %d =====\n", item.Attempt)
		fmt.Fprintf(&builder, "Created at: %s\n", time.Unix(item.CreatedAt, 0).UTC().Format(time.RFC3339))
		if item.ModelName != "" {
			fmt.Fprintf(&builder, "Model: %s\n", item.ModelName)
		}
		fmt.Fprintf(&builder, "Channel: #%d", item.ChannelId)
		if item.ChannelName != "" {
			fmt.Fprintf(&builder, " (%s)", item.ChannelName)
		}
		builder.WriteString("\n")
		if item.ResponseStatus != 0 {
			fmt.Fprintf(&builder, "Response status: %d\n", item.ResponseStatus)
		}
		if item.ResponseContentType != "" {
			fmt.Fprintf(&builder, "Response content type: %s\n", item.ResponseContentType)
		}
		appendCapturedBody(&builder, "Original request", item.RequestBody, item.RequestTruncated)
		appendCapturedBody(&builder, "Upstream request", item.UpstreamRequestBody, item.UpstreamTruncated)
		appendCapturedBody(&builder, "Upstream response", item.ResponseBody, item.ResponseTruncated)
	}
	return builder.String()
}

func appendCapturedBody(builder *strings.Builder, label, body string, truncated bool) {
	if body == "" {
		return
	}
	if truncated {
		label += " (truncated)"
	}
	fmt.Fprintf(builder, "\n[%s]\n%s\n", label, renderCapturedBody(body))
}

func renderCapturedBody(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "(empty)"
	}
	if pretty, err := common.IndentJson([]byte(trimmed)); err == nil {
		return string(pretty)
	}
	if strings.Contains(trimmed, "data:") {
		if rendered := renderEventStream(trimmed); rendered != "" {
			return rendered
		}
	}
	return trimmed
}

func renderEventStream(body string) string {
	var builder strings.Builder
	eventIndex := 0
	for line := range strings.SplitSeq(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		payload, isData := strings.CutPrefix(trimmed, "data:")
		if !isData {
			if strings.HasPrefix(trimmed, "event:") || strings.HasPrefix(trimmed, "id:") || strings.HasPrefix(trimmed, "retry:") {
				continue
			}
			builder.WriteString(trimmed)
			builder.WriteString("\n")
			continue
		}
		payload = strings.TrimSpace(payload)
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			builder.WriteString("[DONE]\n")
			continue
		}
		eventIndex++
		fmt.Fprintf(&builder, "--- event %d ---\n", eventIndex)
		if pretty, err := common.IndentJson([]byte(payload)); err == nil {
			builder.Write(pretty)
		} else {
			builder.WriteString(payload)
		}
		builder.WriteString("\n")
	}
	if eventIndex == 0 {
		return ""
	}
	return builder.String()
}
