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
