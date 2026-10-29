package capture_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func errorFields(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestTargetValidate(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	docType := pulid.MustNew("dt_")

	tests := []struct {
		name     string
		target   capture.Target
		required bool
		fields   []string
	}{
		{name: "empty optional", target: capture.Target{}},
		{
			name:     "empty required",
			target:   capture.Target{},
			required: true,
			fields:   []string{"targetType"},
		},
		{
			name:   "document type without record",
			target: capture.Target{DocumentTypeID: &docType},
			fields: []string{"targetType"},
		},
		{
			name:   "kind without id",
			target: capture.Target{ResourceType: "shipment"},
			fields: []string{"targetId"},
		},
		{
			name:   "id without kind",
			target: capture.Target{ResourceID: &id},
			fields: []string{"targetType"},
		},
		{
			name:   "unfileable kind",
			target: capture.Target{ResourceType: "invoice", ResourceID: &id},
			fields: []string{"targetType"},
		},
		{
			name:     "scan into a conversation",
			target:   capture.Target{ResourceType: "assistant_thread", ResourceID: &id},
			required: true,
		},
		{
			name: "shipment",
			target: capture.Target{
				ResourceType:   "shipment",
				ResourceID:     &id,
				DocumentTypeID: &docType,
			},
			required: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			tt.target.Validate(
				multiErr,
				capture.TargetFields{Type: "targetType", ID: "targetId"},
				tt.required,
			)
			assert.ElementsMatch(t, tt.fields, errorFields(multiErr))
		})
	}
}

func TestAssistantThreadIsATargetButNotAFilingDestination(t *testing.T) {
	t.Parallel()

	assert.True(t, capture.IsCaptureTarget(capture.ResourceAssistantThread))
	assert.False(t, capture.IsFileableResource(capture.ResourceAssistantThread))
}

func TestFileableResourcesAreRealResources(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	for _, resource := range capture.FileableResources() {
		_, ok := registry.Get(resource.String())
		assert.True(t, ok, "%s is fileable but not a registered resource", resource)
	}
}

func TestBatchSettleFiling(t *testing.T) {
	t.Parallel()

	batch := &capture.CaptureBatch{Status: capture.BatchReady, ItemCount: 3}
	batch.SettleFiling()
	assert.Equal(t, capture.BatchReady, batch.Status)

	batch.FiledItemCount = 1
	batch.SettleFiling()
	assert.Equal(t, capture.BatchPartiallyFiled, batch.Status)

	batch.FiledItemCount = 3
	batch.SettleFiling()
	assert.Equal(t, capture.BatchFiled, batch.Status)

	discarded := &capture.CaptureBatch{
		Status:         capture.BatchDiscarded,
		ItemCount:      1,
		FiledItemCount: 1,
	}
	discarded.SettleFiling()
	assert.Equal(t, capture.BatchDiscarded, discarded.Status, "a settled batch stays settled")
}

func TestDeviceRevokeClearsCredentials(t *testing.T) {
	t.Parallel()

	by := pulid.MustNew("usr_")
	device := &capture.CaptureDevice{
		ID:                  pulid.MustNew("cdev_"),
		Status:              capture.DeviceActive,
		AccessTokenHash:     "access",
		RefreshTokenHash:    "refresh",
		PreviousRefreshHash: "previous",
	}

	device.Revoke(&by, "Lost laptop", 500)

	assert.False(t, device.IsActive())
	assert.NotEqual(t, "access", device.AccessTokenHash)
	assert.NotEqual(t, "refresh", device.RefreshTokenHash)
	assert.NotEqual(t, device.AccessTokenHash, "", "the column is unique and not null")
	assert.Empty(t, device.PreviousRefreshHash)
	assert.False(t, device.IsOnline(500))
}

func TestDeviceOnline(t *testing.T) {
	t.Parallel()

	seen := int64(1000)
	device := &capture.CaptureDevice{Status: capture.DeviceActive, LastSeenAt: &seen}
	assert.True(t, device.IsOnline(1000+capture.OnlineWindowSeconds))
	assert.False(t, device.IsOnline(1001+capture.OnlineWindowSeconds))
}

func TestNormalizeRotation(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, capture.NormalizeRotation(360))
	assert.Equal(t, 270, capture.NormalizeRotation(-90))
	assert.Equal(t, 90, capture.NormalizeRotation(450))
	assert.Equal(t, 180, capture.NormalizeRotation(200))
}
