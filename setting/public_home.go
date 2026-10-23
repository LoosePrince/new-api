package setting

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const (
	PublicStatsEnabledOption           = "PublicStatsEnabled"
	PublicChatEnabledOption            = "PublicChatEnabled"
	PublicChatGroupOption              = "PublicChatGroup"
	PublicChatRoutingModeOption        = "PublicChatRoutingMode"
	PublicChatModelOption              = "PublicChatModel"
	PublicStatsRequestAdjustmentOption = "PublicStatsRequestAdjustment"
	PublicStatsTokenAdjustmentOption   = "PublicStatsTokenAdjustment"
	PublicStatsSuccessAdjustmentOption = "PublicStatsSuccessAdjustment"
	PublicStatsFailureAdjustmentOption = "PublicStatsFailureAdjustment"
)

type PublicHomeSettings struct {
	StatsEnabled      bool
	ChatEnabled       bool
	ChatGroup         string
	ChatRoutingMode   string
	ChatModel         string
	RequestAdjustment int64
	TokenAdjustment   int64
	SuccessAdjustment int64
	FailureAdjustment int64
}

var publicHomeSettings = struct {
	sync.RWMutex
	value PublicHomeSettings
}{
	value: PublicHomeSettings{ChatRoutingMode: "group"},
}

func GetPublicHomeSettings() PublicHomeSettings {
	publicHomeSettings.RLock()
	defer publicHomeSettings.RUnlock()
	return publicHomeSettings.value
}

func IsPublicHomeOption(key string) bool {
	switch key {
	case PublicStatsEnabledOption, PublicChatEnabledOption, PublicChatGroupOption,
		PublicChatRoutingModeOption, PublicChatModelOption,
		PublicStatsRequestAdjustmentOption, PublicStatsTokenAdjustmentOption,
		PublicStatsSuccessAdjustmentOption, PublicStatsFailureAdjustmentOption:
		return true
	default:
		return false
	}
}

func ValidatePublicHomeOption(key, value string) error {
	switch key {
	case PublicStatsEnabledOption, PublicChatEnabledOption:
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("%s must be a boolean", key)
		}
	case PublicStatsRequestAdjustmentOption, PublicStatsTokenAdjustmentOption,
		PublicStatsSuccessAdjustmentOption, PublicStatsFailureAdjustmentOption:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("%s must be an integer", key)
		}
	case PublicChatGroupOption:
		if len([]rune(strings.TrimSpace(value))) > 100 {
			return fmt.Errorf("%s must not exceed 100 characters", key)
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

func UpdatePublicHomeOption(key, value string) error {
	if err := ValidatePublicHomeOption(key, value); err != nil {
		return err
	}

	publicHomeSettings.Lock()
	defer publicHomeSettings.Unlock()

	switch key {
	case PublicStatsEnabledOption:
		publicHomeSettings.value.StatsEnabled, _ = strconv.ParseBool(value)
	case PublicChatEnabledOption:
		publicHomeSettings.value.ChatEnabled, _ = strconv.ParseBool(value)
	case PublicChatGroupOption:
		publicHomeSettings.value.ChatGroup = strings.TrimSpace(value)
	case PublicChatRoutingModeOption:
		publicHomeSettings.value.ChatRoutingMode = strings.TrimSpace(value)
	case PublicChatModelOption:
		publicHomeSettings.value.ChatModel = strings.TrimSpace(value)
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
