package bulkedit

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	bun.BaseModel `bun:"table:bulk_edits,alias:bke" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID  `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Resource       string    `json:"resource"       bun:"resource,type:VARCHAR(100),notnull"`
	Field          string    `json:"field"          bun:"field,type:VARCHAR(100),notnull"`
	Value          string    `json:"value"          bun:"value,type:VARCHAR(200),notnull"`
	Selection      Selection `json:"selection"      bun:"selection,type:JSONB,notnull"`
	Targets        []Target  `json:"targets"        bun:"targets,type:JSONB,notnull"`
	Status         Status    `json:"status"         bun:"status,type:VARCHAR(20),notnull"`
	TotalCount     int       `json:"totalCount"     bun:"total_count,type:INTEGER,notnull"`
	ChangedCount   int       `json:"changedCount"   bun:"changed_count,type:INTEGER,notnull"`
	FailedCount    int       `json:"failedCount"    bun:"failed_count,type:INTEGER,notnull"`
	ProcessedCount int       `json:"processedCount" bun:"processed_count,type:INTEGER,notnull"`
	FailureMessage string    `json:"failureMessage" bun:"failure_message,type:TEXT,notnull"`
	CompletedAt    *int64    `json:"completedAt"    bun:"completed_at,type:BIGINT,nullzero"`
	UndoneAt       *int64    `json:"undoneAt"       bun:"undone_at,type:BIGINT,nullzero"`
	Version        int64     `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64     `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	User         *tenant.User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
}

var (
	_ bun.BeforeAppendModelHook          = (*BulkEdit)(nil)
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

func (b *BulkEdit) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("bke_")
		}
		b.CreatedAt = now
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
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
