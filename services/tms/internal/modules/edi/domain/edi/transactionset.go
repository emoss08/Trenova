package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type EDITransactionSet struct {
	bun.BaseModel `json:"-" bun:"table:edi_transaction_sets,alias:ets"`
	keel.Table    `keel:"edi_transaction_sets"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:edits_"`
	Standard       EDIStandard    `json:"standard"       bun:"standard,type:edi_standard_enum,notnull" keel:"standard"`
	Code           TransactionSet `json:"code"           bun:"code,type:VARCHAR(20),notnull" keel:"code"`
	Name           string         `json:"name"           bun:"name,type:VARCHAR(200),notnull" keel:"name"`
	Description    string         `json:"description"    bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	DefaultVersion string         `json:"defaultVersion" bun:"default_version,type:VARCHAR(20),notnull" keel:"default_version"`
	Status         DocumentStatus `json:"status"         bun:"status,type:edi_document_status_enum,notnull" keel:"status"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (s *EDITransactionSet) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("edits_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}

type EDITransactionLoopDefinition struct {
	bun.BaseModel `json:"-" bun:"table:edi_transaction_loop_definitions,alias:etld"`
	keel.Table    `keel:"edi_transaction_loop_definitions"`

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:ediloop_"`
	TransactionSetID pulid.ID          `json:"transactionSetId" bun:"transaction_set_id,type:VARCHAR(100),notnull" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction"        bun:"direction,type:edi_document_direction_enum,notnull" keel:"direction"`
	X12Version       string            `json:"x12Version"       bun:"x12_version,type:VARCHAR(20),notnull" keel:"x12_version"`
	LoopID           string            `json:"loopId"           bun:"loop_id,type:VARCHAR(50),notnull" keel:"loop_id"`
	Name             string            `json:"name"             bun:"name,type:VARCHAR(200),notnull" keel:"name"`
	ParentLoopID     string            `json:"parentLoopId"     bun:"parent_loop_id,type:VARCHAR(50),nullzero" keel:"parent_loop_id,nullzero"`
	Sequence         int64             `json:"sequence"         bun:"sequence,type:BIGINT,notnull" keel:"sequence"`
	RepeatPath       string            `json:"repeatPath"       bun:"repeat_path,type:TEXT,nullzero" keel:"repeat_path,nullzero"`
	UsageNotes       string            `json:"usageNotes"       bun:"usage_notes,type:TEXT,nullzero" keel:"usage_notes,nullzero"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (d *EDITransactionLoopDefinition) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("ediloop_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}
	return nil
}

type EDITransactionSegmentDefinition struct {
	bun.BaseModel `json:"-" bun:"table:edi_transaction_segment_definitions,alias:etsd"`
	keel.Table    `keel:"edi_transaction_segment_definitions"`

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:edisegd_"`
	TransactionSetID pulid.ID          `json:"transactionSetId" bun:"transaction_set_id,type:VARCHAR(100),notnull" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction"        bun:"direction,type:edi_document_direction_enum,notnull" keel:"direction"`
	X12Version       string            `json:"x12Version"       bun:"x12_version,type:VARCHAR(20),notnull" keel:"x12_version"`
	SegmentID        string            `json:"segmentId"        bun:"segment_id,type:VARCHAR(10),notnull" keel:"segment_id"`
	Name             string            `json:"name"             bun:"name,type:VARCHAR(200),notnull" keel:"name"`
	LoopID           string            `json:"loopId"           bun:"loop_id,type:VARCHAR(50),nullzero" keel:"loop_id,nullzero"`
	Sequence         int64             `json:"sequence"         bun:"sequence,type:BIGINT,notnull" keel:"sequence"`
	Required         bool              `json:"required"         bun:"required,type:BOOLEAN,notnull" keel:"required"`
	MaxUse           int64             `json:"maxUse"           bun:"max_use,type:BIGINT,notnull,default:1" keel:"max_use"`
	RepeatPath       string            `json:"repeatPath"       bun:"repeat_path,type:TEXT,nullzero" keel:"repeat_path,nullzero"`
	UsageNotes       string            `json:"usageNotes"       bun:"usage_notes,type:TEXT,nullzero" keel:"usage_notes,nullzero"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (d *EDITransactionSegmentDefinition) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("edisegd_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}
	return nil
}

type EDITransactionElementDefinition struct {
	bun.BaseModel `json:"-" bun:"table:edi_transaction_element_definitions,alias:eted"`
	keel.Table    `keel:"edi_transaction_element_definitions"`

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:edielemd_"`
	TransactionSetID pulid.ID          `json:"transactionSetId" bun:"transaction_set_id,type:VARCHAR(100),notnull" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction"        bun:"direction,type:edi_document_direction_enum,notnull" keel:"direction"`
	X12Version       string            `json:"x12Version"       bun:"x12_version,type:VARCHAR(20),notnull" keel:"x12_version"`
	SegmentID        string            `json:"segmentId"        bun:"segment_id,type:VARCHAR(10),notnull" keel:"segment_id"`
	Position         int               `json:"position"         bun:"position,type:INTEGER,notnull" keel:"position"`
	ElementID        string            `json:"elementId"        bun:"element_id,type:VARCHAR(20),nullzero" keel:"element_id,nullzero"`
	Name             string            `json:"name"             bun:"name,type:VARCHAR(200),notnull" keel:"name"`
	Required         bool              `json:"required"         bun:"required,type:BOOLEAN,notnull" keel:"required"`
	MinLength        int               `json:"minLength"        bun:"min_length,type:INTEGER,notnull" keel:"min_length"`
	MaxLength        int               `json:"maxLength"        bun:"max_length,type:INTEGER,notnull" keel:"max_length"`
	CodeListID       pulid.ID          `json:"codeListId"       bun:"code_list_id,type:VARCHAR(100),nullzero" keel:"code_list_id,nullzero"`
	UsageNotes       string            `json:"usageNotes"       bun:"usage_notes,type:TEXT,nullzero" keel:"usage_notes,nullzero"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (d *EDITransactionElementDefinition) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("edielemd_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}
	return nil
}

type EDICodeListDefinition struct {
	bun.BaseModel `json:"-" bun:"table:edi_code_list_definitions,alias:ecld"`
	keel.Table    `keel:"edi_code_list_definitions"`

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:edicodel_"`
	TransactionSetID pulid.ID          `json:"transactionSetId" bun:"transaction_set_id,type:VARCHAR(100),notnull" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction"        bun:"direction,type:edi_document_direction_enum,notnull" keel:"direction"`
	X12Version       string            `json:"x12Version"       bun:"x12_version,type:VARCHAR(20),notnull" keel:"x12_version"`
	ElementID        string            `json:"elementId"        bun:"element_id,type:VARCHAR(20),notnull" keel:"element_id"`
	Code             string            `json:"code"             bun:"code,type:VARCHAR(40),notnull" keel:"code"`
	Description      string            `json:"description"      bun:"description,type:TEXT,notnull" keel:"description"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (d *EDICodeListDefinition) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("edicodel_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}
	return nil
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
