package worker

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidChecklistKind       = errors.New("invalid checklist kind")
	ErrInvalidChecklistTrigger    = errors.New("invalid checklist trigger")
	ErrInvalidChecklistItemKind   = errors.New("invalid checklist item kind")
	ErrInvalidChecklistOwner      = errors.New("invalid checklist owner")
	ErrInvalidChecklistStatus     = errors.New("invalid checklist status")
	ErrInvalidChecklistItemStatus = errors.New("invalid checklist item status")
)

type ChecklistKind string

const (
	ChecklistKindOnboarding  = ChecklistKind("Onboarding")
	ChecklistKindOffboarding = ChecklistKind("Offboarding")
	ChecklistKindCustom      = ChecklistKind("Custom")
)

func (k ChecklistKind) String() string { return string(k) }

func (k ChecklistKind) IsValid() bool {
	switch k {
	case ChecklistKindOnboarding, ChecklistKindOffboarding, ChecklistKindCustom:
		return true
	default:
		return false
	}
}

type ChecklistTrigger string

const (
	ChecklistTriggerHired      = ChecklistTrigger("Hired")
	ChecklistTriggerRehired    = ChecklistTrigger("Rehired")
	ChecklistTriggerTerminated = ChecklistTrigger("Terminated")
	ChecklistTriggerManual     = ChecklistTrigger("Manual")
)

func (t ChecklistTrigger) String() string { return string(t) }

func (t ChecklistTrigger) IsValid() bool {
	switch t {
	case ChecklistTriggerHired, ChecklistTriggerRehired, ChecklistTriggerTerminated,
		ChecklistTriggerManual:
		return true
	default:
		return false
	}
}

// ChecklistKindClosedByEvent names the kind of open checklist an employment
// event makes moot: a termination ends an unfinished onboarding, and a hire
// or rehire ends an unfinished offboarding. Other events close nothing.
func ChecklistKindClosedByEvent(kind EmploymentEventKind) (ChecklistKind, bool) {
	switch kind {
	case EmploymentEventTerminated:
		return ChecklistKindOnboarding, true
	case EmploymentEventHired, EmploymentEventRehired:
		return ChecklistKindOffboarding, true
	default:
		return "", false
	}
}

// ChecklistTriggerForEvent maps an employment event to the trigger that spawns
// a checklist, or false when the event never spawns one.
func ChecklistTriggerForEvent(kind EmploymentEventKind) (ChecklistTrigger, bool) {
	switch kind {
	case EmploymentEventHired:
		return ChecklistTriggerHired, true
	case EmploymentEventRehired:
		return ChecklistTriggerRehired, true
	case EmploymentEventTerminated:
		return ChecklistTriggerTerminated, true
	default:
		return "", false
	}
}

type ChecklistItemKind string

const (
	ChecklistItemDocument     = ChecklistItemKind("Document")
	ChecklistItemCredential   = ChecklistItemKind("Credential")
	ChecklistItemTask         = ChecklistItemKind("Task")
	ChecklistItemEquipment    = ChecklistItemKind("Equipment")
	ChecklistItemPortalAccess = ChecklistItemKind("PortalAccess")
)

func (k ChecklistItemKind) String() string { return string(k) }

func (k ChecklistItemKind) IsValid() bool {
	switch k {
	case ChecklistItemDocument, ChecklistItemCredential, ChecklistItemTask,
		ChecklistItemEquipment, ChecklistItemPortalAccess:
		return true
	default:
		return false
	}
}

// AutoSatisfiable kinds complete themselves when the evidence exists.
func (k ChecklistItemKind) AutoSatisfiable() bool {
	return k == ChecklistItemDocument || k == ChecklistItemCredential ||
		k == ChecklistItemPortalAccess
}

type ChecklistOwner string

const (
	ChecklistOwnerHR       = ChecklistOwner("HR")
	ChecklistOwnerSafety   = ChecklistOwner("Safety")
	ChecklistOwnerDispatch = ChecklistOwner("Dispatch")
	ChecklistOwnerPayroll  = ChecklistOwner("Payroll")
	ChecklistOwnerIT       = ChecklistOwner("IT")
	ChecklistOwnerFleet    = ChecklistOwner("Fleet")
)

func (o ChecklistOwner) String() string { return string(o) }

func (o ChecklistOwner) IsValid() bool {
	switch o {
	case ChecklistOwnerHR, ChecklistOwnerSafety, ChecklistOwnerDispatch, ChecklistOwnerPayroll,
		ChecklistOwnerIT, ChecklistOwnerFleet:
		return true
	default:
		return false
	}
}

type ChecklistStatus string

const (
	ChecklistStatusOpen      = ChecklistStatus("Open")
	ChecklistStatusCompleted = ChecklistStatus("Completed")
	ChecklistStatusCancelled = ChecklistStatus("Cancelled")
)

func (s ChecklistStatus) String() string { return string(s) }

func (s ChecklistStatus) IsValid() bool {
	switch s {
	case ChecklistStatusOpen, ChecklistStatusCompleted, ChecklistStatusCancelled:
		return true
	default:
		return false
	}
}

type ChecklistItemStatus string

const (
	ChecklistItemPending       = ChecklistItemStatus("Pending")
	ChecklistItemDone          = ChecklistItemStatus("Done")
	ChecklistItemSkipped       = ChecklistItemStatus("Skipped")
	ChecklistItemNotApplicable = ChecklistItemStatus("NotApplicable")
)

func (s ChecklistItemStatus) String() string { return string(s) }

func (s ChecklistItemStatus) IsValid() bool {
	switch s {
	case ChecklistItemPending, ChecklistItemDone, ChecklistItemSkipped, ChecklistItemNotApplicable:
		return true
	default:
		return false
	}
}

// Settled reports whether the item no longer needs attention.
func (s ChecklistItemStatus) Settled() bool { return s != ChecklistItemPending }

var (
	_ bun.BeforeAppendModelHook          = (*WorkerChecklistTemplate)(nil)
	_ domaintypes.PostgresSearchable     = (*WorkerChecklistTemplate)(nil)
	_ pagination.CursorEntity            = (*WorkerChecklistTemplate)(nil)
	_ validationframework.TenantedEntity = (*WorkerChecklistTemplate)(nil)
	_ bun.BeforeAppendModelHook          = (*WorkerChecklistTemplateItem)(nil)
	_ bun.BeforeAppendModelHook          = (*WorkerChecklist)(nil)
	_ validationframework.TenantedEntity = (*WorkerChecklist)(nil)
	_ bun.BeforeAppendModelHook          = (*WorkerChecklistItem)(nil)
)

type WorkerChecklistTemplate struct {
	bun.BaseModel             `bun:"table:worker_checklist_templates,alias:wclt" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                     json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Code           string             `json:"code"           bun:"code,type:VARCHAR(50),notnull"`
	Name           string             `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Description    string             `json:"description"    bun:"description,type:TEXT,nullzero"`
	Kind           ChecklistKind      `json:"kind"           bun:"kind,type:worker_checklist_kind_enum,notnull,default:'Custom'"`
	Trigger        ChecklistTrigger   `json:"trigger"        bun:"trigger,type:worker_checklist_trigger_enum,notnull,default:'Manual'"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	IsDefault      bool               `json:"isDefault"      bun:"is_default,type:BOOLEAN,notnull"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`

	Items []*WorkerChecklistTemplateItem `json:"items,omitempty" bun:"rel:has-many,join:id=template_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (t *WorkerChecklistTemplate) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&t.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[ChecklistKind]("kind must be Onboarding, Offboarding or Custom"),
		),
		validation.Field(&t.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[ChecklistTrigger](
				"trigger must be Hired, Rehired, Terminated or Manual",
			),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
	))

	if t.IsDefault && t.Trigger == ChecklistTriggerManual {
		multiErr.Add(
			"isDefault",
			errortypes.ErrInvalid,
			"A manual checklist cannot be the default; pick the event that should start it",
		)
	}
	if t.IsDefault && t.Status != domaintypes.StatusActive {
		multiErr.Add("isDefault", errortypes.ErrInvalid, "Only an active template can be the default")
	}
	if len(t.Items) == 0 {
		multiErr.Add("items", errortypes.ErrRequired, "Add at least one item")
	}
	for i, item := range t.Items {
		if item == nil {
			continue
		}
		prefix := "items[" + strconv.Itoa(i) + "]"
		item.Validate(multiErr.WithPrefix(prefix))
	}
}

func (t *WorkerChecklistTemplate) NormalizeCode() {
	t.Code = strings.ToUpper(strings.TrimSpace(t.Code))
}

func (t *WorkerChecklistTemplate) GetID() pulid.ID { return t.ID }

func (t *WorkerChecklistTemplate) GetCreatedAt() int64 { return t.CreatedAt }

func (t *WorkerChecklistTemplate) GetOrganizationID() pulid.ID { return t.OrganizationID }

func (t *WorkerChecklistTemplate) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }

func (t *WorkerChecklistTemplate) GetTableName() string { return "worker_checklist_templates" }

func (t *WorkerChecklistTemplate) GetResourceType() string { return "worker_checklist_template" }

func (t *WorkerChecklistTemplate) GetResourceID() string { return t.ID.String() }

func (t *WorkerChecklistTemplate) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "wclt",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (t *WorkerChecklistTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("wclt_")
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

type WorkerChecklistTemplateItem struct {
	bun.BaseModel `bun:"table:worker_checklist_template_items,alias:wclti" json:"-"`

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID          `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID          `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	TemplateID       pulid.ID          `json:"templateId"       bun:"template_id,type:VARCHAR(100),notnull"`
	Label            string            `json:"label"            bun:"label,type:VARCHAR(150),notnull"`
	Description      string            `json:"description"      bun:"description,type:TEXT,nullzero"`
	Kind             ChecklistItemKind `json:"kind"             bun:"kind,type:worker_checklist_item_kind_enum,notnull,default:'Task'"`
	Required         bool              `json:"required"         bun:"required,type:BOOLEAN,notnull"`
	DueOffsetDays    int32             `json:"dueOffsetDays"    bun:"due_offset_days,type:INTEGER,notnull"`
	Owner            ChecklistOwner    `json:"owner"            bun:"owner,type:worker_checklist_owner_enum,notnull,default:'HR'"`
	CredentialTypeID pulid.ID          `json:"credentialTypeId" bun:"credential_type_id,type:VARCHAR(100),nullzero"`
	DocumentTypeID   pulid.ID          `json:"documentTypeId"   bun:"document_type_id,type:VARCHAR(100),nullzero"`
	SortOrder        int32             `json:"sortOrder"        bun:"sort_order,type:INTEGER,notnull"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CredentialType *WorkerCredentialType      `json:"credentialType,omitempty" bun:"rel:belongs-to,join:credential_type_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	DocumentType   *documenttype.DocumentType `json:"documentType,omitempty"   bun:"rel:belongs-to,join:document_type_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (i *WorkerChecklistTemplateItem) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.Label,
			validation.Required.Error("Label is required"),
			validation.Length(1, 150).Error("Label must be between 1 and 150 characters"),
		),
		validation.Field(&i.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[ChecklistItemKind](
				"kind must be Document, Credential, Task, Equipment or PortalAccess",
			),
		),
		validation.Field(&i.Owner,
			validation.Required.Error("Owner is required"),
			domainvalidation.ValidEnum[ChecklistOwner](
				"owner must be HR, Safety, Dispatch, Payroll, IT or Fleet",
			),
		),
		validation.Field(&i.DueOffsetDays,
			validation.Min(int32(0)).Error("Due offset cannot be negative"),
			validation.Max(int32(365)).Error("Due offset cannot exceed 365 days"),
		),
	))

	if i.Kind == ChecklistItemCredential && i.CredentialTypeID.IsNil() {
		multiErr.Add(
			"credentialTypeId",
			errortypes.ErrRequired,
			"Choose which credential this item waits for",
		)
	}
	if i.Kind == ChecklistItemDocument && i.DocumentTypeID.IsNil() {
		multiErr.Add(
			"documentTypeId",
			errortypes.ErrRequired,
			"Choose which document type this item waits for",
		)
	}
}

func (i *WorkerChecklistTemplateItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("wclti_")
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

type WorkerChecklist struct {
	bun.BaseModel `bun:"table:worker_checklists,alias:wcl" json:"-"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID        `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	TemplateID     pulid.ID        `json:"templateId"     bun:"template_id,type:VARCHAR(100),nullzero"`
	Name           string          `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Kind           ChecklistKind   `json:"kind"           bun:"kind,type:worker_checklist_kind_enum,notnull,default:'Custom'"`
	Status         ChecklistStatus `json:"status"         bun:"status,type:worker_checklist_status_enum,notnull,default:'Open'"`
	StartedAt      int64           `json:"startedAt"      bun:"started_at,type:BIGINT,notnull"`
	DueAt          *int64          `json:"dueAt"          bun:"due_at,type:BIGINT,nullzero"`
	CompletedAt    *int64          `json:"completedAt"    bun:"completed_at,type:BIGINT,nullzero"`
	CancelledAt    *int64          `json:"cancelledAt"    bun:"cancelled_at,type:BIGINT,nullzero"`
	CancelReason   string          `json:"cancelReason"   bun:"cancel_reason,type:VARCHAR(255),nullzero"`
	SourceEventID  pulid.ID        `json:"sourceEventId"  bun:"source_event_id,type:VARCHAR(100),nullzero"`
	StartedByID    pulid.ID        `json:"startedById"    bun:"started_by_id,type:VARCHAR(100),nullzero"`
	Version        int64           `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Items     []*WorkerChecklistItem   `json:"items,omitempty"     bun:"rel:has-many,join:id=checklist_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Worker    *Worker                  `json:"worker,omitempty"    bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Template  *WorkerChecklistTemplate `json:"template,omitempty"  bun:"rel:belongs-to,join:template_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	StartedBy *tenant.User             `json:"startedBy,omitempty" bun:"rel:belongs-to,join:started_by_id=id"`
}

func (c *WorkerChecklist) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&c.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[ChecklistKind]("kind must be Onboarding, Offboarding or Custom"),
		),
		validation.Field(&c.StartedAt,
			validation.Required.Error("Start date is required"),
			validation.Min(int64(1)).Error("Start date must be a valid date"),
		),
	))
}

func (c *WorkerChecklist) IsOpen() bool { return c.Status == ChecklistStatusOpen }

func (c *WorkerChecklist) GetID() pulid.ID { return c.ID }

func (c *WorkerChecklist) GetCreatedAt() int64 { return c.CreatedAt }

func (c *WorkerChecklist) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *WorkerChecklist) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *WorkerChecklist) GetTableName() string { return "worker_checklists" }

func (c *WorkerChecklist) GetResourceType() string { return "worker_checklist" }

func (c *WorkerChecklist) GetResourceID() string { return c.ID.String() }

func (c *WorkerChecklist) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("wcl_")
		}
		if c.Status == "" {
			c.Status = ChecklistStatusOpen
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

type WorkerChecklistItem struct {
	bun.BaseModel `bun:"table:worker_checklist_items,alias:wcli" json:"-"`

	ID                   pulid.ID            `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID            `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ChecklistID          pulid.ID            `json:"checklistId"          bun:"checklist_id,type:VARCHAR(100),notnull"`
	TemplateItemID       pulid.ID            `json:"templateItemId"       bun:"template_item_id,type:VARCHAR(100),nullzero"`
	Label                string              `json:"label"                bun:"label,type:VARCHAR(150),notnull"`
	Description          string              `json:"description"          bun:"description,type:TEXT,nullzero"`
	Kind                 ChecklistItemKind   `json:"kind"                 bun:"kind,type:worker_checklist_item_kind_enum,notnull,default:'Task'"`
	Required             bool                `json:"required"             bun:"required,type:BOOLEAN,notnull"`
	Owner                ChecklistOwner      `json:"owner"                bun:"owner,type:worker_checklist_owner_enum,notnull,default:'HR'"`
	DueAt                *int64              `json:"dueAt"                bun:"due_at,type:BIGINT,nullzero"`
	CredentialTypeID     pulid.ID            `json:"credentialTypeId"     bun:"credential_type_id,type:VARCHAR(100),nullzero"`
	DocumentTypeID       pulid.ID            `json:"documentTypeId"       bun:"document_type_id,type:VARCHAR(100),nullzero"`
	Status               ChecklistItemStatus `json:"status"               bun:"status,type:worker_checklist_item_status_enum,notnull,default:'Pending'"`
	CompletedByID        pulid.ID            `json:"completedById"        bun:"completed_by_id,type:VARCHAR(100),nullzero"`
	CompletedAt          *int64              `json:"completedAt"          bun:"completed_at,type:BIGINT,nullzero"`
	AutoCompleted        bool                `json:"autoCompleted"        bun:"auto_completed,type:BOOLEAN,notnull"`
	Note                 string              `json:"note"                 bun:"note,type:VARCHAR(500),nullzero"`
	EvidenceDocumentID   pulid.ID            `json:"evidenceDocumentId"   bun:"evidence_document_id,type:VARCHAR(100),nullzero"`
	EvidenceCredentialID pulid.ID            `json:"evidenceCredentialId" bun:"evidence_credential_id,type:VARCHAR(100),nullzero"`
	SortOrder            int32               `json:"sortOrder"            bun:"sort_order,type:INTEGER,notnull"`
	Version              int64               `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64               `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64               `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CompletedBy        *tenant.User          `json:"completedBy,omitempty"        bun:"rel:belongs-to,join:completed_by_id=id"`
	EvidenceDocument   *document.Document    `json:"evidenceDocument,omitempty"   bun:"rel:belongs-to,join:evidence_document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	EvidenceCredential *WorkerCredential     `json:"evidenceCredential,omitempty" bun:"rel:belongs-to,join:evidence_credential_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	CredentialType     *WorkerCredentialType `json:"credentialType,omitempty"     bun:"rel:belongs-to,join:credential_type_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (i *WorkerChecklistItem) IsOverdue(now int64) bool {
	return i.Status == ChecklistItemPending && i.DueAt != nil && *i.DueAt > 0 && *i.DueAt < now
}

// Blocks reports whether this item keeps the checklist from completing.
func (i *WorkerChecklistItem) Blocks() bool {
	return i.Required && i.Status == ChecklistItemPending
}

func (i *WorkerChecklistItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("wcli_")
		}
		if i.Status == "" {
			i.Status = ChecklistItemPending
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

// ChecklistProgress is the roll-up the UI rings and the DQF readiness rule
// both read from.
type ChecklistProgress struct {
	Total         int
	Settled       int
	RequiredTotal int
	RequiredDone  int
	Overdue       int
	Percent       int
}

func (p ChecklistProgress) Complete() bool {
	return p.RequiredTotal == p.RequiredDone
}

// Progress counts settled items; skipped and not-applicable items count as
// settled so a checklist can close once every required line has an answer.
func (c *WorkerChecklist) Progress(now int64) ChecklistProgress {
	var progress ChecklistProgress
	for _, item := range c.Items {
		if item == nil {
			continue
		}
		progress.Total++
		if item.Status.Settled() {
			progress.Settled++
		}
		if item.Required {
			progress.RequiredTotal++
			if item.Status.Settled() {
				progress.RequiredDone++
			}
		}
		if item.IsOverdue(now) {
			progress.Overdue++
		}
	}
	if progress.Total > 0 {
		progress.Percent = progress.Settled * 100 / progress.Total
	} else {
		progress.Percent = 100
	}
	return progress
}

// Instantiate copies a template onto a worker as an open checklist, resolving
// due dates from the start date.
func (t *WorkerChecklistTemplate) Instantiate(
	workerID pulid.ID,
	startedAt int64,
	startedBy, sourceEventID pulid.ID,
) *WorkerChecklist {
	checklist := &WorkerChecklist{
		OrganizationID: t.OrganizationID,
		BusinessUnitID: t.BusinessUnitID,
		WorkerID:       workerID,
		TemplateID:     t.ID,
		Name:           t.Name,
		Kind:           t.Kind,
		Status:         ChecklistStatusOpen,
		StartedAt:      startedAt,
		SourceEventID:  sourceEventID,
		StartedByID:    startedBy,
		Items:          make([]*WorkerChecklistItem, 0, len(t.Items)),
	}
	var latestDue int64
	for index, source := range t.Items {
		if source == nil {
			continue
		}
		item := &WorkerChecklistItem{
			OrganizationID:   t.OrganizationID,
			BusinessUnitID:   t.BusinessUnitID,
			TemplateItemID:   source.ID,
			Label:            source.Label,
			Description:      source.Description,
			Kind:             source.Kind,
			Required:         source.Required,
			Owner:            source.Owner,
			CredentialTypeID: source.CredentialTypeID,
			DocumentTypeID:   source.DocumentTypeID,
			Status:           ChecklistItemPending,
			SortOrder:        int32(index), //nolint:gosec // item counts are tiny
		}
		if source.DueOffsetDays > 0 {
			due := startedAt + int64(source.DueOffsetDays)*secondsPerDay
			item.DueAt = &due
			if due > latestDue {
				latestDue = due
			}
		}
		checklist.Items = append(checklist.Items, item)
	}
	if latestDue > 0 {
		checklist.DueAt = &latestDue
	}
	return checklist
}

// ChecklistEvidence is what the auto-satisfy pass looks at: the worker's
// active credentials by type, worker documents by type, and portal access.
type ChecklistEvidence struct {
	CredentialsByType map[pulid.ID]*WorkerCredential
	DocumentsByType   map[pulid.ID]*document.Document
	HasPortalAccess   bool
}

// AutoSatisfy marks auto-satisfiable pending items Done when their evidence
// exists and returns the items it changed. PortalAccess reads differently per
// checklist kind: onboarding wants access granted, offboarding wants it gone.
func (c *WorkerChecklist) AutoSatisfy(evidence ChecklistEvidence, now int64) []*WorkerChecklistItem {
	changed := make([]*WorkerChecklistItem, 0, 2)
	for _, item := range c.Items {
		if item == nil || item.Status != ChecklistItemPending || !item.Kind.AutoSatisfiable() {
			continue
		}
		satisfied := false
		switch item.Kind {
		case ChecklistItemCredential:
			if cred, ok := evidence.CredentialsByType[item.CredentialTypeID]; ok && cred != nil &&
				cred.IsActive() && cred.Health(now) != CredentialHealthExpired {
				item.EvidenceCredentialID = cred.ID
				satisfied = true
			}
		case ChecklistItemDocument:
			if doc, ok := evidence.DocumentsByType[item.DocumentTypeID]; ok && doc != nil {
				item.EvidenceDocumentID = doc.ID
				satisfied = true
			}
		case ChecklistItemPortalAccess:
			if c.Kind == ChecklistKindOffboarding {
				satisfied = !evidence.HasPortalAccess
			} else {
				satisfied = evidence.HasPortalAccess
			}
		case ChecklistItemTask, ChecklistItemEquipment:
		}
		if !satisfied {
			continue
		}
		completedAt := now
		item.Status = ChecklistItemDone
		item.CompletedAt = &completedAt
		item.AutoCompleted = true
		changed = append(changed, item)
	}
	return changed
}
