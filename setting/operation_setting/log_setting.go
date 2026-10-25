package operation_setting

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

const LogMaxRangeDaysOptionKey = "log_setting.max_range_days"

// LogSetting limits how far regular users can reach into usage logs.
type LogSetting struct {
	// UserExportEnabled lets regular users export their own usage logs.
	// Administrators can always export.
	UserExportEnabled bool `json:"user_export_enabled"`
	// MaxRangeDays is the longest time range a user can search in their logs
	// and dashboard, and the longest range any export can cover.
	MaxRangeDays                int  `json:"max_range_days"`
	RequestCaptureEnabled       bool `json:"request_capture_enabled"`
	RequestCaptureOriginal      bool `json:"request_capture_original"`
	RequestCaptureUpstream      bool `json:"request_capture_upstream"`
	RequestCaptureResponse      bool `json:"request_capture_response"`
	RequestCaptureMaxBodyKB     int  `json:"request_capture_max_body_kb"`
	RequestCaptureRetentionDays int  `json:"request_capture_retention_days"`
}

var logSetting = LogSetting{
	UserExportEnabled:           true,
	MaxRangeDays:                365,
	RequestCaptureOriginal:      true,
	RequestCaptureUpstream:      true,
	RequestCaptureResponse:      true,
	RequestCaptureMaxBodyKB:     256,
	RequestCaptureRetentionDays: 30,
}

func init() {
	config.GlobalConfig.Register("log_setting", &logSetting)
}

func GetLogSetting() *LogSetting {
	return &logSetting
}

func (s *LogSetting) MaxRangeSeconds() int64 {
	return int64(max(s.MaxRangeDays, 1)) * 24 * 60 * 60
}

func ValidateRequestCaptureOption(key, value string) error {
	switch key {
	case "log_setting.request_capture_enabled", "log_setting.request_capture_original", "log_setting.request_capture_upstream", "log_setting.request_capture_response":
		if _, err := strconv.ParseBool(value); err != nil {
			return common.NewMessage("Invalid request capture setting")
		}
	case "log_setting.request_capture_max_body_kb":
		size, err := strconv.Atoi(value)
		if err != nil || size < 1 || size > 10240 {
			return common.NewMessage("Maximum captured body size must be between 1 and 10240 KB")
		}
	case "log_setting.request_capture_retention_days":
		days, err := strconv.Atoi(value)
		if err != nil || days < 1 || days > 3650 {
			return common.NewMessage("Request capture retention must be between 1 and 3650 days")
		}
	default:
		return common.NewMessage("Invalid request capture setting")
	}
	return nil
}

func ValidateLogMaxRangeDays(value string) error {
	days, err := strconv.Atoi(value)
	if err != nil || days < 1 {
		return common.NewMessage("The longest log time range must be at least 1 day")
	}
	return nil
}
