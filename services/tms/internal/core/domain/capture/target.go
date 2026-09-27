package capture

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// fileableResources are the records a captured document may be filed onto.
// They are the records people keep paperwork for; anything else a document
// can hang off (an assistant thread, an inbound message) is not somewhere a
// person files a scan.
var fileableResources = []permission.Resource{
	permission.ResourceShipment,
	permission.ResourceWorker,
	permission.ResourceTractor,
	permission.ResourceTrailer,
	permission.ResourceCustomer,
	permission.ResourceCarrier,
}

// FileableResources lists the record kinds a capture may be filed onto.
func FileableResources() []permission.Resource {
	return slices.Clone(fileableResources)
}

// IsFileableResource reports whether a record kind may receive a capture.
func IsFileableResource(resourceType string) bool {
	return slices.Contains(fileableResources, permission.Resource(resourceType))
}

// Target is where captured pages are to be filed: a record, and optionally the
// kind of document they are.
type Target struct {
	ResourceType   string    `json:"resourceType"`
	ResourceID     *pulid.ID `json:"resourceId"`
	DocumentTypeID *pulid.ID `json:"documentTypeId"`
}

// HasRecord reports whether the target names a record at all.
func (t Target) HasRecord() bool {
	return t.ResourceType != "" && t.ResourceID != nil && t.ResourceID.IsNotNil()
}

// TargetFields names the fields a target's errors are reported against, so a
// form shows each error beside the control it belongs to.
type TargetFields struct {
	Type string
	ID   string
}

// targetFields is where every entity that carries a target reports its
// errors, since they all name the fields the same way.
var targetFields = TargetFields{Type: "targetType", ID: "targetId"}

// Validate checks the target is somewhere a document can be filed. A target
// with a document type but no record is refused: a type says what the pages
// are, not where they go.
func (t Target) Validate(multiErr *errortypes.MultiError, fields TargetFields, required bool) {
	hasType := t.ResourceType != ""
	hasID := t.ResourceID != nil && t.ResourceID.IsNotNil()

	switch {
	case !hasType && !hasID:
		if required {
			multiErr.Add(fields.Type, errortypes.ErrRequired, "A record is required")
		} else if t.DocumentTypeID != nil {
			multiErr.Add(fields.Type, errortypes.ErrRequired,
				"A document type needs a record to file onto")
		}
	case !hasType:
		multiErr.Add(fields.Type, errortypes.ErrRequired, "Record kind is required")
	case !hasID:
		multiErr.Add(fields.ID, errortypes.ErrRequired, "Record is required")
	case !IsFileableResource(t.ResourceType):
		multiErr.Add(fields.Type, errortypes.ErrInvalid,
			"Captured documents cannot be filed onto that kind of record")
	}
}
