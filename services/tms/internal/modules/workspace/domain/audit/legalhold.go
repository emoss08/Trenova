package audit

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type LegalHold struct {
	keel.Table `keel:"audit_legal_holds"`

	ID             pulid.ID            `json:"id"                      keel:"id,pk,autoid:alh_"`
	BusinessUnitID pulid.ID            `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID            `json:"organizationId"          keel:"organization_id,pk,tenant"`
	Name           string              `json:"name"                    keel:"name"`
	Reason         string              `json:"reason"                  keel:"reason"`
	Resource       permission.Resource `json:"resource,omitempty"      keel:"resource,nullzero"`
	ResourceID     string              `json:"resourceId,omitempty"    keel:"resource_id,nullzero"`
	PlacedByID     pulid.ID            `json:"placedById"              keel:"placed_by_id"`
	PlacedAt       int64               `json:"placedAt"                keel:"placed_at"`
	ReleasedByID   pulid.ID            `json:"releasedById,omitempty"  keel:"released_by_id,nullzero"`
	ReleasedAt     *int64              `json:"releasedAt,omitempty"    keel:"released_at"`
	ReleaseReason  string              `json:"releaseReason,omitempty" keel:"release_reason,nullzero"`
	Version        int64               `json:"version"                 keel:"version,version"`
	CreatedAt      int64               `json:"createdAt"               keel:"created_at,autocreate"`
	UpdatedAt      int64               `json:"updatedAt"               keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

const (
	maxLegalHoldNameLength = 200
	maxLegalHoldReason     = 4000
)

func (h *LegalHold) GetTableName() string {
	return "audit_legal_holds"
}

func (h *LegalHold) GetID() pulid.ID {
	return h.ID
}

func (h *LegalHold) GetCreatedAt() int64 {
	return h.PlacedAt
}

func (h *LegalHold) BeforeCreate(_ context.Context) error {
	if h.PlacedAt == 0 {
		h.PlacedAt = timeutils.NowUnix()
	}
	return nil
}

func (h *LegalHold) Active() bool {
	return h != nil && h.ReleasedAt == nil
}

func (h *LegalHold) Validate(multiErr *errortypes.MultiError) {
	h.Name = strings.TrimSpace(h.Name)
	h.Reason = strings.TrimSpace(h.Reason)
	h.ResourceID = strings.TrimSpace(h.ResourceID)

	switch {
	case h.Name == "":
		multiErr.Add("name", errortypes.ErrRequired, "Name the hold")
	case len(h.Name) > maxLegalHoldNameLength:
		multiErr.Add("name", errortypes.ErrInvalid, "A hold's name is at most 200 characters")
	}
	switch {
	case h.Reason == "":
		multiErr.Add("reason", errortypes.ErrRequired, "Say why the entries must be kept")
	case len(h.Reason) > maxLegalHoldReason:
		multiErr.Add("reason", errortypes.ErrInvalid, "A hold's reason is at most 4,000 characters")
	}
	if h.ResourceID != "" && h.Resource == "" {
		multiErr.Add("resource", errortypes.ErrRequired, "Name the kind of record the hold covers")
	}
	if h.Resource != "" && !permissioncatalog.Default().HasResource(h.Resource.String()) {
		multiErr.Add("resource", errortypes.ErrInvalid, "Trenova has no such kind of record")
	}
}

func (h *LegalHold) Covers(resource permission.Resource, resourceID string) bool {
	if !h.Active() {
		return false
	}
	if h.Resource == "" {
		return true
	}
	if h.Resource != resource {
		return false
	}
	return h.ResourceID == "" || h.ResourceID == resourceID
}
