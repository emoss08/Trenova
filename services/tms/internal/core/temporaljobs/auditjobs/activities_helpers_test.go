package auditjobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActivities(t *testing.T) {
	t.Parallel()

	mockAuditRepo := new(mockAuditRepository)
	mockBufferRepo := new(mockAuditBufferRepository)
	mockDLQRepo := new(mockAuditDLQRepository)
	mockDRRepo := new(mockDataRetentionRepository)

	params := ActivitiesParams{
		AuditRepository:         mockAuditRepo,
		AuditBufferRepository:   mockBufferRepo,
		AuditDLQRepository:      mockDLQRepo,
		DataRetentionRepository: mockDRRepo,
	}

	activities := NewActivities(params)

	require.NotNil(t, activities)
	assert.Equal(t, mockAuditRepo, activities.ar)
	assert.Equal(t, mockBufferRepo, activities.abr)
	assert.Equal(t, mockDLQRepo, activities.adlq)
	assert.Equal(t, mockDRRepo, activities.dr)
	assert.Nil(t, activities.rt)
	assert.Nil(t, activities.metrics)
}

func TestNewActivities_WithNilRepos(t *testing.T) {
	t.Parallel()

	params := ActivitiesParams{}

	activities := NewActivities(params)

	require.NotNil(t, activities)
	assert.Nil(t, activities.ar)
	assert.Nil(t, activities.abr)
	assert.Nil(t, activities.adlq)
	assert.Nil(t, activities.dr)
	assert.Nil(t, activities.rt)
	assert.Nil(t, activities.metrics)
}

func TestDefaultConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 500, defaultBatchSize)
	assert.Equal(t, 5000, defaultMaxEntries)
	assert.Equal(t, 5, defaultDLQMaxRetry)
}
