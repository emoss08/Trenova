package bulkedit

import (
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MaxTargets        = 10_000
	MaxSelectedIDs    = 10_000
	MaxStoredFailures = 100
	UndoWindowSeconds = 24 * 60 * 60
	MaxValueLength    = 200
	MaxFailureMessage = 500
)

type Status string

const (
	StatusQueued  = Status("Queued")
	StatusRunning = Status("Running")
	StatusDone    = Status("Completed")
	StatusFailed  = Status("Failed")
	StatusUndoing = Status("Undoing")
	StatusUndone  = Status("Undone")
)

func (s Status) IsTerminal() bool {
	return s == StatusDone || s == StatusFailed || s == StatusUndone
}

type SelectionFilter struct {
	Query        string                    `json:"query"`
	FieldFilters []domaintypes.FieldFilter `json:"fieldFilters"`
	FilterGroups []domaintypes.FilterGroup `json:"filterGroups"`
	Options      map[string]any            `json:"options"`
}

type Selection struct {
	IDs    []string         `json:"ids"`
	Filter *SelectionFilter `json:"filter"`
}

type Target struct {
	ID       string `json:"id"`
	Previous string `json:"previous"`
	Done     bool   `json:"done"`
	Changed  bool   `json:"changed"`
	Error    string `json:"error,omitempty"`
}

type BulkEdit struct {
	keel.Table `keel:"bulk_edits"`

	ID             pulid.ID  `json:"id"             keel:"id,pk,autoid:bke_"`
	OrganizationID pulid.ID  `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID  `json:"userId"         keel:"user_id"`
	Resource       string    `json:"resource"       keel:"resource"`
	Field          string    `json:"field"          keel:"field"`
	Value          string    `json:"value"          keel:"value"`
	Selection      Selection `json:"selection"      keel:"selection,jsonb"`
	Targets        []Target  `json:"targets"        keel:"targets,jsonb"`
	Status         Status    `json:"status"         keel:"status"`
	TotalCount     int       `json:"totalCount"     keel:"total_count"`
	ChangedCount   int       `json:"changedCount"   keel:"changed_count"`
	FailedCount    int       `json:"failedCount"    keel:"failed_count"`
	ProcessedCount int       `json:"processedCount" keel:"processed_count"`
	FailureMessage string    `json:"failureMessage" keel:"failure_message"`
	CompletedAt    *int64    `json:"completedAt"    keel:"completed_at"`
	UndoneAt       *int64    `json:"undoneAt"       keel:"undone_at"`
	Version        int64     `json:"version"        keel:"version,version"`
	CreatedAt      int64     `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64     `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	User         *tenant.User         `json:"user,omitempty"         keel:"belongs:user_id"`
}

var (
	_ validationframework.TenantedEntity = (*BulkEdit)(nil)
)

func (b *BulkEdit) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.OrganizationID, validation.Required),
		validation.Field(&b.BusinessUnitID, validation.Required),
		validation.Field(&b.UserID, validation.Required),
		validation.Field(&b.Resource, validation.Required, validation.Length(1, 100)),
		validation.Field(&b.Field, validation.Required, validation.Length(1, 100)),
		validation.Field(&b.Value, validation.Length(0, MaxValueLength)),
	))

	hasIDs := len(b.Selection.IDs) > 0
	hasFilter := b.Selection.Filter != nil
	switch {
	case hasIDs && hasFilter:
		multiErr.Add(
			"selection",
			errortypes.ErrInvalid,
			"Choose either the selected rows or every row matching the filters",
		)
	case !hasIDs && !hasFilter:
		multiErr.Add("selection", errortypes.ErrRequired, "Choose the rows to change")
	case len(b.Selection.IDs) > MaxSelectedIDs:
		multiErr.Add(
			"selection.ids",
			errortypes.ErrInvalid,
			fmt.Sprintf("A bulk edit can change at most %d rows", MaxSelectedIDs),
		)
	}
	for idx, id := range b.Selection.IDs {
		if strings.TrimSpace(id) == "" || len(id) > 100 {
			multiErr.Add(
				fmt.Sprintf("selection.ids[%d]", idx),
				errortypes.ErrInvalid,
				"A row ID is not valid",
			)
		}
	}
}

func (b *BulkEdit) CanUndo(now int64) bool {
	return b.Status == StatusDone &&
		b.ChangedCount > 0 &&
		b.CompletedAt != nil &&
		now-*b.CompletedAt <= UndoWindowSeconds
}

func (b *BulkEdit) GetID() pulid.ID {
	return b.ID
}

func (b *BulkEdit) GetOrganizationID() pulid.ID {
	return b.OrganizationID
}

func (b *BulkEdit) GetBusinessUnitID() pulid.ID {
	return b.BusinessUnitID
}

func (b *BulkEdit) GetTableName() string {
	return "bulk_edits"
}
