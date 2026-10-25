package controller

import (
	"net/http"
	"strconv"

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
