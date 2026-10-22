package service

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

var noModelRoundRobin = struct {
	sync.Mutex
	indexes map[string]int
}{indexes: make(map[string]int)}

func ResolveGroupRequestModel(c *gin.Context, group, requested string) (string, error) {
	policy := ratio_setting.GetGroupPolicy(group)
	if requested != "" && !policy.IgnoreModel {
		return requested, nil
	}
	if requested == "" && !policy.AllowNoModel {
		return "", errors.New("model is required")
	}
	models := model.GetGroupEnabledModels(group)
	if len(models) == 0 {
		return "", errors.New("no models are available for this group")
	}
	if policy.NoModelStrategy == ratio_setting.NoModelSpecified {
		if policy.NoModel == "" {
			return "", errors.New("no model is configured for this group")
		}
		return policy.NoModel, nil
	}
	models = orderGroupFallbackModels(models, requested, policy.NoModelStrategy)
	if policy.NoModelStrategy == ratio_setting.AutoRouteRandom {
		return models[0], nil
	}
	if policy.NoModelStrategy == ratio_setting.AutoRouteSequential {
		noModelRoundRobin.Lock()
		index := noModelRoundRobin.indexes[group] % len(models)
		noModelRoundRobin.indexes[group] = index + 1
		noModelRoundRobin.Unlock()
		return models[index], nil
	}
	return models[0], nil
}

func directSelectionFilters(c *gin.Context) []dto.ChannelFilter {
	filters := append([]dto.ChannelFilter(nil), GetChannelConstraints(c).Filters...)
	for _, filter := range filters {
		if filter.Kind == dto.FilterDirectSelection {
			return filters
		}
	}
	return append(filters, dto.ChannelFilter{Kind: dto.FilterDirectSelection})
}

func fallbackSelectionFilters(c *gin.Context) []dto.ChannelFilter {
	filters := make([]dto.ChannelFilter, 0, len(GetChannelConstraints(c).Filters))
	for _, filter := range GetChannelConstraints(c).Filters {
		if filter.Kind != dto.FilterDirectSelection {
			filters = append(filters, filter)
		}
	}
	return filters
}

func GetChannelConstraints(c *gin.Context) *dto.ChannelConstraints {
	if c == nil {
		return &dto.ChannelConstraints{}
	}
	if existing, ok := common.GetContextKeyType[*dto.ChannelConstraints](c, constant.ContextKeyChannelConstraints); ok && existing != nil {
		return existing
	}
	constraints := &dto.ChannelConstraints{}
	common.SetContextKey(c, constant.ContextKeyChannelConstraints, constraints)
	return constraints
}

func AppendTaskPluginIdentityFilter(c *gin.Context, pluginKey string) {
	if c == nil {
		return
	}
	channelTypes, pluginKeys := pinnedTaskPluginIdentities(c, pluginKey)
	GetChannelConstraints(c).AddFilter(dto.ChannelFilter{
		Kind:                   dto.FilterTaskPluginIdentity,
		TaskPluginKey:          pluginKey,
		TaskPluginChannelTypes: channelTypes,
		TaskPluginKeys:         pluginKeys,
	})
}

type RetryParam struct {
	Ctx            *gin.Context
	TokenGroup     string
	ModelName      string
	RequestPath    string
	Retry          *int
	resetNextTry   bool
	fallbackCount  int
	fallbackModels map[string][]string
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

// AdvanceFallback moves the next attempt to the next model fallback slot.
// The global retry counter remains the source of truth for the request budget.
func (p *RetryParam) AdvanceFallback() {
	p.fallbackCount++
}

func (p *RetryParam) fallbackIndex() int {
	if p.fallbackCount == 0 {
		return -1
	}
	return p.fallbackCount - 1
}

func ModelFallbackRetryBudget(c *gin.Context, retry *RetryParam) int {
	if retry == nil || !retryFallbackEnabled(c, retry) {
		return 0
	}
	budget := 0
	for _, group := range requestFallbackGroups(c, retry) {
		policy := ratio_setting.GetGroupPolicy(group)
		if policy.FallbackTimes != nil {
			budget = max(budget, *policy.FallbackTimes)
			if policy.FallbackGroup != "" {
				budget = max(budget, *policy.FallbackTimes+1)
			}
		}
	}
	return budget
}

func ShouldRetryForModelFallback(c *gin.Context, retry *RetryParam, err *types.NewAPIError) bool {
	if err == nil || types.IsSkipRetryError(err) || GetChannelConstraints(c).SuppressesRetry() || ShouldSkipRetryAfterChannelAffinityFailure(c) {
		return false
	}
	if err.StatusCode >= 200 && err.StatusCode < 300 {
		return false
	}
	if !retryFallbackEnabled(c, retry) {
		return false
	}
	for _, group := range requestFallbackGroups(c, retry) {
		policy := ratio_setting.GetGroupPolicy(group)
		if policy.FallbackTimes == nil {
			continue
		}
		budget := *policy.FallbackTimes
		if policy.FallbackGroup != "" {
			budget++
		}
		if retry.fallbackCount < budget {
			return true
		}
	}
	return false
}

func retryFallbackEnabled(c *gin.Context, retry *RetryParam) bool {
	if retry == nil || GetChannelConstraints(c).SuppressesRetry() {
		return false
	}
	for _, group := range requestFallbackGroups(c, retry) {
		if ratio_setting.GetGroupPolicy(group).AutoRoute {
			return true
		}
	}
	return false
}

func requestFallbackGroups(c *gin.Context, retry *RetryParam) []string {
	if retry.TokenGroup == "auto" {
		groups := GetRequestAutoGroups(c, common.GetContextKeyString(c, constant.ContextKeyUserGroup))
		if len(groups) > 0 {
			return groups
		}
	}
	return []string{retry.TokenGroup}
}

// CacheGetRandomSatisfiedChannel tries to get a random channel that satisfies the requirements.
// 尝试获取一个满足要求的随机渠道。
//
// For "auto" tokenGroup with cross-group Retry enabled:
// 对于启用了跨分组重试的 "auto" tokenGroup：
//
//   - Each group will exhaust all its priorities before moving to the next group.
//     每个分组会用完所有优先级后才会切换到下一个分组。
//
//   - Uses ContextKeyAutoGroupIndex to track current group index.
//     使用 ContextKeyAutoGroupIndex 跟踪当前分组索引。
//
//   - Uses ContextKeyAutoGroupRetryIndex to track the global Retry count when current group started.
//     使用 ContextKeyAutoGroupRetryIndex 跟踪当前分组开始时的全局重试次数。
//
//   - priorityRetry = Retry - startRetryIndex, represents the priority level within current group.
//     priorityRetry = Retry - startRetryIndex，表示当前分组内的优先级级别。
//
//   - When GetRandomSatisfiedChannel returns nil (priorities exhausted), moves to next group.
//     当 GetRandomSatisfiedChannel 返回 nil（优先级用完）时，切换到下一个分组。
//
// Example flow (2 groups, each with 2 priorities, RetryTimes=3):
// 示例流程（2个分组，每个有2个优先级，RetryTimes=3）：
//
//	Retry=0: GroupA, priority0 (startRetryIndex=0, priorityRetry=0)
//	         分组A, 优先级0
//
//	Retry=1: GroupA, priority1 (startRetryIndex=0, priorityRetry=1)
//	         分组A, 优先级1
//
//	Retry=2: GroupA exhausted → GroupB, priority0 (startRetryIndex=2, priorityRetry=0)
//	         分组A用完 → 分组B, 优先级0
//
//	Retry=3: GroupB, priority1 (startRetryIndex=2, priorityRetry=1)
//	         分组B, 优先级1
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	common.SetContextKey(param.Ctx, constant.ContextKeyRoutedModel, "")
	var channel *model.Channel
	var err error
	selectGroup := param.TokenGroup
	userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)
	filters := directSelectionFilters(param.Ctx)

	if param.TokenGroup == "auto" {
		autoGroups := GetRequestAutoGroups(param.Ctx, userGroup)
		if len(autoGroups) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}

		// startGroupIndex: the group index to start searching from
		// startGroupIndex: 开始搜索的分组索引
		startGroupIndex := 0
		crossGroupRetry := common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry)

		if lastGroupIndex, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
			if idx, ok := lastGroupIndex.(int); ok {
				startGroupIndex = idx
			}
		}

		for i := startGroupIndex; i < len(autoGroups); i++ {
			autoGroup := autoGroups[i]
			// Calculate priorityRetry for current group
			// 计算当前分组的 priorityRetry
			priorityRetry := param.GetRetry()
			// If moved to a new group, reset priorityRetry and update startRetryIndex
			// 如果切换到新分组，重置 priorityRetry 并更新 startRetryIndex
			if i > startGroupIndex {
				priorityRetry = 0
			}
			logger.LogDebug(param.Ctx, "Auto selecting group: %s, priorityRetry: %d", autoGroup, priorityRetry)

			channel, _ = model.GetRandomSatisfiedChannel(
				autoGroup,
				param.ModelName,
				priorityRetry,
				filters,
			)
			if channel == nil {
				// Current group has no available channel for this model, try next group
				// 当前分组没有该模型的可用渠道，尝试下一个分组
				logger.LogDebug(param.Ctx, "No available channel in group %s for model %s at priorityRetry %d, trying next group", autoGroup, param.ModelName, priorityRetry)
				// 重置状态以尝试下一个分组
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupRetryIndex, 0)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				continue
			}
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, autoGroup)
			selectGroup = autoGroup
			logger.LogDebug(param.Ctx, "Auto selected group: %s", autoGroup)

			// Prepare state for next retry
			// 为下一次重试准备状态
			if crossGroupRetry && priorityRetry >= common.RetryTimes {
				// Current group has exhausted all retries, prepare to switch to next group
				// This request still uses current group, but next retry will use next group
				// 当前分组已用完所有重试次数，准备切换到下一个分组
				// 本次请求仍使用当前分组，但下次重试将使用下一个分组
				logger.LogDebug(param.Ctx, "Current group %s retries exhausted (priorityRetry=%d >= RetryTimes=%d), preparing switch to next group for next retry", autoGroup, priorityRetry, common.RetryTimes)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				param.ResetRetryNextTry()
			} else {
				// Stay in current group, save current state
				// 保持在当前分组，保存当前状态
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i)
			}
			break
		}
	} else {
		channel, err = model.GetRandomSatisfiedChannel(
			param.TokenGroup,
			param.ModelName,
			param.GetRetry(),
			filters,
		)
		if err != nil {
			return nil, param.TokenGroup, err
		}
	}
	return channel, selectGroup, nil
}

func pinnedTaskPluginIdentities(c *gin.Context, expected string) ([]int, []string) {
	if c == nil || expected == "" {
		return nil, nil
	}
	if value, exists := c.Get(jsplugin.ContextKeyPinnedEndpoint); exists {
		pinned, ok := value.(jsplugin.PinnedEndpoint)
		if ok && pinned.Generation != nil && len(pinned.Candidates) > 1 {
			expectedFound := false
			channelTypes := make([]int, 0, len(pinned.Candidates))
			pluginKeys := make([]string, 0, len(pinned.Candidates))
			seen := make(map[int]struct{}, len(pinned.Candidates))
			for _, candidate := range pinned.Candidates {
				if candidate.Plugin == nil {
					continue
				}
				if candidate.Plugin.Meta.Key == expected {
					expectedFound = true
				}
				pluginKeys = append(pluginKeys, candidate.Plugin.Meta.Key)
				for _, channelType := range candidate.Plugin.Meta.ChannelTypes {
					if channelType == 0 || channelType == constant.ChannelTypeTaskPlugin {
						continue
					}
					if _, duplicate := seen[channelType]; duplicate {
						continue
					}
					if plugin, indexed := pinned.Generation.GetByChannelType(channelType); indexed && plugin == candidate.Plugin {
						seen[channelType] = struct{}{}
						channelTypes = append(channelTypes, channelType)
					}
				}
			}
			if expectedFound {
				return channelTypes, pluginKeys
			}
		}
	}
	value, exists := c.Get(jsplugin.ContextKeyPinnedPlugin)
	pinned, ok := value.(jsplugin.PinnedPlugin)
	if !exists || !ok || pinned.Generation == nil || pinned.Plugin == nil || pinned.Plugin.Meta.Key != expected {
		return nil, nil
	}
	channelTypes := make([]int, 0, len(pinned.Plugin.Meta.ChannelTypes))
	for _, channelType := range pinned.Plugin.Meta.ChannelTypes {
		if channelType == 0 || channelType == constant.ChannelTypeTaskPlugin {
			continue
		}
		channelTypes = append(channelTypes, channelType)
	}
	return channelTypes, []string{expected}
}

// ChannelSelectError explains why SelectChannelForRequest found no channel.
// Callers render it for their transport: the HTTP distributor localizes
// MessageID with its own helpers and the Responses WebSocket relay wraps it in
// a NewAPIError. Message is set instead of MessageID when the text is a fixed
// error code that clients match on.
type ChannelSelectError struct {
	StatusCode int
	Code       types.ErrorCode
	MessageID  string
	Params     map[string]any
	Message    string
	// FilterKind and Channel identify a candidate rejected by request filters.
	FilterKind dto.ChannelFilterKind
	Channel    *model.Channel
	// NoAvailableChannel marks the "no channel for this group and model"
	// outcome so the distributor can name the claiming task plugin.
	NoAvailableChannel bool
}

// SelectChannelForRequest resolves the channel for one attempt with the rules
// shared by the HTTP distributor and the Responses WebSocket relay: a pinned
// channel wins, then session affinity (first attempt only), then a random
// eligible channel; every candidate must satisfy the request's channel
// filters. A configured group fallback runs only after the requested model has
// no eligible channel. The selected group is returned for auto-group callers;
// the caller still applies SetupContextForSelectedChannel.
func SelectChannelForRequest(c *gin.Context, modelName string, retry *RetryParam) (*model.Channel, string, *ChannelSelectError) {
	common.SetContextKey(c, constant.ContextKeyRoutedModel, "")
	constraints := GetChannelConstraints(c)
	selectionFilters := directSelectionFilters(c)
	if pin, found, overridden := constraints.ResolvedPin(); found {
		for _, lost := range overridden {
			logger.LogWarn(c, common.LogText(
				"channel pin overridden: winning_source=%s winning_channel_id=%d overridden_source=%s overridden_channel_id=%d",
				pin.Source, pin.ChannelId, lost.Source, lost.ChannelId,
			))
		}
		channel, err := model.CacheGetChannel(pin.ChannelId)
		if err != nil {
			return nil, "", pinnedChannelUnavailable(pin, http.StatusBadRequest, i18n.MsgDistributorInvalidChannelId)
		}
		if channel.Status != common.ChannelStatusEnabled {
			return nil, "", pinnedChannelUnavailable(pin, http.StatusForbidden, i18n.MsgDistributorChannelDisabled)
		}
		if ok, kind := model.ChannelSatisfiesFilters(channel, modelName, selectionFilters); !ok {
			return nil, "", &ChannelSelectError{
				StatusCode: http.StatusBadRequest, Code: types.ErrorCode(kind), MessageID: i18n.MsgDistributorNoAvailableChannel,
				Params:     map[string]any{"Group": common.GetContextKeyString(c, constant.ContextKeyUsingGroup), "Model": modelName},
				FilterKind: kind, Channel: channel,
			}
		}
		return channel, "", nil
	}

	usingGroup := retry.TokenGroup
	var channel *model.Channel
	var selectGroup string
	directSelection := true
	if retry.fallbackCount > 0 {
		if fallbackChannel, fallbackGroup, fallbackModel := selectGroupFallback(c, retry, modelName); fallbackChannel != nil {
			common.SetContextKey(c, constant.ContextKeyRoutedModel, fallbackModel)
			common.SetContextKey(c, constant.ContextKeyAutoGroup, fallbackGroup)
			return fallbackChannel, fallbackGroup, nil
		}
	}
	if retry.GetRetry() == 0 {
		if preferredChannelID, found := GetPreferredChannelByAffinity(c, modelName, usingGroup); found {
			affinityUsable := false
			preferred, err := model.CacheGetChannel(preferredChannelID)
			affinitySatisfied := false
			if err == nil && preferred != nil && preferred.Status == common.ChannelStatusEnabled {
				affinitySatisfied, _ = model.ChannelSatisfiesFilters(preferred, modelName, selectionFilters)
			}
			if affinitySatisfied {
				if usingGroup == "auto" {
					userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
					for _, g := range GetRequestAutoGroups(c, userGroup) {
						if model.IsChannelEnabledForGroupModel(g, modelName, preferred.Id) {
							selectGroup = g
							common.SetContextKey(c, constant.ContextKeyAutoGroup, g)
							channel = preferred
							affinityUsable = true
							MarkChannelAffinityUsed(c, g, preferred.Id)
							break
						}
					}
				} else if model.IsChannelEnabledForGroupModel(usingGroup, modelName, preferred.Id) {
					channel = preferred
					selectGroup = usingGroup
					affinityUsable = true
					MarkChannelAffinityUsed(c, usingGroup, preferred.Id)
				}
			}
			if !affinityUsable && !ShouldKeepChannelAffinityOnChannelDisabled() {
				ClearCurrentChannelAffinityCache(c)
			}
			if !affinityUsable && RequestPolicy(c).SessionMode == "strict" {
				return nil, "", &ChannelSelectError{StatusCode: http.StatusServiceUnavailable, Message: "strict_session_binding_unavailable"}
			}
		}
	}

	if channel == nil {
		var err error
		channel, selectGroup, err = CacheGetRandomSatisfiedChannel(retry)
		if err != nil {
			showGroup := usingGroup
			if usingGroup == "auto" {
				showGroup = fmt.Sprintf("auto(%s)", selectGroup)
			}
			return nil, selectGroup, &ChannelSelectError{
				StatusCode: http.StatusServiceUnavailable, Code: types.ErrorCodeModelNotFound, MessageID: i18n.MsgDistributorGetChannelFailed,
				Params: map[string]any{"Group": showGroup, "Model": modelName, "Error": err.Error()},
			}
		}
		if channel == nil {
			if fallbackChannel, fallbackGroup, fallbackModel := selectGroupFallback(c, retry, modelName); fallbackChannel != nil {
				directSelection = false
				channel = fallbackChannel
				selectGroup = fallbackGroup
				common.SetContextKey(c, constant.ContextKeyRoutedModel, fallbackModel)
				common.SetContextKey(c, constant.ContextKeyAutoGroup, fallbackGroup)
			}
			if channel == nil {
				return nil, selectGroup, &ChannelSelectError{
					StatusCode: http.StatusServiceUnavailable, Code: types.ErrorCodeModelNotFound, MessageID: i18n.MsgDistributorNoAvailableChannel,
					Params: map[string]any{"Group": usingGroup, "Model": modelName}, NoAvailableChannel: true,
				}
			}
		}
	}
	effectiveModel := common.GetContextKeyString(c, constant.ContextKeyRoutedModel)
	if effectiveModel != "" {
		modelName = effectiveModel
	}
	finalFilters := selectionFilters
	if !directSelection {
		finalFilters = GetChannelConstraints(c).Filters
	}
	if ok, kind := model.ChannelSatisfiesFilters(channel, modelName, finalFilters); !ok {
		return nil, selectGroup, &ChannelSelectError{
			StatusCode: http.StatusServiceUnavailable, Code: types.ErrorCodeModelNotFound, MessageID: i18n.MsgDistributorNoAvailableChannel,
			Params:     map[string]any{"Group": common.GetContextKeyString(c, constant.ContextKeyUsingGroup), "Model": modelName},
			FilterKind: kind, Channel: channel, NoAvailableChannel: true,
		}
	}
	return channel, selectGroup, nil
}

func selectGroupFallback(c *gin.Context, retry *RetryParam, requestedModel string) (*model.Channel, string, string) {
	if retry == nil || retry.TokenGroup == "" {
		return nil, "", ""
	}

	groups := []string{retry.TokenGroup}
	if retry.TokenGroup == "auto" {
		groups = GetRequestAutoGroups(c, common.GetContextKeyString(c, constant.ContextKeyUserGroup))
	}
	for _, group := range groups {
		policy := ratio_setting.GetGroupPolicy(group)
		if !policy.AutoRoute {
			continue
		}
		groupRetry := *retry
		groupRetry.TokenGroup = group
		channel, fallbackGroup, fallbackModel := selectGroupFallbackForGroup(c, &groupRetry, group, requestedModel, policy)
		if channel != nil {
			return channel, fallbackGroup, fallbackModel
		}
	}
	return nil, "", ""
}

func selectGroupFallbackForGroup(c *gin.Context, retry *RetryParam, group, requestedModel string, policy ratio_setting.GroupPolicy) (*model.Channel, string, string) {
	filters := fallbackSelectionFilters(c)
	candidates := model.GetGroupEnabledModels(group)
	fallbackTimes := -1
	if policy.FallbackTimes != nil {
		fallbackTimes = *policy.FallbackTimes
	}
	modelLimitEnabled := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
	var modelLimit map[string]bool
	if modelLimitEnabled {
		value, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
		modelLimit, _ = value.(map[string]bool)
		if !ok || modelLimit == nil {
			return nil, "", ""
		}
	}
	filtered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == requestedModel || (modelLimitEnabled && !groupFallbackModelAllowed(modelLimit, candidate)) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	if retry.fallbackModels == nil {
		retry.fallbackModels = make(map[string][]string)
	}
	candidates, exists := retry.fallbackModels[group]
	if !exists {
		candidates = orderGroupFallbackModels(filtered, requestedModel, policy.RouteStrategy)
		retry.fallbackModels[group] = candidates
	}
	if retry.fallbackCount == 0 {
		if len(candidates) == 0 || fallbackTimes == 0 {
			return selectConfiguredFallback(c, retry, requestedModel, policy)
		}
		if fallbackTimes > 0 && len(candidates) > fallbackTimes {
			candidates = candidates[:fallbackTimes]
		}
	} else {
		fallbackIndex := retry.fallbackIndex()
		if (fallbackTimes >= 0 && fallbackIndex >= fallbackTimes) || fallbackIndex >= len(candidates) {
			return selectConfiguredFallback(c, retry, requestedModel, policy)
		}
		candidates = candidates[fallbackIndex : fallbackIndex+1]
	}

	for _, candidate := range candidates {
		channel, err := model.GetRandomSatisfiedChannel(group, candidate, 0, filters)
		if err != nil || channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		if ok, _ := model.ChannelSatisfiesFilters(channel, candidate, filters); !ok {
			continue
		}
		return channel, group, candidate
	}
	return nil, "", ""
}

func selectConfiguredFallback(c *gin.Context, retry *RetryParam, requestedModel string, policy ratio_setting.GroupPolicy) (*model.Channel, string, string) {
	filters := fallbackSelectionFilters(c)
	if policy.FallbackGroup == "" {
		return nil, "", ""
	}
	targetModels := []string{policy.FallbackModel}
	if policy.FallbackModel == "" || policy.FallbackModel == "random" {
		if retry.fallbackModels == nil {
			retry.fallbackModels = make(map[string][]string)
		}
		key := "fallback:" + policy.FallbackGroup
		var exists bool
		targetModels, exists = retry.fallbackModels[key]
		if !exists {
			targetModels = model.GetGroupEnabledModels(policy.FallbackGroup)
			targetModels = orderGroupFallbackModels(targetModels, requestedModel, ratio_setting.GetGroupPolicy(policy.FallbackGroup).RouteStrategy)
			retry.fallbackModels[key] = targetModels
		}
	}
	for _, candidate := range targetModels {
		if candidate == "" || candidate == requestedModel || !groupFallbackModelAllowedByToken(c, candidate) {
			continue
		}
		channel, err := model.GetRandomSatisfiedChannel(policy.FallbackGroup, candidate, 0, filters)
		if err == nil && channel != nil && channel.Status == common.ChannelStatusEnabled {
			return channel, policy.FallbackGroup, candidate
		}
	}
	return nil, "", ""
}

func groupFallbackModelAllowedByToken(c *gin.Context, candidate string) bool {
	if !common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled) {
		return true
	}
	value, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
	limit, ok := value.(map[string]bool)
	return ok && limit != nil && groupFallbackModelAllowed(limit, candidate)
}

func orderGroupFallbackModels(models []string, requestedModel, strategy string) []string {
	ordered := append([]string(nil), models...)
	sort.Strings(ordered)
	switch strategy {
	case ratio_setting.AutoRouteRandom:
		start := common.GetRandomInt(len(ordered))
		ordered = append(ordered[start:], ordered[:start]...)
	case ratio_setting.AutoRouteNearest:
		sort.SliceStable(ordered, func(i, j int) bool {
			left, right := modelNameDistance(requestedModel, ordered[i]), modelNameDistance(requestedModel, ordered[j])
			if left == right {
				return ordered[i] < ordered[j]
			}
			return left < right
		})
	}
	return ordered
}

func groupFallbackModelAllowed(limit map[string]bool, modelName string) bool {
	return limit[modelName] || limit[ratio_setting.FormatMatchingModelName(modelName)] || limit[ratio_setting.RoutingMatchModelName(modelName)]
}

func modelNameDistance(left, right string) int {
	leftRunes, rightRunes := []rune(left), []rune(right)
	previous := make([]int, len(rightRunes)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, leftRune := range leftRunes {
		current := make([]int, len(rightRunes)+1)
		current[0] = i + 1
		for j, rightRune := range rightRunes {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[j+1] = min(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(rightRunes)]
}

// Origin-task pins report a fixed code so task polling can tell a retired
// channel from a malformed request.
func pinnedChannelUnavailable(pin dto.ChannelPin, statusCode int, messageID string) *ChannelSelectError {
	if pin.Source == dto.PinSourceOriginTask {
		return &ChannelSelectError{StatusCode: http.StatusBadRequest, Code: "origin_task_channel_disabled", Message: "origin_task_channel_disabled"}
	}
	return &ChannelSelectError{StatusCode: statusCode, MessageID: messageID}
}

// AppendUsedChannel records an attempted channel in the request's channel
// trail, which the retry log and the consume log's admin_info both read.
func AppendUsedChannel(c *gin.Context, channelID int) {
	c.Set("use_channel", append(c.GetStringSlice("use_channel"), fmt.Sprintf("%d", channelID)))
}
