package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRequestCaptureBodiesDeduplicateAndHydrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&RequestCapture{}, &RequestCaptureBody{}))
	previous := LOG_DB
	LOG_DB = db
	t.Cleanup(func() { LOG_DB = previous })

	body := requestCaptureBody{data: []byte(`{"model":"gpt-4o-mini"}`)}
	firstID, err := storeRequestCaptureBody(body)
	require.NoError(t, err)
	secondID, err := storeRequestCaptureBody(body)
	require.NoError(t, err)
	assert.Equal(t, firstID, secondID)

	captures := []*RequestCapture{
		{Id: 1, RequestBodyId: firstID},
		{Id: 2, UpstreamRequestBodyId: secondID},
	}
	require.NoError(t, db.Create(captures).Error)
	require.NoError(t, hydrateRequestCaptureBodies(captures))
	assert.Equal(t, string(body.data), captures[0].RequestBody)
	assert.Equal(t, string(body.data), captures[1].UpstreamRequestBody)

	var storedCount int64
	require.NoError(t, db.Model(&RequestCaptureBody{}).Count(&storedCount).Error)
	assert.EqualValues(t, 1, storedCount)

	require.NoError(t, DeleteRequestCapture(1))
	require.NoError(t, db.Model(&RequestCaptureBody{}).Count(&storedCount).Error)
	assert.EqualValues(t, 1, storedCount)
	require.NoError(t, DeleteRequestCapture(2))
	require.NoError(t, db.Model(&RequestCaptureBody{}).Count(&storedCount).Error)
	assert.EqualValues(t, 0, storedCount)
}
