package setting

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const (
	PublicStatsEnabledOption              = "PublicStatsEnabled"
	PublicChannelStatusEnabledOption      = "PublicChannelStatusEnabled"
	PublicChannelStatusHomeModeOption     = "PublicChannelStatusHomeMode"
	PublicChannelStatusChannelModeOption  = "PublicChannelStatusChannelMode"
	PublicChannelStatusChannelIDsOption   = "PublicChannelStatusChannelIDs"
	PublicChannelStatusModelEnabledOption = "PublicChannelStatusModelEnabled"
	PublicChatEnabledOption               = "PublicChatEnabled"
	PublicChatGroupOption                 = "PublicChatGroup"
	PublicChatRoutingModeOption           = "PublicChatRoutingMode"
	PublicChatModelOption                 = "PublicChatModel"
	PublicApiGroupOption                  = "PublicApiGroup"
	PublicStatsRequestAdjustmentOption    = "PublicStatsRequestAdjustment"
	PublicStatsTokenAdjustmentOption      = "PublicStatsTokenAdjustment"
	PublicStatsSuccessAdjustmentOption    = "PublicStatsSuccessAdjustment"
	PublicStatsFailureAdjustmentOption    = "PublicStatsFailureAdjustment"
)

// The home page can show the compact site bar (coarse) or the full overview
// card of the status page (detailed), and either choice may hide the dedicated
// status page. They are one setting because the four combinations are the
// choice an operator actually makes, and splitting them into two switches lets
// the two halves drift into states nobody asked for.
const (
	PublicChannelStatusHomeModeCoarse         = "coarse"
	PublicChannelStatusHomeModeCoarseNoPage   = "coarse_no_page"
	PublicChannelStatusHomeModeDetailed       = "detailed"
	PublicChannelStatusHomeModeDetailedNoPage = "detailed_no_page"
)

// Channel information of the public status page: every served channel, no
// channel at all, or the channels the administrator hand-picked.
const (
	PublicChannelStatusChannelModeAll      = "all"
	PublicChannelStatusChannelModeNone     = "none"
	PublicChannelStatusChannelModeSelected = "selected"
)

// maxPublicChannelStatusChannelIDs bounds the hand-picked channel list so a
// malformed option cannot make every status request arbitrarily large.
const maxPublicChannelStatusChannelIDs = 1000

type PublicHomeSettings struct {
	StatsEnabled               bool
	ChannelStatusEnabled       bool
	ChannelStatusHomeMode      string
	ChannelStatusChannelMode   string
	ChannelStatusChannelIDs    []int
	ChannelStatusModelsEnabled bool
	ChatEnabled                bool
	ChatGroup                  string
	ChatRoutingMode            string
	ChatModel                  string
	ApiGroup                   string
	RequestAdjustment          int64
	TokenAdjustment            int64
	SuccessAdjustment          int64
	FailureAdjustment          int64
}

// ChannelStatusHomeDetailed reports whether the home page renders the full
// overview card rather than the compact bar.
func (settings PublicHomeSettings) ChannelStatusHomeDetailed() bool {
	return settings.ChannelStatusHomeMode == PublicChannelStatusHomeModeDetailed ||
		settings.ChannelStatusHomeMode == PublicChannelStatusHomeModeDetailedNoPage
}

// ChannelStatusPageEnabled reports whether the dedicated status page stays
// reachable for visitors.
func (settings PublicHomeSettings) ChannelStatusPageEnabled() bool {
	return settings.ChannelStatusHomeMode != PublicChannelStatusHomeModeCoarseNoPage &&
		settings.ChannelStatusHomeMode != PublicChannelStatusHomeModeDetailedNoPage
}

var publicHomeSettings = struct {
	sync.RWMutex
	value PublicHomeSettings
}{
	value: PublicHomeSettings{
		ChatRoutingMode:            "group",
		ApiGroup:                   "default",
		ChannelStatusHomeMode:      PublicChannelStatusHomeModeCoarse,
		ChannelStatusChannelMode:   PublicChannelStatusChannelModeAll,
		ChannelStatusModelsEnabled: true,
	},
}

func GetPublicHomeSettings() PublicHomeSettings {
	publicHomeSettings.RLock()
	defer publicHomeSettings.RUnlock()
	settings := publicHomeSettings.value
	// The channel selection is the only slice in the struct, so it is copied to
	// keep a caller from mutating the shared settings behind the lock.
	settings.ChannelStatusChannelIDs = slices.Clone(publicHomeSettings.value.ChannelStatusChannelIDs)
	return settings
}

func IsPublicHomeOption(key string) bool {
	switch key {
	case PublicStatsEnabledOption, PublicChannelStatusEnabledOption,
		PublicChannelStatusHomeModeOption,
		PublicChannelStatusChannelModeOption, PublicChannelStatusChannelIDsOption,
		PublicChannelStatusModelEnabledOption,
		PublicChatEnabledOption,
		PublicChatGroupOption, PublicChatRoutingModeOption, PublicChatModelOption, PublicApiGroupOption,
		PublicStatsRequestAdjustmentOption, PublicStatsTokenAdjustmentOption,
		PublicStatsSuccessAdjustmentOption, PublicStatsFailureAdjustmentOption:
		return true
	default:
		return false
	}
}

func ValidatePublicHomeOption(key, value string) error {
	switch key {
	case PublicStatsEnabledOption, PublicChannelStatusEnabledOption,
		PublicChannelStatusModelEnabledOption, PublicChatEnabledOption:
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("%s must be a boolean", key)
		}
	case PublicChannelStatusHomeModeOption:
		switch value {
		case PublicChannelStatusHomeModeCoarse, PublicChannelStatusHomeModeCoarseNoPage,
			PublicChannelStatusHomeModeDetailed, PublicChannelStatusHomeModeDetailedNoPage:
		default:
			return fmt.Errorf("%s must be coarse, coarse_no_page, detailed or detailed_no_page", key)
		}
	case PublicChannelStatusChannelModeOption:
		switch value {
		case PublicChannelStatusChannelModeAll, PublicChannelStatusChannelModeNone,
			PublicChannelStatusChannelModeSelected:
		default:
			return fmt.Errorf("%s must be all, none or selected", key)
		}
	case PublicChannelStatusChannelIDsOption:
		ids, err := parsePublicChannelStatusChannelIDs(value)
		if err != nil {
			return fmt.Errorf("%s must be a JSON array of channel ids", key)
		}
		if len(ids) > maxPublicChannelStatusChannelIDs {
			return fmt.Errorf("%s must not exceed %d channels", key, maxPublicChannelStatusChannelIDs)
		}
	case PublicStatsRequestAdjustmentOption, PublicStatsTokenAdjustmentOption,
		PublicStatsSuccessAdjustmentOption, PublicStatsFailureAdjustmentOption:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("%s must be an integer", key)
		}
	case PublicChatGroupOption, PublicApiGroupOption:
		group := strings.TrimSpace(value)
		if len([]rune(group)) > 100 {
			return fmt.Errorf("%s must not exceed 100 characters", key)
		}
		if key == PublicApiGroupOption && group != "" && !ratio_setting.ContainsGroupRatio(group) {
			return fmt.Errorf("%s must be an existing group", key)
		}
	case PublicChatRoutingModeOption:
		if value != "group" && value != "model" {
			return fmt.Errorf("%s must be group or model", key)
		}
	case PublicChatModelOption:
		if len([]rune(strings.TrimSpace(value))) > 200 {
			return fmt.Errorf("%s must not exceed 200 characters", key)
		}
	default:
		return nil
	}
	return nil
}

// parsePublicChannelStatusChannelIDs reads the stored channel selection. An
// empty value means "no channel picked yet", which is not an error.
func parsePublicChannelStatusChannelIDs(value string) ([]int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	var raw []int
	if err := common.UnmarshalJsonStr(trimmed, &raw); err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(raw))
	for _, id := range raw {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func UpdatePublicHomeOption(key, value string) error {
	if err := ValidatePublicHomeOption(key, value); err != nil {
		return err
	}

	publicHomeSettings.Lock()
	defer publicHomeSettings.Unlock()

	switch key {
	case PublicStatsEnabledOption:
		publicHomeSettings.value.StatsEnabled, _ = strconv.ParseBool(value)
	case PublicChannelStatusEnabledOption:
		publicHomeSettings.value.ChannelStatusEnabled, _ = strconv.ParseBool(value)
	case PublicChannelStatusHomeModeOption:
		publicHomeSettings.value.ChannelStatusHomeMode = value
	case PublicChannelStatusChannelModeOption:
		publicHomeSettings.value.ChannelStatusChannelMode = value
	case PublicChannelStatusChannelIDsOption:
		ids, _ := parsePublicChannelStatusChannelIDs(value)
		publicHomeSettings.value.ChannelStatusChannelIDs = ids
	case PublicChannelStatusModelEnabledOption:
		publicHomeSettings.value.ChannelStatusModelsEnabled, _ = strconv.ParseBool(value)
	case PublicChatEnabledOption:
		publicHomeSettings.value.ChatEnabled, _ = strconv.ParseBool(value)
	case PublicChatGroupOption:
		publicHomeSettings.value.ChatGroup = strings.TrimSpace(value)
	case PublicChatRoutingModeOption:
		publicHomeSettings.value.ChatRoutingMode = strings.TrimSpace(value)
	case PublicChatModelOption:
		publicHomeSettings.value.ChatModel = strings.TrimSpace(value)
	case PublicApiGroupOption:
		publicHomeSettings.value.ApiGroup = strings.TrimSpace(value)
	case PublicStatsRequestAdjustmentOption:
		publicHomeSettings.value.RequestAdjustment, _ = strconv.ParseInt(value, 10, 64)
	case PublicStatsTokenAdjustmentOption:
		publicHomeSettings.value.TokenAdjustment, _ = strconv.ParseInt(value, 10, 64)
	case PublicStatsSuccessAdjustmentOption:
		publicHomeSettings.value.SuccessAdjustment, _ = strconv.ParseInt(value, 10, 64)
	case PublicStatsFailureAdjustmentOption:
		publicHomeSettings.value.FailureAdjustment, _ = strconv.ParseInt(value, 10, 64)
	}
	return nil
}
