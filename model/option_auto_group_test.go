package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOptionValueRejectsInvalidMaxTokenAutoGroups(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "1.5", "invalid"} {
		t.Run(value, func(t *testing.T) {
			assert.Error(t, validateOptionValue("MaxTokenAutoGroups", value))
		})
	}
	require.NoError(t, validateOptionValue("MaxTokenAutoGroups", "999999"))
}

func TestValidateOptionValueRequestCaptureSettings(t *testing.T) {
	require.NoError(t, validateOptionValue("log_setting.request_capture_enabled", "true"))
	require.NoError(t, validateOptionValue("log_setting.request_capture_max_body_kb", "256"))
	require.NoError(t, validateOptionValue("log_setting.request_capture_retention_days", "30"))

	for _, testCase := range []struct {
		key   string
		value string
	}{
		{key: "log_setting.request_capture_enabled", value: "maybe"},
		{key: "log_setting.request_capture_max_body_kb", value: "0"},
		{key: "log_setting.request_capture_max_body_kb", value: "10241"},
		{key: "log_setting.request_capture_retention_days", value: "0"},
		{key: "log_setting.request_capture_retention_days", value: "3651"},
	} {
		t.Run(testCase.key+"/"+testCase.value, func(t *testing.T) {
			assert.Error(t, validateOptionValue(testCase.key, testCase.value))
		})
	}
}
