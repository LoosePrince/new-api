package setting

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicApiGroupOption(t *testing.T) {
	previous := GetPublicHomeSettings().ApiGroup
	t.Cleanup(func() {
		require.NoError(t, UpdatePublicHomeOption(PublicApiGroupOption, previous))
	})

	assert.True(t, IsPublicHomeOption(PublicApiGroupOption))
	assert.NoError(t, ValidatePublicHomeOption(PublicApiGroupOption, "default"))
	assert.Error(t, ValidatePublicHomeOption(PublicApiGroupOption, strings.Repeat("x", 101)))
	assert.Error(t, ValidatePublicHomeOption(PublicApiGroupOption, "missing"))

	require.NoError(t, UpdatePublicHomeOption(PublicApiGroupOption, "default"))
	assert.Equal(t, "default", GetPublicHomeSettings().ApiGroup)
}

// TestPublicChannelStatusOptions pins the published status settings: the home
// mode decides both the home layout and the page availability, and the channel
// selection is validated before it can reach the public report.
func TestPublicChannelStatusOptions(t *testing.T) {
	previous := GetPublicHomeSettings()
	t.Cleanup(func() {
		require.NoError(t, UpdatePublicHomeOption(PublicChannelStatusHomeModeOption, previous.ChannelStatusHomeMode))
		require.NoError(t, UpdatePublicHomeOption(PublicChannelStatusChannelModeOption, previous.ChannelStatusChannelMode))
		require.NoError(t, UpdatePublicHomeOption(PublicChannelStatusChannelIDsOption, "[]"))
		require.NoError(t, UpdatePublicHomeOption(PublicChannelStatusModelEnabledOption, "true"))
	})

	for _, option := range []string{
		PublicChannelStatusHomeModeOption,
		PublicChannelStatusChannelModeOption,
		PublicChannelStatusChannelIDsOption,
		PublicChannelStatusModelEnabledOption,
	} {
		assert.True(t, IsPublicHomeOption(option))
	}

	cases := []struct {
		mode         string
		wantDetailed bool
		wantPage     bool
	}{
		{mode: PublicChannelStatusHomeModeCoarse, wantDetailed: false, wantPage: true},
		{mode: PublicChannelStatusHomeModeCoarseNoPage, wantDetailed: false, wantPage: false},
		{mode: PublicChannelStatusHomeModeDetailed, wantDetailed: true, wantPage: true},
		{mode: PublicChannelStatusHomeModeDetailedNoPage, wantDetailed: true, wantPage: false},
	}
	for _, tc := range cases {
		require.NoError(t, UpdatePublicHomeOption(PublicChannelStatusHomeModeOption, tc.mode))
		settings := GetPublicHomeSettings()
		assert.Equal(t, tc.wantDetailed, settings.ChannelStatusHomeDetailed(), tc.mode)
		assert.Equal(t, tc.wantPage, settings.ChannelStatusPageEnabled(), tc.mode)
	}
	assert.Error(t, ValidatePublicHomeOption(PublicChannelStatusHomeModeOption, "loud"))

	assert.Error(t, ValidatePublicHomeOption(PublicChannelStatusChannelModeOption, "some"))
	require.NoError(t, UpdatePublicHomeOption(PublicChannelStatusChannelIDsOption, "[3,7]"))
	assert.Equal(t, []int{3, 7}, GetPublicHomeSettings().ChannelStatusChannelIDs)
	// A malformed selection is rejected instead of silently publishing nothing.
	assert.Error(t, ValidatePublicHomeOption(PublicChannelStatusChannelIDsOption, "[3,"))
	require.NoError(t, UpdatePublicHomeOption(PublicChannelStatusChannelIDsOption, "[]"))
	assert.Empty(t, GetPublicHomeSettings().ChannelStatusChannelIDs)
}
