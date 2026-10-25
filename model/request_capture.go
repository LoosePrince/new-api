package model

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const requestCaptureContextKey = "request_capture_id"

const requestCaptureCleanupInterval = time.Hour

var (
	lastRequestCaptureCleanup atomic.Int64
	requestCaptureSequence    atomic.Int64
)

func nextRequestCaptureID() int64 {
	now := time.Now().UnixNano()
	for {
		previous := requestCaptureSequence.Load()
		candidate := max(now, previous+1)
		if requestCaptureSequence.CompareAndSwap(previous, candidate) {
			return candidate
		}
	}
}

func scheduleRequestCaptureCleanup(retentionDays int) {
	now := time.Now().Unix()
	last := lastRequestCaptureCleanup.Load()
	if now-last < int64(requestCaptureCleanupInterval.Seconds()) || !lastRequestCaptureCleanup.CompareAndSwap(last, now) {
		return
	}
	if retentionDays < 1 {
		retentionDays = 30
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		target := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()
		if _, err := DeleteOldRequestCaptures(ctx, target); err != nil {
			common.SysLog(common.LogText("failed to clean expired request captures: %s", err.Error()))
		}
	}()
}

// RequestCapture stores one bounded upstream attempt. Bodies are intentionally
// kept outside logs so normal list and export queries remain small.
type RequestCapture struct {
	Id                  int64  `json:"id" gorm:"primaryKey"`
	RequestId           string `json:"request_id" gorm:"index;type:varchar(64)"`
	UpstreamRequestId   string `json:"upstream_request_id" gorm:"index;type:varchar(128)"`
	CreatedAt           int64  `json:"created_at" gorm:"index"`
	UserId              int    `json:"user_id" gorm:"index"`
	Username            string `json:"username" gorm:"index;default:''"`
	ModelName           string `json:"model_name" gorm:"index;default:''"`
	ChannelId           int    `json:"channel_id" gorm:"index"`
	ChannelName         string `json:"channel_name" gorm:"default:''"`
	Attempt             int    `json:"attempt"`
	RequestBody         string `json:"request_body,omitempty" gorm:"type:text"`
	UpstreamRequestBody string `json:"upstream_request_body,omitempty" gorm:"type:text"`
	ResponseBody        string `json:"response_body,omitempty" gorm:"type:text"`
	ResponseStatus      int    `json:"response_status,omitempty"`
	ResponseContentType string `json:"response_content_type,omitempty" gorm:"type:varchar(255)"`
	RequestTruncated    bool   `json:"request_truncated,omitempty"`
	UpstreamTruncated   bool   `json:"upstream_truncated,omitempty"`
	ResponseTruncated   bool   `json:"response_truncated,omitempty"`
}

type requestCaptureBody struct {
	data      []byte
	truncated bool
}

type requestCaptureResponseReader struct {
	body        io.ReadCloser
	captureID   int64
	status      int
	contentType string
	maxBytes    int64
	data        bytes.Buffer
	truncated   bool
	once        sync.Once
}

func (r *requestCaptureResponseReader) Read(p []byte) (int, error) {
	n, err := r.body.Read(p)
	if n > 0 {
		remaining := r.maxBytes - int64(r.data.Len())
		if remaining > 0 {
			if int64(n) > remaining {
				_, _ = r.data.Write(p[:remaining])
				r.truncated = true
			} else {
				_, _ = r.data.Write(p[:n])
			}
		} else {
			r.truncated = true
		}
	}
	if err == io.EOF {
		r.persist()
	}
	return n, err
}

func (r *requestCaptureResponseReader) Close() error {
	err := r.body.Close()
	r.persist()
	return err
}

func (r *requestCaptureResponseReader) persist() {
	r.once.Do(func() {
		UpdateRequestCaptureResponse(r.captureID, r.data.String(), r.status, r.contentType, r.truncated)
	})
}

func requestCaptureLimit() int64 {
	kb := operation_setting.GetLogSetting().RequestCaptureMaxBodyKB
	if kb < 1 {
		kb = 256
	}
	return int64(kb) * 1024
}

func captureBytes(reader io.Reader, maxBytes int64) (requestCaptureBody, bool) {
	if reader == nil || maxBytes <= 0 {
		return requestCaptureBody{}, false
	}
	var source io.Reader = reader
	if replayable, ok := reader.(interface{ NewReader() (io.ReadCloser, error) }); ok {
		clone, err := replayable.NewReader()
		if err != nil {
			return requestCaptureBody{}, false
		}
		defer clone.Close()
		source = clone
	}
	data, err := io.ReadAll(io.LimitReader(source, maxBytes+1))
	if err != nil {
		return requestCaptureBody{}, false
	}
	truncated := int64(len(data)) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}
	return requestCaptureBody{data: data, truncated: truncated}, true
}

func captureOriginalBody(c *gin.Context, maxBytes int64) (requestCaptureBody, bool) {
	if c == nil {
		return requestCaptureBody{}, false
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return requestCaptureBody{}, false
	}
	return captureBytes(storage, maxBytes)
}

// BeginRequestCapture creates a row for the current upstream attempt and keeps
// its id in the request context for response capture.
func BeginRequestCapture(c *gin.Context, userID int, username, modelName string, channelID, attempt int, upstreamBody io.Reader) {
	settings := operation_setting.GetLogSetting()
	if !settings.RequestCaptureEnabled || c == nil {
		return
	}
	scheduleRequestCaptureCleanup(settings.RequestCaptureRetentionDays)
	capture := &RequestCapture{
		Id:        nextRequestCaptureID(),
		RequestId: c.GetString(common.RequestIdKey),
		CreatedAt: common.GetTimestamp(),
		UserId:    userID,
		Username:  username,
		ModelName: modelName,
		ChannelId: channelID,
		Attempt:   attempt,
	}
	if channel, err := CacheGetChannel(channelID); err == nil {
		capture.ChannelName = channel.Name
	}
	if settings.RequestCaptureOriginal {
		if body, ok := captureOriginalBody(c, requestCaptureLimit()); ok {
			capture.RequestBody = string(body.data)
			capture.RequestTruncated = body.truncated
		}
	}
	if settings.RequestCaptureUpstream {
		if body, ok := captureBytes(upstreamBody, requestCaptureLimit()); ok {
			capture.UpstreamRequestBody = string(body.data)
			capture.UpstreamTruncated = body.truncated
		}
	}
	if err := LOG_DB.Create(capture).Error; err != nil {
		common.SysLog(common.LogText("failed to record request capture: %s", err.Error()))
		return
	}
	c.Set(requestCaptureContextKey, capture.Id)
}

func WrapRequestCaptureResponse(c *gin.Context, response *http.Response) {
	if c == nil || response == nil || response.Body == nil || !operation_setting.GetLogSetting().RequestCaptureResponse {
		return
	}
	id := RequestCaptureID(c)
	if id == 0 {
		return
	}
	response.Body = &requestCaptureResponseReader{
		body:        response.Body,
		captureID:   id,
		status:      response.StatusCode,
		contentType: response.Header.Get("Content-Type"),
		maxBytes:    requestCaptureLimit(),
	}
}

func UpdateRequestCaptureResponse(id int64, body string, status int, contentType string, truncated bool) {
	if id == 0 || !operation_setting.GetLogSetting().RequestCaptureResponse {
		return
	}
	updates := map[string]any{
		"response_body":         body,
		"response_status":       status,
		"response_content_type": contentType,
		"response_truncated":    truncated,
	}
	if err := LOG_DB.Model(&RequestCapture{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		common.SysLog(common.LogText("failed to update request capture response: %s", err.Error()))
	}
}

func SetRequestCaptureUpstreamRequestID(c *gin.Context, id string) {
	if c == nil || id == "" {
		return
	}
	captureID, ok := c.Get(requestCaptureContextKey)
	if !ok {
		return
	}
	parsed, ok := captureID.(int64)
	if !ok {
		return
	}
	_ = LOG_DB.Model(&RequestCapture{}).Where("id = ?", parsed).Update("upstream_request_id", id).Error
}

func GetRequestCaptures(requestID string, page, pageSize int) (items []*RequestCapture, total int64, err error) {
	query := LOG_DB.Model(&RequestCapture{}).Where("request_id = ?", requestID)
	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err = query.Order("created_at desc, id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&items).Error
	return items, total, err
}

func GetRequestCapture(id int64) (*RequestCapture, error) {
	var capture RequestCapture
	if err := LOG_DB.First(&capture, id).Error; err != nil {
		return nil, err
	}
	return &capture, nil
}

func DeleteRequestCapture(id int64) error {
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return LOG_DB.Exec("ALTER TABLE request_captures DELETE WHERE id = ? SETTINGS mutations_sync = 1", id).Error
	}
	return LOG_DB.Delete(&RequestCapture{}, id).Error
}

func DeleteOldRequestCaptures(ctx context.Context, targetTimestamp int64) (int64, error) {
	var total int64
	query := LOG_DB.WithContext(ctx).Model(&RequestCapture{}).Where("created_at < ?", targetTimestamp)
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, nil
	}
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		if err := LOG_DB.WithContext(ctx).Exec("ALTER TABLE request_captures DELETE WHERE created_at < ? SETTINGS mutations_sync = 1", targetTimestamp).Error; err != nil {
			return 0, err
		}
		return total, nil
	}
	result := query.Delete(&RequestCapture{})
	return result.RowsAffected, result.Error
}

func DeleteAllRequestCaptures(ctx context.Context) (int64, error) {
	var total int64
	if err := LOG_DB.WithContext(ctx).Model(&RequestCapture{}).Count(&total).Error; err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, nil
	}
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		if err := LOG_DB.WithContext(ctx).Exec("ALTER TABLE request_captures DELETE WHERE 1 = 1 SETTINGS mutations_sync = 1").Error; err != nil {
			return 0, err
		}
		return total, nil
	}
	if err := LOG_DB.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RequestCapture{}).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func RequestCaptureID(c *gin.Context) int64 {
	if c == nil {
		return 0
	}
	value, _ := c.Get(requestCaptureContextKey)
	id, _ := value.(int64)
	return id
}
