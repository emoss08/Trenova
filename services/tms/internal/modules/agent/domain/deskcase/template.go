package deskcase

import (
	"context"
	"errors"
	"database/sql/driver"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// ItemMode is how one step of a checklist counts. A required step blocks the
// record from being ready and can be the next step; an optional one is shown
// and ticked but never blocks; one that is off is not shown.
type ItemMode string

const (
	ModeRequired = ItemMode("Required")
	ModeOptional = ItemMode("Optional")
	ModeOff      = ItemMode("Off")
)

func (m ItemMode) IsValid() bool {
	switch m {
	case ModeRequired, ModeOptional, ModeOff:
		return true
	default:
		return false
	}
}

// CustomCheck is what ticks a step the organization added: a person, on the
// case, or a document of a type being on file for the shipment.
type CustomCheck string

const (
	CheckManual   = CustomCheck("Manual")
	CheckDocument = CustomCheck("Document")
)

func (c CustomCheck) IsValid() bool {
	switch c {
	case CheckManual, CheckDocument:
		return true
	default:
		return false
	}
}

func (k ChecklistKind) IsValid() bool {
	switch k {
	case ChecklistReadyToBill, ChecklistReadyToClose:
		return true
	default:
		return false
	}
}

func AllChecklistKinds() []ChecklistKind {
	return []ChecklistKind{ChecklistReadyToBill, ChecklistReadyToClose}
}

const (
	customKeyPrefix   = "custom:"
	maxCustomItems    = 12
	maxCustomLabel    = 80
	maxCustomStep     = 40
	maxCustomPrompt   = 500
	maxTemplateItemsN = 32
)

var customKeyPattern = regexp.MustCompile(`^custom:[a-z0-9]{6,32}$`)

// IsCustom reports a step the organization added rather than one Trenova
// defines.
func (k ItemKey) IsCustom() bool {
	return strings.HasPrefix(string(k), customKeyPrefix)
}

// builtinItem is a step Trenova defines: the kind of checklist it is on and
// whether the organization may set how it counts. A step that is locked
// follows a rule kept elsewhere (the billing profile's documents, rate
// validation, the billing holds), or is one nothing can be billed or closed
// without; it can still be moved.
type builtinItem struct {
	key    ItemKey
	kind   ChecklistKind
	locked bool
}

var builtinItems = []builtinItem{
	{key: ItemDelivered, kind: ChecklistReadyToBill, locked: true},
	{key: ItemPOD, kind: ChecklistReadyToBill, locked: true},
	{key: ItemPaperwork, kind: ChecklistReadyToBill, locked: true},
	{key: ItemRateConfirmation, kind: ChecklistReadyToBill, locked: true},
	{key: ItemCarrierRateCon, kind: ChecklistReadyToBill},
	{key: ItemAccessorials, kind: ChecklistReadyToBill},
	{key: ItemCustomerNotified, kind: ChecklistReadyToBill},
	{key: ItemBillingHolds, kind: ChecklistReadyToBill, locked: true},
	{key: ItemPosted, kind: ChecklistReadyToClose, locked: true},
	{key: ItemSent, kind: ChecklistReadyToClose},
	{key: ItemDispute, kind: ChecklistReadyToClose, locked: true},
	{key: ItemPaid, kind: ChecklistReadyToClose, locked: true},
}

func builtinOf(kind ChecklistKind, key ItemKey) (builtinItem, bool) {
	for _, item := range builtinItems {
		if item.kind == kind && item.key == key {
			return item, true
		}
	}

	return builtinItem{}, false
}

// LockedKeys are the built-in steps of a kind whose mode the organization
// cannot set, for the settings screen to say so.
func LockedKeys(kind ChecklistKind) []ItemKey {
	out := make([]ItemKey, 0, len(builtinItems))
	for _, item := range builtinItems {
		if item.kind == kind && item.locked {
			out = append(out, item.key)
		}
	}

	return out
}

// IsLocked reports a built-in step whose mode the organization cannot set.
func IsLocked(kind ChecklistKind, key ItemKey) bool {
	item, ok := builtinOf(kind, key)
	return ok && item.locked
}

// CustomItem is a step the organization added: what it is called, what ticks
// it, and what the next step asks the case's agent when it is the one that
// blocks.
type CustomItem struct {
	Label          string      `json:"label"`
	Check          CustomCheck `json:"check"`
	DocumentTypeID pulid.ID    `json:"documentTypeId,omitempty"`
	StepLabel      string      `json:"stepLabel,omitempty"`
	Prompt         string      `json:"prompt,omitempty"`
}

// TemplateItem is one step of a template, in the order the case shows it.
type TemplateItem struct {
	Key    ItemKey     `json:"key"`
	Mode   ItemMode    `json:"mode"`
	Custom *CustomItem `json:"custom,omitempty"`
}

// TemplateItems is stored as JSON in one column; a template is read whole
// and written whole.
type TemplateItems []TemplateItem

func (t TemplateItems) Value() (driver.Value, error) {
	return marshalItems(t)
}

func (t *TemplateItems) Scan(src any) error {
	return unmarshalItems(src, t)
}

// ChecklistTemplate is how an organization, or one customer of it, wants a kind of
// checklist: which steps, in what order, and how each counts. A customer's
// template replaces the organization's for that customer; with neither, the
// default applies.
type ChecklistTemplate struct {
	bun.BaseModel `bun:"table:case_checklist_templates,alias:cct" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`

	Kind       ChecklistKind `json:"kind"       bun:"kind,type:VARCHAR(30),notnull"`
	CustomerID pulid.ID      `json:"customerId" bun:"customer_id,type:VARCHAR(100),nullzero"`
	Items      TemplateItems `json:"items"      bun:"items,type:JSONB,notnull"`

	UpdatedByID pulid.ID `json:"updatedById" bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version     int64    `json:"version"     bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt   int64    `json:"createdAt"   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt   int64    `json:"updatedAt"   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// CustomerName is read with the template for the settings list; never
	// stored.
	CustomerName string `json:"customerName,omitempty" bun:"customer_name,scanonly"`
}

func (t *ChecklistTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("cct_")
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

// DefaultItems is a kind's steps in Trenova's own order, every one
// required.
func DefaultItems(kind ChecklistKind) TemplateItems {
	out := make(TemplateItems, 0, len(builtinItems))
	for _, item := range builtinItems {
		if item.kind == kind {
			out = append(out, TemplateItem{Key: item.key, Mode: ModeRequired})
		}
	}

	return out
}

// Normalize brings stored steps up to date with the steps Trenova defines:
// a built-in step added since the template was saved joins it where the
// default order puts it, required; one Trenova no longer defines leaves.
// Locked steps are always required.
func Normalize(kind ChecklistKind, items TemplateItems) TemplateItems {
	out := make(TemplateItems, 0, len(items)+len(builtinItems))
	seen := make(map[ItemKey]struct{}, len(items))
	for _, item := range items {
		if item.Key.IsCustom() {
			if item.Custom != nil {
				out = append(out, item)
				seen[item.Key] = struct{}{}
			}
			continue
		}
		builtin, ok := builtinOf(kind, item.Key)
		if !ok {
			continue
		}
		if builtin.locked {
			item.Mode = ModeRequired
		}
		out = append(out, item)
		seen[item.Key] = struct{}{}
	}

	defaults := DefaultItems(kind)
	for idx, item := range defaults {
		if _, ok := seen[item.Key]; ok {
			continue
		}
		at := insertionPoint(out, defaults[:idx])
		out = slices.Insert(out, at, item)
		seen[item.Key] = struct{}{}
	}

	return out
}

// insertionPoint is just after the last of the steps that come before a
// missing one in the default order, or the top when none of them is there.
func insertionPoint(items TemplateItems, before TemplateItems) int {
	at := 0
	for _, prior := range before {
		if idx := slices.IndexFunc(items, func(item TemplateItem) bool {
			return item.Key == prior.Key
		}); idx >= 0 && idx+1 > at {
			at = idx + 1
		}
	}

	return at
}

// Tickable reports a step a person can tick on a case: one the
// organization added, checked by a person, and not turned off.
func Tickable(kind ChecklistKind, template TemplateItems, key ItemKey) bool {
	if template == nil {
		return false
	}
	for _, item := range Normalize(kind, template) {
		if item.Key == key {
			return item.Mode != ModeOff && item.Custom != nil && item.Custom.Check == CheckManual
		}
	}

	return false
}

// Validate checks a template as a person sent it: every built-in step of
// its kind once and no other, locked steps left required, and each added
// step named, checked by something that can tick it, and not too long.
func (t *ChecklistTemplate) Validate(multiErr *errortypes.MultiError) {
	if !t.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Choose ready to bill or ready to close")
		return
	}
	if len(t.Items) > maxTemplateItemsN {
		multiErr.Add("items", errortypes.ErrInvalid, "A checklist can have at most 32 steps")
		return
	}

	seen := make(map[ItemKey]struct{}, len(t.Items))
	customs := 0
	for idx, item := range t.Items {
		field := fmt.Sprintf("items[%d]", idx)
		if _, dup := seen[item.Key]; dup {
			multiErr.Add(field+".key", errortypes.ErrDuplicate, "This step is on the checklist twice")
			continue
		}
		seen[item.Key] = struct{}{}

		if !item.Mode.IsValid() {
			multiErr.Add(field+".mode", errortypes.ErrInvalid, "Choose required, optional or off")
		}

		if item.Key.IsCustom() {
			customs++
			validateCustom(t.Kind, field, item, multiErr)
			continue
		}

		builtin, ok := builtinOf(t.Kind, item.Key)
		switch {
		case !ok:
			multiErr.Add(field+".key", errortypes.ErrInvalid, "This step is not part of this checklist")
		case builtin.locked && item.Mode != ModeRequired:
			multiErr.Add(field+".mode", errortypes.ErrInvalid,
				"This step follows a rule set elsewhere and stays required")
		case item.Custom != nil:
			multiErr.Add(field+".custom", errortypes.ErrInvalid,
				"A built-in step cannot be renamed")
		}
	}

	if customs > maxCustomItems {
		multiErr.Add("items", errortypes.ErrInvalid, "A checklist can have at most 12 added steps")
	}
	for _, item := range DefaultItems(t.Kind) {
		if _, ok := seen[item.Key]; !ok {
			multiErr.Add("items", errortypes.ErrRequired,
				"Every built-in step must stay on the checklist; turn it off instead")
			break
		}
	}
}

func validateCustom(kind ChecklistKind, field string, item TemplateItem, multiErr *errortypes.MultiError) {
	if !customKeyPattern.MatchString(string(item.Key)) {
		multiErr.Add(field+".key", errortypes.ErrInvalid, "The step's key is invalid")
	}
	custom := item.Custom
	if custom == nil {
		multiErr.Add(field+".custom", errortypes.ErrRequired, "Name the step and say what ticks it")
		return
	}

	label := strings.TrimSpace(custom.Label)
	switch {
	case label == "":
		multiErr.Add(field+".custom.label", errortypes.ErrRequired, "Name the step")
	case utf8.RuneCountInString(label) > maxCustomLabel:
		multiErr.Add(field+".custom.label", errortypes.ErrInvalid,
			"A step's name can be at most 80 characters")
	}
	if utf8.RuneCountInString(custom.StepLabel) > maxCustomStep {
		multiErr.Add(field+".custom.stepLabel", errortypes.ErrInvalid,
			"A step's button can say at most 40 characters")
	}
	if utf8.RuneCountInString(custom.Prompt) > maxCustomPrompt {
		multiErr.Add(field+".custom.prompt", errortypes.ErrInvalid,
			"What the step asks the agent can be at most 500 characters")
	}

	switch {
	case !custom.Check.IsValid():
		multiErr.Add(field+".custom.check", errortypes.ErrInvalid,
			"Choose whether a person ticks it or a document does")
	case custom.Check == CheckDocument && kind != ChecklistReadyToBill:
		multiErr.Add(field+".custom.check", errortypes.ErrInvalid,
			"Only a shipment's checklist can be ticked by a document")
	case custom.Check == CheckDocument && custom.DocumentTypeID.IsNil():
		multiErr.Add(field+".custom.documentTypeId", errortypes.ErrRequired,
			"Choose the document type that ticks it")
	case custom.Check == CheckManual && custom.DocumentTypeID.IsNotNil():
		multiErr.Add(field+".custom.documentTypeId", errortypes.ErrInvalid,
			"A step a person ticks names no document type")
	}
}

// Clean trims what a person typed into the added steps.
func (t *ChecklistTemplate) Clean() {
	for idx := range t.Items {
		if custom := t.Items[idx].Custom; custom != nil {
			custom.Label = strings.TrimSpace(custom.Label)
			custom.StepLabel = strings.TrimSpace(custom.StepLabel)
			custom.Prompt = strings.TrimSpace(custom.Prompt)
		}
	}
}

// DocumentTypeIDs are the document types the template's added steps wait
// on, for checking they are the organization's.
func (t *ChecklistTemplate) DocumentTypeIDs() []pulid.ID {
	out := make([]pulid.ID, 0)
	for _, item := range t.Items {
		if item.Custom != nil && item.Custom.Check == CheckDocument &&
			!slices.Contains(out, item.Custom.DocumentTypeID) {
			out = append(out, item.Custom.DocumentTypeID)
		}
	}

	return out
}

// ChecklistTick is a person's tick on an added step of one record's checklist. It
// belongs to the record, so everyone working a case about it sees it.
type ChecklistTick struct {
	bun.BaseModel `bun:"table:case_checklist_ticks,alias:cctk" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`

	SubjectType string   `json:"subjectType" bun:"subject_type,type:VARCHAR(50),notnull"`
	SubjectID   pulid.ID `json:"subjectId"   bun:"subject_id,type:VARCHAR(100),notnull"`
	ItemKey     ItemKey  `json:"itemKey"     bun:"item_key,type:VARCHAR(50),notnull"`
	TickedByID  pulid.ID `json:"tickedById"  bun:"ticked_by_id,type:VARCHAR(100),notnull"`
	TickedAt    int64    `json:"tickedAt"    bun:"ticked_at,type:BIGINT,notnull"`

	// TickedByName is read with the tick; never stored.
	TickedByName string `json:"tickedByName,omitempty" bun:"ticked_by_name,scanonly"`
}

func (t *ChecklistTick) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok && t.ID.IsNil() {
		t.ID = pulid.MustNew("cctk_")
	}

	return nil
}

func marshalItems(items TemplateItems) (driver.Value, error) {
	if items == nil {
		items = TemplateItems{}
	}
	encoded, err := sonic.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("encode checklist steps: %w", err)
	}

	return string(encoded), nil
}

func unmarshalItems(src any, out *TemplateItems) error {
	var raw []byte
	switch value := src.(type) {
	case nil:
		*out = TemplateItems{}
		return nil
	case []byte:
		raw = value
	case string:
		raw = []byte(value)
	default:
		return errors.New("checklist steps are not JSON")
	}

	if err := sonic.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode checklist steps: %w", err)
	}

	return nil
}
