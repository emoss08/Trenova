package desktranscript

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity        = (*Export)(nil)
	_ domaintypes.PostgresSearchable = (*Export)(nil)
)

const ExportIDPrefix = "dtex_"

type ExportKind string

const (
	ExportKindFilter    = ExportKind("Filter")
	ExportKindSelection = ExportKind("Selection")
)

func (k ExportKind) IsValid() bool {
	return k == ExportKindFilter || k == ExportKindSelection
}

type ExportStatus string

const (
	ExportStatusPending   = ExportStatus("Pending")
	ExportStatusRunning   = ExportStatus("Running")
	ExportStatusSucceeded = ExportStatus("Succeeded")
	ExportStatusFailed    = ExportStatus("Failed")
	ExportStatusExpired   = ExportStatus("Expired")
)

func AllExportStatuses() []ExportStatus {
	return []ExportStatus{
		ExportStatusPending,
		ExportStatusRunning,
		ExportStatusSucceeded,
		ExportStatusFailed,
		ExportStatusExpired,
	}
}

func (s ExportStatus) IsValid() bool {
	for _, known := range AllExportStatuses() {
		if s == known {
			return true
		}
	}

	return false
}

func (s ExportStatus) Terminal() bool {
	switch s {
	case ExportStatusSucceeded, ExportStatusFailed, ExportStatusExpired:
		return true
	case ExportStatusPending, ExportStatusRunning:
		return false
	default:
		return false
	}
}

type Coverage struct {
	Kind          ExportKind                  `json:"kind"`
	Timeframe     *Timeframe                  `json:"timeframe,omitempty"`
	Window        Window                      `json:"window"`
	PersonName    string                      `json:"personName,omitempty"`
	AgentName     string                      `json:"agentName,omitempty"`
	Origins       []conversation.ThreadOrigin `json:"origins"`
	SelectedCount int                         `json:"selectedCount,omitempty"`
	OwnerNames    []string                    `json:"ownerNames"`
	OwnerCount    int                         `json:"ownerCount"`
}

type Export struct {
	keel.Table `keel:"desk_transcript_exports"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:dtex_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RequestedByUserID pulid.ID            `json:"requestedByUserId" keel:"requested_by_user_id"`
	Kind              ExportKind          `json:"kind"              keel:"kind"`
	Filter            *ConversationFilter `json:"filter,omitempty"  keel:"filter,jsonb"`
	ThreadIDs         []pulid.ID          `json:"-"                 keel:"thread_ids,jsonb,nullzero"`
	Coverage          *Coverage           `json:"coverage"          keel:"coverage,jsonb"`
	ConversationCount int                 `json:"conversationCount" keel:"conversation_count"`
	Status            ExportStatus        `json:"status"            keel:"status"`
	ByteSize          int64               `json:"byteSize"          keel:"byte_size"`
	SHA256            string              `json:"sha256"            keel:"sha256,nullzero"`
	ArtifactKey       string              `json:"-"                 keel:"artifact_key,nullzero"`
	ArtifactExpiresAt *int64              `json:"artifactExpiresAt" keel:"artifact_expires_at"`
	WorkflowID        string              `json:"-"                 keel:"workflow_id,nullzero"`
	ErrorMessage      string              `json:"errorMessage"      keel:"error_message,nullzero"`
	StartedAt         *int64              `json:"startedAt"         keel:"started_at"`
	CompletedAt       *int64              `json:"completedAt"       keel:"completed_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (x *Export) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		x,
		validation.Field(&x.RequestedByUserID,
			validation.Required.Error("An export must be requested by a person"),
		),
		validation.Field(&x.Kind,
			validation.Required.Error("Choose a timeframe or conversations to export"),
			domainvalidation.ValidEnum[ExportKind]("Choose a timeframe or conversations to export"),
		),
		validation.Field(&x.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ExportStatus]("Status is invalid"),
		),
	))

	switch x.Kind {
	case ExportKindFilter:
		if x.Filter == nil {
			multiErr.Add("filter", errortypes.ErrRequired, "Choose a timeframe to export")
		} else {
			x.Filter.Validate("filter", multiErr)
		}
	case ExportKindSelection:
		if len(x.ThreadIDs) == 0 {
			multiErr.Add(
				"conversationIds",
				errortypes.ErrRequired,
				"Select at least one conversation to export",
			)
		}
	}

	if x.ConversationCount > MaxExportConversations {
		multiErr.Add(
			"filter",
			errortypes.ErrInvalid,
			"One export can hold at most {0} conversations. Narrow the timeframe, the person or the agent.",
			MaxExportConversations,
		)
	}
}

func (x *Export) Downloadable(now int64) bool {
	if x.Status != ExportStatusSucceeded || x.ArtifactKey == "" {
		return false
	}

	return x.ArtifactExpiresAt == nil || *x.ArtifactExpiresAt > now
}

func (x *Export) GetID() pulid.ID {
	return x.ID
}

func (x *Export) GetCreatedAt() int64 {
	return x.CreatedAt
}

func (x *Export) GetOrganizationID() pulid.ID {
	return x.OrganizationID
}

func (x *Export) GetBusinessUnitID() pulid.ID {
	return x.BusinessUnitID
}

func (x *Export) GetTableName() string {
	return "desk_transcript_exports"
}

func (x *Export) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dtex",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "kind", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (x *Export) BeforeCreate(_ context.Context) error {
	if x.Status == "" {
		x.Status = ExportStatusPending
	}
	if x.Coverage == nil {
		x.Coverage = &Coverage{Kind: x.Kind}
	}
	if x.Coverage.Origins == nil {
		x.Coverage.Origins = []conversation.ThreadOrigin{}
	}
	if x.Coverage.OwnerNames == nil {
		x.Coverage.OwnerNames = []string{}
	}

	return nil
}
