package auditservice

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingRealtimeService struct {
	mu       sync.Mutex
	requests []*agentports.PublishResourceInvalidationRequest
}

func (s *recordingRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	req *agentports.PublishResourceInvalidationRequest,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)
	return nil
}

func (s *recordingRealtimeService) recordChanges() []*agentports.PublishResourceInvalidationRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	changes := make([]*agentports.PublishResourceInvalidationRequest, 0, len(s.requests))
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

	require.NoError(t, srv.LogActions([]auditports.BulkLogEntry{
		{Params: first},
		{Params: &second},
		{Params: &other},
	}))

	changes := realtime.recordChanges()
	require.Len(t, changes, 2)
	byResource := map[string]*agentports.PublishResourceInvalidationRequest{}
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
