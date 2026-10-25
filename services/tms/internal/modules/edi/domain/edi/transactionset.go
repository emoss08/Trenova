package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type EDITransactionSet struct {
	keel.Table `keel:"edi_transaction_sets"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:edits_"`
	Standard       EDIStandard    `json:"standard"       keel:"standard"`
	Code           TransactionSet `json:"code"           keel:"code"`
	Name           string         `json:"name"           keel:"name"`
	Description    string         `json:"description"    keel:"description,nullzero"`
	DefaultVersion string         `json:"defaultVersion" keel:"default_version"`
	Status         DocumentStatus `json:"status"         keel:"status"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64          `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

type EDITransactionLoopDefinition struct {
	keel.Table `keel:"edi_transaction_loop_definitions"`

	ID               pulid.ID          `json:"id"               keel:"id,pk,autoid:ediloop_"`
	TransactionSetID pulid.ID          `json:"transactionSetId" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction"        keel:"direction"`
	X12Version       string            `json:"x12Version"       keel:"x12_version"`
	LoopID           string            `json:"loopId"           keel:"loop_id"`
	Name             string            `json:"name"             keel:"name"`
	ParentLoopID     string            `json:"parentLoopId"     keel:"parent_loop_id,nullzero"`
	Sequence         int64             `json:"sequence"         keel:"sequence"`
	RepeatPath       string            `json:"repeatPath"       keel:"repeat_path,nullzero"`
	UsageNotes       string            `json:"usageNotes"       keel:"usage_notes,nullzero"`
	CreatedAt        int64             `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        keel:"updated_at,autoupdate"`
}

type EDITransactionSegmentDefinition struct {
	keel.Table `keel:"edi_transaction_segment_definitions"`

	ID               pulid.ID          `json:"id"               keel:"id,pk,autoid:edisegd_"`
	TransactionSetID pulid.ID          `json:"transactionSetId" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction"        keel:"direction"`
	X12Version       string            `json:"x12Version"       keel:"x12_version"`
	SegmentID        string            `json:"segmentId"        keel:"segment_id"`
	Name             string            `json:"name"             keel:"name"`
	LoopID           string            `json:"loopId"           keel:"loop_id,nullzero"`
	Sequence         int64             `json:"sequence"         keel:"sequence"`
	Required         bool              `json:"required"         keel:"required"`
	MaxUse           int64             `json:"maxUse"           keel:"max_use"`
	RepeatPath       string            `json:"repeatPath"       keel:"repeat_path,nullzero"`
	UsageNotes       string            `json:"usageNotes"       keel:"usage_notes,nullzero"`
	CreatedAt        int64             `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        keel:"updated_at,autoupdate"`
}

func (d *EDITransactionSegmentDefinition) BeforeCreate(_ context.Context) error {
	if d.MaxUse == 0 {
		d.MaxUse = 1
	}

	return nil
}

type EDITransactionElementDefinition struct {
	keel.Table `keel:"edi_transaction_element_definitions"`

	ID               pulid.ID          `json:"id"               keel:"id,pk,autoid:edielemd_"`
	TransactionSetID pulid.ID          `json:"transactionSetId" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction"        keel:"direction"`
	X12Version       string            `json:"x12Version"       keel:"x12_version"`
	SegmentID        string            `json:"segmentId"        keel:"segment_id"`
	Position         int               `json:"position"         keel:"position"`
	ElementID        string            `json:"elementId"        keel:"element_id,nullzero"`
	Name             string            `json:"name"             keel:"name"`
	Required         bool              `json:"required"         keel:"required"`
	MinLength        int               `json:"minLength"        keel:"min_length"`
	MaxLength        int               `json:"maxLength"        keel:"max_length"`
	CodeListID       pulid.ID          `json:"codeListId"       keel:"code_list_id,nullzero"`
	UsageNotes       string            `json:"usageNotes"       keel:"usage_notes,nullzero"`
	CreatedAt        int64             `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        keel:"updated_at,autoupdate"`
}

type EDICodeListDefinition struct {
	keel.Table `keel:"edi_code_list_definitions"`

	ID               pulid.ID          `json:"id"               keel:"id,pk,autoid:edicodel_"`
	TransactionSetID pulid.ID          `json:"transactionSetId" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction"        keel:"direction"`
	X12Version       string            `json:"x12Version"       keel:"x12_version"`
	ElementID        string            `json:"elementId"        keel:"element_id"`
	Code             string            `json:"code"             keel:"code"`
	Description      string            `json:"description"      keel:"description"`
	CreatedAt        int64             `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        keel:"updated_at,autoupdate"`
}

func (s *EDITransactionSet) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.Standard,
			validation.Required.Error("Standard is required"),
			domainvalidation.ValidEnum[EDIStandard]("Standard is invalid"),
		),
		validation.Field(&s.Code,
			validation.Required.Error("Code is required"),
			domainvalidation.ValidEnum[TransactionSet]("Code is invalid"),
		),
		validation.Field(&s.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&s.DefaultVersion,
			validation.Required.Error("Default version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Default version cannot be longer than 20 characters"),
		),
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DocumentStatus]("Status is invalid"),
		),
	))
}

func (l *EDITransactionLoopDefinition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(&l.TransactionSetID,
			validation.Required.Error("Transaction set is required"),
		),
		validation.Field(&l.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&l.X12Version,
			validation.Required.Error("X12 version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("X12 version cannot be longer than 20 characters"),
		),
		validation.Field(&l.LoopID,
			validation.Required.Error("Loop is required"),
			validation.Length(1, maxLoopIDLength).
				Error("Loop cannot be longer than 50 characters"),
		),
		// A loop nested inside itself would never terminate when the template
		// walks the definition tree.
		validation.Field(&l.ParentLoopID,
			validation.Length(0, maxLoopIDLength).
				Error("Parent loop cannot be longer than 50 characters"),
			validation.NotIn(l.LoopID).Error("A loop cannot be its own parent"),
		),
		validation.Field(&l.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&l.Sequence,
			validation.Min(int64(0)).Error("Sequence cannot be negative"),
		),
	))
}

func (s *EDITransactionSegmentDefinition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.TransactionSetID,
			validation.Required.Error("Transaction set is required"),
		),
		validation.Field(&s.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&s.X12Version,
			validation.Required.Error("X12 version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("X12 version cannot be longer than 20 characters"),
		),
		validation.Field(&s.SegmentID,
			validation.Required.Error("Segment is required"),
			validation.Length(1, maxSegmentIDLength).
				Error("Segment cannot be longer than 10 characters"),
		),
		validation.Field(&s.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&s.LoopID,
			validation.Length(0, maxLoopIDLength).
				Error("Loop cannot be longer than 50 characters"),
		),
		validation.Field(&s.Sequence,
			validation.Min(int64(0)).Error("Sequence cannot be negative"),
		),
		validation.Field(&s.MaxUse,
			validation.Required.Error("Max use is required"),
			validation.Min(int64(1)).Error("Max use must be at least one"),
		),
	))
}

func (e *EDITransactionElementDefinition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.TransactionSetID,
			validation.Required.Error("Transaction set is required"),
		),
		validation.Field(&e.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&e.X12Version,
			validation.Required.Error("X12 version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("X12 version cannot be longer than 20 characters"),
		),
		validation.Field(&e.SegmentID,
			validation.Required.Error("Segment is required"),
			validation.Length(1, maxSegmentIDLength).
				Error("Segment cannot be longer than 10 characters"),
		),
		// Element positions are one-based in X12; a zero would address the
		// segment id itself.
		validation.Field(&e.Position,
			validation.Required.Error("Position is required"),
			validation.Min(1).Error("Position must be at least one"),
		),
		validation.Field(&e.ElementID,
			validation.Length(0, maxEDICodeLength).
				Error("Element cannot be longer than 20 characters"),
		),
		validation.Field(&e.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&e.MinLength,
			validation.Min(0).Error("Minimum length cannot be negative"),
		),
		validation.Field(&e.MaxLength,
			validation.Min(e.MinLength).Error("Maximum length cannot be less than the minimum"),
		),
	))
}

func (c *EDICodeListDefinition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.TransactionSetID,
			validation.Required.Error("Transaction set is required"),
		),
		validation.Field(&c.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&c.X12Version,
			validation.Required.Error("X12 version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("X12 version cannot be longer than 20 characters"),
		),
		validation.Field(&c.ElementID,
			validation.Required.Error("Element is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Element cannot be longer than 20 characters"),
		),
		validation.Field(&c.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, maxCodeListCodeLength).
				Error("Code cannot be longer than 40 characters"),
		),
		validation.Field(&c.Description, validation.Required.Error("Description is required")),
	))
}

const maxCodeListCodeLength = 40
