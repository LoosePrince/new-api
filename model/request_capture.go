package model

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	lastRequestCaptureCleanup  atomic.Int64
	requestCaptureSequence     atomic.Int64
	requestCaptureBodySequence atomic.Int64
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

func nextRequestCaptureBodyID() int64 {
	now := time.Now().UnixNano()
	for {
		previous := requestCaptureBodySequence.Load()
		candidate := max(now, previous+1)
		if requestCaptureBodySequence.CompareAndSwap(previous, candidate) {
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
	Id                    int64  `json:"id" gorm:"primaryKey"`
	RequestId             string `json:"request_id" gorm:"index;type:varchar(64)"`
	UpstreamRequestId     string `json:"upstream_request_id" gorm:"index;type:varchar(128)"`
	CreatedAt             int64  `json:"created_at" gorm:"index"`
	UserId                int    `json:"user_id" gorm:"index"`
	Username              string `json:"username" gorm:"index;default:''"`
	ModelName             string `json:"model_name" gorm:"index;default:''"`
	ChannelId             int    `json:"channel_id" gorm:"index"`
	ChannelName           string `json:"channel_name" gorm:"default:''"`
	Attempt               int    `json:"attempt"`
	RequestBody           string `json:"request_body,omitempty" gorm:"type:text"`
	UpstreamRequestBody   string `json:"upstream_request_body,omitempty" gorm:"type:text"`
	ResponseBody          string `json:"response_body,omitempty" gorm:"type:text"`
	ResponseStatus        int    `json:"response_status,omitempty"`
	ResponseContentType   string `json:"response_content_type,omitempty" gorm:"type:varchar(255)"`
	RequestTruncated      bool   `json:"request_truncated,omitempty"`
	UpstreamTruncated     bool   `json:"upstream_truncated,omitempty"`
	ResponseTruncated     bool   `json:"response_truncated,omitempty"`
	RequestBodyId         int64  `json:"-" gorm:"index"`
	UpstreamRequestBodyId int64  `json:"-" gorm:"index"`
	ResponseBodyId        int64  `json:"-" gorm:"index"`
}

// RequestCaptureBody stores each distinct captured body once. RequestCapture
// rows keep references to these records so retries do not duplicate large text.
type RequestCaptureBody struct {
	Id          int64  `json:"-" gorm:"primaryKey"`
	ContentHash string `json:"-" gorm:"index:idx_request_capture_body_hash,unique;type:varchar(64)"`
	CreatedAt   int64  `json:"-" gorm:"index"`
	Body        string `json:"-" gorm:"type:text"`
	Truncated   bool   `json:"-"`
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

func storeRequestCaptureBody(body requestCaptureBody) (int64, error) {
	if len(body.data) == 0 {
		return 0, nil
	}
	hashInput := append(bytes.Clone(body.data), byte(0))
	if body.truncated {
		hashInput[len(hashInput)-1] = 1
	}
	digest := sha256.Sum256(hashInput)
	hash := hex.EncodeToString(digest[:])
	var stored RequestCaptureBody
	err := LOG_DB.Where("content_hash = ?", hash).First(&stored).Error
	if err == nil {
		return stored.Id, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	stored = RequestCaptureBody{
		Id:          nextRequestCaptureBodyID(),
		ContentHash: hash,
		CreatedAt:   common.GetTimestamp(),
		Body:        string(body.data),
		Truncated:   body.truncated,
	}
	if err := LOG_DB.Create(&stored).Error; err != nil {
		var existing RequestCaptureBody
		if findErr := LOG_DB.Where("content_hash = ?", hash).First(&existing).Error; findErr == nil {
			return existing.Id, nil
		}
		return 0, err
	}
	return stored.Id, nil
}

func hydrateRequestCaptureBodies(items []*RequestCapture) error {
	bodyIDs := make(map[int64]struct{})
	for _, capture := range items {
		for _, id := range []int64{
			capture.RequestBodyId,
			capture.UpstreamRequestBodyId,
			capture.ResponseBodyId,
		} {
			if id > 0 {
				bodyIDs[id] = struct{}{}
			}
		}
	}
	if len(bodyIDs) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(bodyIDs))
	for id := range bodyIDs {
		ids = append(ids, id)
	}
	var bodies []RequestCaptureBody
	if err := LOG_DB.Where("id IN ?", ids).Find(&bodies).Error; err != nil {
		return err
	}
	byID := make(map[int64]RequestCaptureBody, len(bodies))
	for _, body := range bodies {
		byID[body.Id] = body
	}
	for _, capture := range items {
		if body, ok := byID[capture.RequestBodyId]; ok {
			capture.RequestBody = body.Body
			capture.RequestTruncated = body.Truncated
		}
		if body, ok := byID[capture.UpstreamRequestBodyId]; ok {
			capture.UpstreamRequestBody = body.Body
			capture.UpstreamTruncated = body.Truncated
		}
		if body, ok := byID[capture.ResponseBodyId]; ok {
			capture.ResponseBody = body.Body
			capture.ResponseTruncated = body.Truncated
		}
	}
	return nil
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
			bodyID, err := storeRequestCaptureBody(body)
			if err != nil {
				common.SysLog(common.LogText("failed to store original request capture body: %s", err.Error()))
				capture.RequestBody = string(body.data)
				capture.RequestTruncated = body.truncated
			} else {
				capture.RequestBodyId = bodyID
			}
		}
	}
	if settings.RequestCaptureUpstream {
		if body, ok := captureBytes(upstreamBody, requestCaptureLimit()); ok {
			bodyID, err := storeRequestCaptureBody(body)
			if err != nil {
				common.SysLog(common.LogText("failed to store upstream request capture body: %s", err.Error()))
				capture.UpstreamRequestBody = string(body.data)
				capture.UpstreamTruncated = body.truncated
			} else {
				capture.UpstreamRequestBodyId = bodyID
			}
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
	capturedBody := requestCaptureBody{data: []byte(body), truncated: truncated}
	bodyID, err := storeRequestCaptureBody(capturedBody)
	updates := map[string]any{
		"response_body_id":      bodyID,
		"response_status":       status,
		"response_content_type": contentType,
		"response_truncated":    truncated,
	}
	if err != nil {
		common.SysLog(common.LogText("failed to store upstream response capture body: %s", err.Error()))
		updates["response_body"] = body
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
	if err != nil {
		return nil, total, err
	}
	if err := hydrateRequestCaptureBodies(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func GetRequestCapture(id int64) (*RequestCapture, error) {
	var capture RequestCapture
	if err := LOG_DB.First(&capture, id).Error; err != nil {
		return nil, err
	}
	if err := hydrateRequestCaptureBodies([]*RequestCapture{&capture}); err != nil {
		return nil, err
	}
	return &capture, nil
}

func requestCaptureBodyIDs(captures []RequestCapture) []int64 {
	seen := make(map[int64]struct{})
	for _, capture := range captures {
		for _, id := range []int64{
			capture.RequestBodyId,
			capture.UpstreamRequestBodyId,
			capture.ResponseBodyId,
		} {
			if id > 0 {
				seen[id] = struct{}{}
			}
		}
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}

func deleteUnusedRequestCaptureBodies(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	var references []RequestCapture
	if err := LOG_DB.WithContext(ctx).Select("request_body_id, upstream_request_body_id, response_body_id").
		Where("request_body_id IN ? OR upstream_request_body_id IN ? OR response_body_id IN ?", ids, ids, ids).
		Find(&references).Error; err != nil {
		return err
	}
	used := make(map[int64]struct{}, len(references)*3)
	for _, capture := range references {
		for _, id := range []int64{capture.RequestBodyId, capture.UpstreamRequestBodyId, capture.ResponseBodyId} {
			if id > 0 {
				used[id] = struct{}{}
			}
		}
	}
	unused := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, exists := used[id]; !exists {
			unused = append(unused, id)
		}
	}
	if len(unused) == 0 {
		return nil
	}
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return LOG_DB.WithContext(ctx).Exec("ALTER TABLE request_capture_bodies DELETE WHERE id IN ? SETTINGS mutations_sync = 1", unused).Error
	}
	return LOG_DB.WithContext(ctx).Where("id IN ?", unused).Delete(&RequestCaptureBody{}).Error
}

func DeleteRequestCapture(id int64) error {
	var capture RequestCapture
	findErr := LOG_DB.First(&capture, id).Error
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return findErr
	}
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		if err := LOG_DB.Exec("ALTER TABLE request_captures DELETE WHERE id = ? SETTINGS mutations_sync = 1", id).Error; err != nil {
			return err
		}
	} else if err := LOG_DB.Delete(&RequestCapture{}, id).Error; err != nil {
		return err
	}
	if findErr != nil {
		return nil
	}
	return deleteUnusedRequestCaptureBodies(context.Background(), requestCaptureBodyIDs([]RequestCapture{capture}))
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
	var captures []RequestCapture
	if err := query.Select("request_body_id, upstream_request_body_id, response_body_id").Find(&captures).Error; err != nil {
		return 0, err
	}
	bodyIDs := requestCaptureBodyIDs(captures)
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		if err := LOG_DB.WithContext(ctx).Exec("ALTER TABLE request_captures DELETE WHERE created_at < ? SETTINGS mutations_sync = 1", targetTimestamp).Error; err != nil {
			return 0, err
		}
	} else {
		result := query.Delete(&RequestCapture{})
		if result.Error != nil {
			return 0, result.Error
		}
		total = result.RowsAffected
	}
	if err := deleteUnusedRequestCaptureBodies(ctx, bodyIDs); err != nil {
		return 0, err
	}
	return total, nil
}

func DeleteAllRequestCaptures(ctx context.Context) (int64, error) {
	var total int64
	query := LOG_DB.WithContext(ctx).Model(&RequestCapture{})
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, nil
	}
	var captures []RequestCapture
	if err := query.Select("request_body_id, upstream_request_body_id, response_body_id").Find(&captures).Error; err != nil {
		return 0, err
	}
	bodyIDs := requestCaptureBodyIDs(captures)
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		if err := LOG_DB.WithContext(ctx).Exec("ALTER TABLE request_captures DELETE WHERE 1 = 1 SETTINGS mutations_sync = 1").Error; err != nil {
			return 0, err
		}
	} else if err := query.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RequestCapture{}).Error; err != nil {
		return 0, err
	}
	if err := deleteUnusedRequestCaptureBodies(ctx, bodyIDs); err != nil {
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
