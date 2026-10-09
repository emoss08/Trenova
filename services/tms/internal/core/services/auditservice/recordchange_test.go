package auditservice

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingRealtimeService struct {
	mu       sync.Mutex
	requests []*services.PublishResourceInvalidationRequest
}

func (s *recordingRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	req *services.PublishResourceInvalidationRequest,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)
	return nil
}

func (s *recordingRealtimeService) recordChanges() []*services.PublishResourceInvalidationRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	changes := make([]*services.PublishResourceInvalidationRequest, 0, len(s.requests))
	for _, req := range s.requests {
		if req.Resource != "audit-logs" {
			changes = append(changes, req)
		}
	}
	return changes
}

func TestLogAction_PublishesTheRecordChangeForTheAuditedResource(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	realtime := &recordingRealtimeService{}
	srv := newTestService(repo, bufferRepo, realtime)
	bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

	params := validLogActionParams()
	params.Resource = permission.ResourceCommodity
	params.ResourceID = "com_01ABC"
	params.Operation = permission.OpUpdate

	require.NoError(t, srv.LogAction(params))

	changes := realtime.recordChanges()
	require.Len(t, changes, 1)
	assert.Equal(t, "audited:commodity", changes[0].Resource)
	assert.Equal(t, "updated", changes[0].Action)
	assert.Equal(t, "com_01ABC", changes[0].RecordID.String())
	assert.Equal(t, params.OrganizationID, changes[0].OrganizationID)
	assert.Equal(t, params.BusinessUnitID, changes[0].BusinessUnitID)
	assert.Nil(t, changes[0].Entity, "a record change never carries the record")
}

func TestLogActions_PublishesOneBulkChangePerResource(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	realtime := &recordingRealtimeService{}
	srv := newTestService(repo, bufferRepo, realtime)
	bufferRepo.On("PushBatch", mock.Anything, mock.Anything).Return(nil)
	bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

	first := validLogActionParams()
	first.Resource = permission.ResourceLocation
	first.Operation = permission.OpUpdate
	second := *first
	second.ResourceID = "loc_02"
	other := *first
	other.Resource = permission.ResourceCommodity
	other.Operation = permission.OpDelete

	require.NoError(t, srv.LogActions([]services.BulkLogEntry{
		{Params: first},
		{Params: &second},
		{Params: &other},
	}))

	changes := realtime.recordChanges()
	require.Len(t, changes, 2)
	byResource := map[string]*services.PublishResourceInvalidationRequest{}
	for _, change := range changes {
		byResource[change.Resource] = change
	}
	assert.Equal(t, "bulk_updated", byResource["audited:location"].Action)
	assert.True(t, byResource["audited:location"].RecordID.IsNil())
	assert.Equal(t, "deleted", byResource["audited:commodity"].Action)
}

func TestLogAction_PublishesNoRecordChangeForAReadOrTheAuditLogItself(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		resource  permission.Resource
		operation permission.Operation
	}{
		{name: "read", resource: permission.ResourceCommodity, operation: permission.OpRead},
		{name: "export", resource: permission.ResourceCommodity, operation: permission.OpExport},
		{name: "audit log", resource: permission.ResourceAuditLog, operation: permission.OpUpdate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := new(mockAuditRepository)
			bufferRepo := new(mockBufferRepository)
			realtime := &recordingRealtimeService{}
			srv := newTestService(repo, bufferRepo, realtime)
			bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

			params := validLogActionParams()
			params.Resource = tc.resource
			params.Operation = tc.operation
			require.NoError(t, srv.LogAction(params))

			assert.Empty(t, realtime.recordChanges())
		})
	}
}
