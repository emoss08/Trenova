package edi

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type EDITransactionSet struct {
	bun.BaseModel `json:"-" bun:"table:edi_transaction_sets,alias:ets"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	Standard       EDIStandard    `json:"standard"       bun:"standard,type:edi_standard_enum,notnull"`
	Code           TransactionSet `json:"code"           bun:"code,type:VARCHAR(20),notnull"`
	Name           string         `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
	Description    string         `json:"description"    bun:"description,type:TEXT,nullzero"`
	DefaultVersion string         `json:"defaultVersion" bun:"default_version,type:VARCHAR(20),notnull"`
	Status         DocumentStatus `json:"status"         bun:"status,type:edi_document_status_enum,notnull"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	TransactionSetID pulid.ID          `json:"transactionSetId" bun:"transaction_set_id,type:VARCHAR(100),notnull"`
	Direction        DocumentDirection `json:"direction"        bun:"direction,type:edi_document_direction_enum,notnull"`
	X12Version       string            `json:"x12Version"       bun:"x12_version,type:VARCHAR(20),notnull"`
	LoopID           string            `json:"loopId"           bun:"loop_id,type:VARCHAR(50),notnull"`
	Name             string            `json:"name"             bun:"name,type:VARCHAR(200),notnull"`
	ParentLoopID     string            `json:"parentLoopId"     bun:"parent_loop_id,type:VARCHAR(50),nullzero"`
	Sequence         int64             `json:"sequence"         bun:"sequence,type:BIGINT,notnull"`
	RepeatPath       string            `json:"repeatPath"       bun:"repeat_path,type:TEXT,nullzero"`
	UsageNotes       string            `json:"usageNotes"       bun:"usage_notes,type:TEXT,nullzero"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	TransactionSetID pulid.ID          `json:"transactionSetId" bun:"transaction_set_id,type:VARCHAR(100),notnull"`
	Direction        DocumentDirection `json:"direction"        bun:"direction,type:edi_document_direction_enum,notnull"`
	X12Version       string            `json:"x12Version"       bun:"x12_version,type:VARCHAR(20),notnull"`
	SegmentID        string            `json:"segmentId"        bun:"segment_id,type:VARCHAR(10),notnull"`
	Name             string            `json:"name"             bun:"name,type:VARCHAR(200),notnull"`
	LoopID           string            `json:"loopId"           bun:"loop_id,type:VARCHAR(50),nullzero"`
	Sequence         int64             `json:"sequence"         bun:"sequence,type:BIGINT,notnull"`
	Required         bool              `json:"required"         bun:"required,type:BOOLEAN,notnull,default:false"`
	MaxUse           int64             `json:"maxUse"           bun:"max_use,type:BIGINT,notnull,default:1"`
	RepeatPath       string            `json:"repeatPath"       bun:"repeat_path,type:TEXT,nullzero"`
	UsageNotes       string            `json:"usageNotes"       bun:"usage_notes,type:TEXT,nullzero"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	TransactionSetID pulid.ID          `json:"transactionSetId" bun:"transaction_set_id,type:VARCHAR(100),notnull"`
	Direction        DocumentDirection `json:"direction"        bun:"direction,type:edi_document_direction_enum,notnull"`
	X12Version       string            `json:"x12Version"       bun:"x12_version,type:VARCHAR(20),notnull"`
	SegmentID        string            `json:"segmentId"        bun:"segment_id,type:VARCHAR(10),notnull"`
	Position         int               `json:"position"         bun:"position,type:INTEGER,notnull"`
	ElementID        string            `json:"elementId"        bun:"element_id,type:VARCHAR(20),nullzero"`
	Name             string            `json:"name"             bun:"name,type:VARCHAR(200),notnull"`
	Required         bool              `json:"required"         bun:"required,type:BOOLEAN,notnull,default:false"`
	MinLength        int               `json:"minLength"        bun:"min_length,type:INTEGER,notnull,default:0"`
	MaxLength        int               `json:"maxLength"        bun:"max_length,type:INTEGER,notnull,default:0"`
	CodeListID       pulid.ID          `json:"codeListId"       bun:"code_list_id,type:VARCHAR(100),nullzero"`
	UsageNotes       string            `json:"usageNotes"       bun:"usage_notes,type:TEXT,nullzero"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	TransactionSetID pulid.ID          `json:"transactionSetId" bun:"transaction_set_id,type:VARCHAR(100),notnull"`
	Direction        DocumentDirection `json:"direction"        bun:"direction,type:edi_document_direction_enum,notnull"`
	X12Version       string            `json:"x12Version"       bun:"x12_version,type:VARCHAR(20),notnull"`
	ElementID        string            `json:"elementId"        bun:"element_id,type:VARCHAR(20),notnull"`
	Code             string            `json:"code"             bun:"code,type:VARCHAR(40),notnull"`
	Description      string            `json:"description"      bun:"description,type:TEXT,notnull"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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
