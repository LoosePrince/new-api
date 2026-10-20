package ratio_setting

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupPolicyEndpointNormalization(t *testing.T) {
	require.NoError(t, ValidateGroupPoliciesJSON(`{"default":{"endpoint":"/123","require_key":false}}`))
	require.NoError(t, UpdateGroupPoliciesByJSONString(`{"default":{"endpoint":"/123","require_key":false}}`))
	t.Cleanup(func() { require.NoError(t, UpdateGroupPoliciesByJSONString(`{}`)) })

	path, endpoint, ok := NormalizeGroupEndpointPath("/123/v1/chat/completions")
	require.True(t, ok)
	assert.Equal(t, "/v1/chat/completions", path)
	assert.Equal(t, "/123", endpoint)

	path, _, ok = NormalizeGroupEndpointPath("/123/chat/completions")
	require.True(t, ok)
	assert.Equal(t, "/v1/chat/completions", path)

	for _, route := range []string{
		"/123/v1beta/models/gemini-2.5-pro:generateContent",
		"/123/v1beta/models/gemini-2.5-pro:streamGenerateContent",
	} {
		path, endpoint, ok = NormalizeGroupEndpointPath(route)
		require.True(t, ok)
		assert.Equal(t, strings.TrimPrefix(route, "/123"), path)
		assert.Equal(t, "/123", endpoint)
	}
}

func TestGroupPolicyAllowsSharedEndpointsWithoutSelectingGroup(t *testing.T) {
	require.NoError(t, ValidateGroupPoliciesJSON(`{"a":{"endpoint":"/123"},"b":{"endpoint":"/123"}}`))
	require.NoError(t, UpdateGroupPoliciesByJSONString(`{"a":{"endpoint":"/123"},"b":{"endpoint":"/123"}}`))
	t.Cleanup(func() { require.NoError(t, UpdateGroupPoliciesByJSONString(`{}`)) })

	path, endpoint, ok := NormalizeGroupEndpointPath("/123/v1/chat/completions")
	require.True(t, ok)
	assert.Equal(t, "/v1/chat/completions", path)
	assert.Equal(t, "/123", endpoint)
	assert.Len(t, GroupPoliciesForEndpoint(endpoint), 2)
}

func TestGroupPolicyUsesLongestSharedEndpointAlias(t *testing.T) {
	require.NoError(t, ValidateGroupPoliciesJSON(`{"base":{"endpoint":"/api"},"nested":{"endpoint":"/api/v2"}}`))
	require.NoError(t, UpdateGroupPoliciesByJSONString(`{"base":{"endpoint":"/api"},"nested":{"endpoint":"/api/v2"}}`))
	t.Cleanup(func() { require.NoError(t, UpdateGroupPoliciesByJSONString(`{}`)) })

	path, endpoint, ok := NormalizeGroupEndpointPath("/api/v2/chat/completions")
	require.True(t, ok)
	assert.Equal(t, "/v1/chat/completions", path)
	assert.Equal(t, "/api/v2", endpoint)
}

func TestFormatMatchingModelNameDoesNotStripBase(t *testing.T) {
	assert.Equal(t, "qwen3-max@thinking:on", FormatMatchingModelName("qwen3-max@thinking:on"))
	assert.Equal(t, "claude-3-7-sonnet-thinking", FormatMatchingModelName("claude-3-7-sonnet-thinking"))
	assert.Equal(t, "gemini-2.5-flash-thinking-*", FormatMatchingModelName("gemini-2.5-flash-thinking-8192"))
	assert.Equal(t, "gpt-4-gizmo-*", FormatMatchingModelName("gpt-4-gizmo-abc"))
}

func TestRoutingMatchModelNameStripsThenWildcards(t *testing.T) {
	assert.Equal(t, "qwen3-max", RoutingMatchModelName("qwen3-max@thinking:on@temperature:0.2"))
	assert.Equal(t, "claude-3-7-sonnet", RoutingMatchModelName("claude-3-7-sonnet-thinking"))
	assert.Equal(t, "gemini-2.5-flash-thinking-*", RoutingMatchModelName("gemini-2.5-flash-thinking-8192"))
	assert.Equal(t, "gpt-5.1-codex-max", RoutingMatchModelName("gpt-5.1-codex-max"))

	geminiSettings := model_setting.GetGeminiSettings()
	old := geminiSettings.ThinkingAdapterEnabled
	geminiSettings.ThinkingAdapterEnabled = true
	t.Cleanup(func() { geminiSettings.ThinkingAdapterEnabled = old })
	assert.Equal(t, "gemini-2.5-flash", RoutingMatchModelName("gemini-2.5-flash-thinking-8192"))
}

func TestRoutingMatchModelNamePreservesExemptAtName(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	original := append([]string(nil), settings.ThinkingModelBlacklist...)
	t.Cleanup(func() { settings.ThinkingModelBlacklist = original })
	settings.ThinkingModelBlacklist = append(original, "re:.*@sha256:.*")

	assert.Equal(t, "opaque@sha256:deadbeef", RoutingMatchModelName("opaque@sha256:deadbeef"))
	assert.Equal(t, "kimi-k2-thinking", RoutingMatchModelName("kimi-k2-thinking"))
}
