package controller

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// Channel status grades derived from the request success rate of the
// monitored window. The thresholds are shared with the web console legend.
const (
	channelStatusNormal      = "normal"
	channelStatusLatency     = "latency"
	channelStatusDegraded    = "degraded"
	channelStatusAbnormal    = "abnormal"
	channelStatusUnavailable = "unavailable"
	channelStatusNoTraffic   = "no_traffic"

	// channelStatusHistoryDays is the number of daily buckets kept per channel
	// and model. Historical model detection reuses the same window so the page
	// and the restore action never disagree about what counts as history.
	channelStatusHistoryDays = 30
	channelStatusDaySeconds  = 86400
	channelStatusHourSeconds = 3600
	// channelStatusHoursPerDay caps the hourly cells of the current day.
	channelStatusHoursPerDay = 24
)

type channelStatusDay struct {
	Day            int64    `json:"day"`
	Status         string   `json:"status"`
	Availability   *float64 `json:"availability"`
	Attempts       int64    `json:"attempts"`
	AverageLatency float64  `json:"average_latency_ms"`
}

type channelStatusModel struct {
	Name            string             `json:"name"`
	Enabled         bool               `json:"enabled"`
	Historical      bool               `json:"historical"`
	Status          string             `json:"status"`
	LatencyMS       float64            `json:"latency_ms"`
	Availability7D  *float64           `json:"availability_7d"`
	Availability30D *float64           `json:"availability_30d"`
	Attempts7D      int64              `json:"attempts_7d"`
	Attempts30D     int64              `json:"attempts_30d"`
	LastActivity    int64              `json:"last_activity"`
	History         []channelStatusDay `json:"history"`
}

type channelStatusChannel struct {
	ID              int                  `json:"id"`
	Name            string               `json:"name"`
	Enabled         bool                 `json:"enabled"`
	Status          string               `json:"status"`
	LatencyMS       float64              `json:"latency_ms"`
	Availability7D  *float64             `json:"availability_7d"`
	Availability30D *float64             `json:"availability_30d"`
	Attempts7D      int64                `json:"attempts_7d"`
	LastActivity    int64                `json:"last_activity"`
	History         []channelStatusDay   `json:"history"`
	Models          []channelStatusModel `json:"models"`
}

type channelStatusSummary struct {
	NormalChannels      int `json:"normal_channels"`
	NonNormalChannels   int `json:"non_normal_channels"`
	UnavailableChannels int `json:"unavailable_channels"`
	NoTrafficChannels   int `json:"no_traffic_channels"`
	ChannelCount        int `json:"channel_count"`
	ModelCount          int `json:"model_count"`
	MonitoredModels     int `json:"monitored_models"`
}

type channelStatusResponse struct {
	GeneratedAt int                    `json:"generated_at"`
	WindowStart int64                  `json:"window_start"`
	Summary     channelStatusSummary   `json:"summary"`
	Channels    []channelStatusChannel `json:"channels"`
}

type channelStatusAggregate struct {
	Attempts       int64
	Successes      int64
	LatencySeconds int64
	LastActivity   int64
}

func (aggregate *channelStatusAggregate) add(metric model.ChannelStatusMetric) {
	aggregate.Attempts += metric.Count
	if metric.Type == model.LogTypeConsume {
		aggregate.Successes += metric.Count
	}
	aggregate.LatencySeconds += metric.UseTime
	if metric.LastActivity > aggregate.LastActivity {
		aggregate.LastActivity = metric.LastActivity
	}
}

// availability reports the success rate in percent, or nil when the bucket saw
// no traffic at all.
func (aggregate channelStatusAggregate) availability() *float64 {
	if aggregate.Attempts == 0 {
		return nil
	}
	value := float64(aggregate.Successes) * 100 / float64(aggregate.Attempts)
	return &value
}

func (aggregate channelStatusAggregate) averageLatencyMS() float64 {
	// Both consume and error logs record their elapsed seconds, so the average
	// is taken over every attempt in the window.
	if aggregate.Attempts == 0 {
		return 0
	}
	return float64(aggregate.LatencySeconds) * 1000 / float64(aggregate.Attempts)
}

func channelHealthStatus(availability *float64) string {
	if availability == nil {
		return channelStatusNoTraffic
	}
	switch {
	case *availability >= 95:
		return channelStatusNormal
	case *availability >= 85:
		return channelStatusLatency
	case *availability >= 60:
		return channelStatusDegraded
	case *availability >= 30:
		return channelStatusAbnormal
	default:
		return channelStatusUnavailable
	}
}

func channelStatusHistory(startDay, today int64, metrics map[int64]channelStatusAggregate) []channelStatusDay {
	history := make([]channelStatusDay, 0, channelStatusHistoryDays)
	for day := startDay; day <= today; day += channelStatusDaySeconds {
		aggregate := metrics[day]
		availability := aggregate.availability()
		history = append(history, channelStatusDay{
			Day:            day,
			Status:         channelHealthStatus(availability),
			Availability:   availability,
			Attempts:       aggregate.Attempts,
			AverageLatency: aggregate.averageLatencyMS(),
		})
	}
	return history
}

// GetChannelStatus aggregates the request logs of the monitored window into a
// per-channel and per-model availability view for the admin status page.
func GetChannelStatus(c *gin.Context) {
	now := common.GetTimestamp()
	today := now / channelStatusDaySeconds * channelStatusDaySeconds
	startDay := today - (channelStatusHistoryDays-1)*channelStatusDaySeconds

	metrics, err := model.GetChannelStatusMetrics(startDay, now)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	channels := make([]*model.Channel, 0)
	if err := model.DB.Omit("key").Find(&channels).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	channelIDs := make([]int, 0, len(channels))
	for _, channel := range channels {
		channelIDs = append(channelIDs, channel.Id)
	}
	historicalModels, err := model.GetHistoricalChannelModels(channelIDs, startDay)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	metricsByChannel := make(map[int]map[string][]model.ChannelStatusMetric)
	for _, metric := range metrics {
		if metricsByChannel[metric.ChannelID] == nil {
			metricsByChannel[metric.ChannelID] = make(map[string][]model.ChannelStatusMetric)
		}
		metricsByChannel[metric.ChannelID][metric.ModelName] = append(metricsByChannel[metric.ChannelID][metric.ModelName], metric)
	}

	response := channelStatusResponse{
		GeneratedAt: int(now),
		WindowStart: startDay,
		Channels:    make([]channelStatusChannel, 0, len(channels)),
	}
	sevenDayStart := today - 6*channelStatusDaySeconds

	for _, channel := range channels {
		configuredModels, configuredSet := configuredChannelModels(channel)
		modelNames := append(configuredModels, removedHistoricalModels(historicalModels[channel.Id], configuredSet)...)

		channelMetrics := metricsByChannel[channel.Id]
		channelDays := make(map[int64]channelStatusAggregate)
		channel7D := channelStatusAggregate{}
		channel30D := channelStatusAggregate{}
		for _, modelMetrics := range channelMetrics {
			for _, metric := range modelMetrics {
				channel30D.add(metric)
				if metric.Day >= sevenDayStart {
					channel7D.add(metric)
				}
				day := channelDays[metric.Day]
				day.add(metric)
				channelDays[metric.Day] = day
			}
		}

		status := channelHealthStatus(channel7D.availability())
		switch status {
		case channelStatusNormal:
			response.Summary.NormalChannels++
		case channelStatusNoTraffic:
			response.Summary.NoTrafficChannels++
		default:
			response.Summary.NonNormalChannels++
		}
		if status == channelStatusUnavailable {
			response.Summary.UnavailableChannels++
		}

		channelResult := channelStatusChannel{
			ID:              channel.Id,
			Name:            channel.Name,
			Enabled:         channel.Status == common.ChannelStatusEnabled,
			Status:          status,
			LatencyMS:       channel7D.averageLatencyMS(),
			Availability7D:  channel7D.availability(),
			Availability30D: channel30D.availability(),
			Attempts7D:      channel7D.Attempts,
			LastActivity:    channel30D.LastActivity,
			History:         channelStatusHistory(startDay, today, channelDays),
			Models:          make([]channelStatusModel, 0, len(modelNames)),
		}

		for _, name := range modelNames {
			modelDays := make(map[int64]channelStatusAggregate)
			model7D := channelStatusAggregate{}
			model30D := channelStatusAggregate{}
			for _, metric := range channelMetrics[name] {
				model30D.add(metric)
				if metric.Day >= sevenDayStart {
					model7D.add(metric)
				}
				day := modelDays[metric.Day]
				day.add(metric)
				modelDays[metric.Day] = day
			}
			_, enabled := configuredSet[name]
			_, historical := historicalModels[channel.Id][name]
			if model7D.Attempts > 0 {
				response.Summary.MonitoredModels++
			}
			channelResult.Models = append(channelResult.Models, channelStatusModel{
				Name:            name,
				Enabled:         enabled,
				Historical:      historical,
				Status:          channelHealthStatus(model7D.availability()),
				LatencyMS:       model7D.averageLatencyMS(),
				Availability7D:  model7D.availability(),
				Availability30D: model30D.availability(),
				Attempts7D:      model7D.Attempts,
				Attempts30D:     model30D.Attempts,
				LastActivity:    model30D.LastActivity,
				History:         channelStatusHistory(startDay, today, modelDays),
			})
		}

		response.Summary.ChannelCount++
		response.Summary.ModelCount += len(configuredSet)
		response.Channels = append(response.Channels, channelResult)
	}

	common.ApiSuccess(c, response)
}

// configuredChannelModels returns the trimmed, de-duplicated model list of a
// channel in its configured order.
func configuredChannelModels(channel *model.Channel) ([]string, map[string]struct{}) {
	models := channel.GetModels()
	ordered := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, name := range models {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		ordered = append(ordered, name)
	}
	return ordered, seen
}

// removedHistoricalModels lists models that still have log history but are no
// longer configured on the channel, sorted for a stable response order.
func removedHistoricalModels(historical map[string]struct{}, configured map[string]struct{}) []string {
	removed := make([]string, 0, len(historical))
	for name := range historical {
		if _, exists := configured[name]; !exists {
			removed = append(removed, name)
		}
	}
	slices.Sort(removed)
	return removed
}

type updateChannelModelStatusRequest struct {
	Model  string `json:"model"`
	Enable bool   `json:"enable"`
}

// UpdateChannelModelStatus removes a model from a channel's model list, or
// restores it when the channel has log history for that model. Models that were
// never seen in the logs cannot be restored from this page.
func UpdateChannelModelStatus(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorT(c, "Invalid channel ID")
		return
	}
	var request updateChannelModelStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	request.Model = strings.TrimSpace(request.Model)
	if request.Model == "" {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}

	// The key is not needed to edit the model list, so it is not loaded.
	channel, err := model.GetChannelById(channelID, false)
	if err != nil {
		common.ApiErrorT(c, "Channel does not exist")
		return
	}

	models := channel.GetModels()
	index := -1
	for i, name := range models {
		if strings.TrimSpace(name) == request.Model {
			index = i
			break
		}
	}
	if request.Enable {
		if index < 0 {
			now := common.GetTimestamp()
			windowStart := now/channelStatusDaySeconds*channelStatusDaySeconds - (channelStatusHistoryDays-1)*channelStatusDaySeconds
			historical, err := model.HasHistoricalChannelModel(channelID, request.Model, windowStart)
			if err != nil {
				common.ApiError(c, err)
				return
			}
			if !historical {
				common.ApiErrorT(c, "Only models with request history can be restored")
				return
			}
			models = append(models, request.Model)
		}
	} else if index >= 0 {
		models = append(models[:index], models[index+1:]...)
	}

	channel.Models = strings.Join(models, ",")
	if err := channel.UpdateModels(); err != nil {
		common.ApiError(c, err)
		return
	}
	model.InitChannelCache()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"channel_id": channelID,
			"model":      request.Model,
			"enabled":    request.Enable,
		},
	})
}
