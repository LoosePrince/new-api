package controller

import (
	"cmp"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

// The public status page reports the same availability grades as the admin
// channel status page, but it is read-only and hides everything an anonymous
// visitor must not see: channel identifiers, channel keys, configured model
// lists and per-channel request volume.

// publicStatusDay is one completed day of the history strip.
type publicStatusDay struct {
	Day          int64    `json:"day"`
	Status       string   `json:"status"`
	Availability *float64 `json:"availability"`
}

// publicStatusHour is one elapsed hour of the current day. The status strip
// reports today hour by hour, because a daily bucket cannot show what happened
// in the last few hours.
type publicStatusHour struct {
	Hour         int64    `json:"hour"`
	Status       string   `json:"status"`
	Availability *float64 `json:"availability"`
}

type publicStatusChannel struct {
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	LatencyMS      float64  `json:"latency_ms"`
	Availability7D *float64 `json:"availability_7d"`
}

type publicStatusModel struct {
	Name            string                `json:"name"`
	Status          string                `json:"status"`
	LatencyMS       float64               `json:"latency_ms"`
	Availability7D  *float64              `json:"availability_7d"`
	Availability30D *float64              `json:"availability_30d"`
	Attempts        int64                 `json:"attempts"`
	History         []publicStatusDay     `json:"history"`
	Today           []publicStatusHour    `json:"today"`
	Channels        []publicStatusChannel `json:"channels"`
}

type publicStatusSummary struct {
	Status          string             `json:"status"`
	LatencyMS       float64            `json:"latency_ms"`
	Availability7D  *float64           `json:"availability_7d"`
	Availability30D *float64           `json:"availability_30d"`
	Attempts        int64              `json:"attempts"`
	ChannelCount    int                `json:"channel_count"`
	ModelCount      int                `json:"model_count"`
	History         []publicStatusDay  `json:"history"`
	Today           []publicStatusHour `json:"today"`
}

type publicChannelStatusResponse struct {
	Enabled     bool  `json:"enabled"`
	GeneratedAt int64 `json:"generated_at"`
	WindowStart int64 `json:"window_start"`
	// HomeDetailed asks the home page for the full summary card instead of the
	// compact bar. PageEnabled and ModelsEnabled travel with it so the console
	// never needs a second request to learn the published layout.
	HomeDetailed  bool `json:"home_detailed"`
	PageEnabled   bool `json:"page_enabled"`
	ModelsEnabled bool `json:"models_enabled"`
	// ChannelsVisible reports whether channel names are published at all, so the
	// console can drop the channel figure instead of showing a count of zero.
	ChannelsVisible bool                `json:"channels_visible"`
	Summary         publicStatusSummary `json:"summary"`
	Models          []publicStatusModel `json:"models"`
}

// publicStatusModelAggregate collects the metrics of one model across every
// enabled channel that served it.
type publicStatusModelAggregate struct {
	name     string
	channel  map[string]*channelStatusAggregate
	days     map[int64]*channelStatusAggregate
	hours    map[int64]*channelStatusAggregate
	total30D channelStatusAggregate
	total7D  channelStatusAggregate
}

// GetPublicChannelStatus serves the anonymous status page. It returns
// `enabled: false` while the operator has the page switched off, so the web
// console can hide both the page and its home page section without a second
// request.
func GetPublicChannelStatus(c *gin.Context) {
	settings := setting.GetPublicHomeSettings()
	if !settings.ChannelStatusEnabled {
		c.JSON(200, gin.H{
			"success": true,
			"message": "",
			"data":    gin.H{"enabled": false},
		})
		return
	}

	now := common.GetTimestamp()
	today := now / channelStatusDaySeconds * channelStatusDaySeconds
	lastCompletedDay := today - channelStatusDaySeconds
	// The strip holds the 29 completed days plus today, so the daily window is
	// still 30 days wide while today is reported hour by hour below.
	startDay := today - (channelStatusHistoryDays-1)*channelStatusDaySeconds
	sevenDayStart := today - 6*channelStatusDaySeconds

	dailyMetrics, err := model.GetChannelStatusMetrics(startDay, now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	hourlyMetrics, err := model.GetChannelStatusHourlyMetrics(today, now)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	channels := make([]*model.Channel, 0)
	if err := model.DB.Omit("key").Find(&channels).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	// Only channels that are currently enabled describe the health of the site;
	// a disabled channel is intentionally out of rotation.
	enabledChannels := make(map[int]*model.Channel, len(channels))
	for _, channel := range channels {
		if channel.Status == common.ChannelStatusEnabled {
			enabledChannels[channel.Id] = channel
		}
	}
	// The operator can publish every served channel, none of them, or a
	// hand-picked list. The selection only drops channel names from the report;
	// the site and model aggregates keep counting the same attempts either way,
	// so hiding channels never makes the headline figures disagree with facts.
	shownChannels := filterPublicStatusChannels(enabledChannels, settings)

	aggregates := make(map[string]*publicStatusModelAggregate)
	site30D := channelStatusAggregate{}
	site7D := channelStatusAggregate{}
	siteDays := make(map[int64]*channelStatusAggregate)
	siteHours := make(map[int64]*channelStatusAggregate)
	servedChannels := make(map[int]struct{}, len(channels))
	servedModels := make(map[string]struct{}, len(aggregates))

	for _, metric := range dailyMetrics {
		channel, ok := enabledChannels[metric.ChannelID]
		if !ok || metric.ModelName == "" {
			continue
		}
		aggregate := publicStatusAggregateOf(aggregates, metric.ModelName)

		if _, shown := shownChannels[metric.ChannelID]; shown {
			channelName := publicChannelName(channel)
			channelAggregate := aggregate.channel[channelName]
			if channelAggregate == nil {
				channelAggregate = &channelStatusAggregate{}
				aggregate.channel[channelName] = channelAggregate
			}
			channelAggregate.add(metric)
			// The header count follows the channels whose names the visitor can
			// actually see, so a hidden channel is never counted behind its back.
			servedChannels[metric.ChannelID] = struct{}{}
		}

		aggregate.total30D.add(metric)
		if metric.Day >= sevenDayStart {
			aggregate.total7D.add(metric)
		}
		addPublicStatusBucket(aggregate.days, metric.Day, metric)

		site30D.add(metric)
		if metric.Day >= sevenDayStart {
			site7D.add(metric)
		}
		addPublicStatusBucket(siteDays, metric.Day, metric)

		servedModels[metric.ModelName] = struct{}{}
	}

	// Today repeats the same traffic in hourly buckets, so it only feeds the
	// hourly cells and never the 7- or 30-day totals counted above.
	for _, metric := range hourlyMetrics {
		if _, ok := enabledChannels[metric.ChannelID]; !ok || metric.ModelName == "" {
			continue
		}
		aggregate := publicStatusAggregateOf(aggregates, metric.ModelName)
		addPublicStatusBucket(aggregate.hours, metric.Day, metric)
		addPublicStatusBucket(siteHours, metric.Day, metric)
	}

	// Models with traffic lead the list, so the page opens on what the site
	// actually serves; the rest stay reachable below in a stable order.
	names := make([]string, 0, len(aggregates))
	for name := range aggregates {
		names = append(names, name)
	}
	slices.Sort(names)
	slices.SortStableFunc(names, func(a, b string) int {
		return cmp.Compare(aggregates[b].total7D.Attempts, aggregates[a].total7D.Attempts)
	})

	response := publicChannelStatusResponse{
		Enabled:         true,
		GeneratedAt:     now,
		WindowStart:     startDay,
		HomeDetailed:    settings.ChannelStatusHomeDetailed(),
		PageEnabled:     settings.ChannelStatusPageEnabled(),
		ModelsEnabled:   settings.ChannelStatusModelsEnabled,
		ChannelsVisible: settings.ChannelStatusChannelMode != setting.PublicChannelStatusChannelModeNone,
		Summary: publicStatusSummary{
			Status:          channelHealthStatus(site7D.availability()),
			LatencyMS:       site7D.averageLatencyMS(),
			Availability7D:  site7D.availability(),
			Availability30D: site30D.availability(),
			Attempts:        site7D.Attempts,
			ChannelCount:    len(servedChannels),
			ModelCount:      len(servedModels),
			History:         publicStatusHistory(startDay, lastCompletedDay, siteDays),
			Today:           publicStatusToday(today, now, siteHours),
		},
		Models: make([]publicStatusModel, 0, len(names)),
	}

	// The home page only renders the whole-site bar, and an unpublished model
	// section exposes nothing either, so both stop before the per-model loop.
	if c.Query("summary") == "1" || !settings.ChannelStatusModelsEnabled {
		c.JSON(200, gin.H{
			"success": true,
			"message": "",
			"data":    response,
		})
		return
	}

	for _, name := range names {
		aggregate := aggregates[name]
		channelNames := make([]string, 0, len(aggregate.channel))
		for channelName := range aggregate.channel {
			channelNames = append(channelNames, channelName)
		}
		slices.Sort(channelNames)

		modelChannels := make([]publicStatusChannel, 0, len(channelNames))
		for _, channelName := range channelNames {
			channelAggregate := aggregate.channel[channelName]
			modelChannels = append(modelChannels, publicStatusChannel{
				Name:           channelName,
				Status:         channelHealthStatus(channelAggregate.availability()),
				LatencyMS:      channelAggregate.averageLatencyMS(),
				Availability7D: channelAggregate.availability(),
			})
		}

		response.Models = append(response.Models, publicStatusModel{
			Name:            name,
			Status:          channelHealthStatus(aggregate.total7D.availability()),
			LatencyMS:       aggregate.total7D.averageLatencyMS(),
			Availability7D:  aggregate.total7D.availability(),
			Availability30D: aggregate.total30D.availability(),
			Attempts:        aggregate.total7D.Attempts,
			History:         publicStatusHistory(startDay, lastCompletedDay, aggregate.days),
			Today:           publicStatusToday(today, now, aggregate.hours),
			Channels:        modelChannels,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    response,
	})
}

// filterPublicStatusChannels resolves the configured channel visibility into the
// set of channel ids whose names may appear in the public report.
func filterPublicStatusChannels(enabled map[int]*model.Channel, settings setting.PublicHomeSettings) map[int]struct{} {
	switch settings.ChannelStatusChannelMode {
	case setting.PublicChannelStatusChannelModeNone:
		return map[int]struct{}{}
	case setting.PublicChannelStatusChannelModeSelected:
		selected := make(map[int]struct{}, len(settings.ChannelStatusChannelIDs))
		for _, id := range settings.ChannelStatusChannelIDs {
			if _, ok := enabled[id]; ok {
				selected[id] = struct{}{}
			}
		}
		return selected
	default:
		all := make(map[int]struct{}, len(enabled))
		for id := range enabled {
			all[id] = struct{}{}
		}
		return all
	}
}

// publicChannelName returns the display name of a channel, falling back to a
// neutral label so an unnamed channel never exposes its numeric identifier.
func publicChannelName(channel *model.Channel) string {
	name := strings.TrimSpace(channel.Name)
	if name == "" {
		return "channel"
	}
	return name
}

// publicStatusAggregateOf returns the aggregate of a model, creating it on
// first sight so a model only counts once it has traffic.
func publicStatusAggregateOf(aggregates map[string]*publicStatusModelAggregate, name string) *publicStatusModelAggregate {
	aggregate := aggregates[name]
	if aggregate == nil {
		aggregate = &publicStatusModelAggregate{
			name:    name,
			channel: make(map[string]*channelStatusAggregate),
			days:    make(map[int64]*channelStatusAggregate),
			hours:   make(map[int64]*channelStatusAggregate),
		}
		aggregates[name] = aggregate
	}
	return aggregate
}

// addPublicStatusBucket accumulates one metric into a single time bucket,
// creating the bucket on first sight.
func addPublicStatusBucket(buckets map[int64]*channelStatusAggregate, key int64, metric model.ChannelStatusMetric) {
	bucket := buckets[key]
	if bucket == nil {
		bucket = &channelStatusAggregate{}
		buckets[key] = bucket
	}
	bucket.add(metric)
}

// publicStatusHistory builds the daily cells of the public strip: one cell per
// completed day of the window, oldest first.
func publicStatusHistory(startDay int64, lastDay int64, metrics map[int64]*channelStatusAggregate) []publicStatusDay {
	history := make([]publicStatusDay, 0, channelStatusHistoryDays)
	for day := startDay; day <= lastDay; day += channelStatusDaySeconds {
		aggregate := metrics[day]
		if aggregate == nil {
			aggregate = &channelStatusAggregate{}
		}
		availability := aggregate.availability()
		history = append(history, publicStatusDay{
			Day:          day,
			Status:       channelHealthStatus(availability),
			Availability: availability,
		})
	}
	return history
}

// publicStatusToday builds the hourly cells of the current day: one cell per
// hour that has already begun, so the strip grows through the day and stops at
// a full day.
func publicStatusToday(today int64, now int64, metrics map[int64]*channelStatusAggregate) []publicStatusHour {
	elapsedHours := (now-today)/channelStatusHourSeconds + 1
	if elapsedHours < 1 {
		elapsedHours = 1
	}
	if elapsedHours > channelStatusHoursPerDay {
		elapsedHours = channelStatusHoursPerDay
	}

	hours := make([]publicStatusHour, 0, elapsedHours)
	for index := int64(0); index < elapsedHours; index++ {
		hour := today + index*channelStatusHourSeconds
		aggregate := metrics[hour]
		if aggregate == nil {
			aggregate = &channelStatusAggregate{}
		}
		availability := aggregate.availability()
		hours = append(hours, publicStatusHour{
			Hour:         hour,
			Status:       channelHealthStatus(availability),
			Availability: availability,
		})
	}
	return hours
}
