/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package edi

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/edi/pkg/configtypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

type DelimiterConfig struct {
	Element    string `json:"element"`
	Component  string `json:"component"`
	Segment    string `json:"segment"`
	Repetition string `json:"repetition"`
}

type ValidationConfig struct {
	Strictness               configtypes.Strictness `json:"strictness"`
	EnforceSECount           *bool                  `json:"enforceSeCount,omitempty"`
	RequirePickupAndDelivery *bool                  `json:"requirePickupAndDelivery,omitempty"`
	RequireB2ShipID          *bool                  `json:"requireB2ShipID,omitempty"`
	RequireN1SH              *bool                  `json:"requireN1SH,omitempty"` // SH
	RequireN1ST              *bool                  `json:"requireN1ST,omitempty"` // ST
}

type PartnerConfig struct {
	bun.BaseModel `bun:"table:edi_profiles,alias:ep" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:varchar(100),pk,notnull"               json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:varchar(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:varchar(100),pk,notnull"  json:"organizationId"`

	// Core profile
	Name       string           `bun:"name,type:varchar(255),notnull"        json:"name"`
	SchemaPath string           `bun:"schema_path,type:varchar(255),notnull" json:"schema"`
	Delims     DelimiterConfig  `bun:"delims,type:jsonb,notnull"             json:"delimiters"`
	Validation ValidationConfig `bun:"validation,type:jsonb,notnull"         json:"validation"`

	// Mapping options
	References          map[string][]string `bun:"references,type:jsonb,notnull"                   json:"references,omitempty"`
	PartyRoles          map[string][]string `bun:"party_roles,type:jsonb,notnull"                  json:"party_roles,omitempty"`
	StopTypeMap         map[string]string   `bun:"stop_type_map,type:jsonb,notnull"                json:"stop_type_map,omitempty"`
	ShipmentIDQuals     []string            `bun:"shipment_id_quals,type:jsonb,notnull"            json:"shipment_id_quals,omitempty"`
	ShipmentIDMode      string              `bun:"shipment_id_mode,type:varchar(255),notnull"      json:"shipment_id_mode,omitempty"`
	CarrierSCACFallback string              `bun:"carrier_scac_fallback,type:varchar(255),notnull" json:"carrier_scac_fallback,omitempty"`
	IncludeRawL11       bool                `bun:"include_raw_l11,type:boolean,notnull"            json:"include_raw_l11,omitempty"`
	RawL11Filter        []string            `bun:"raw_l11_filter,type:jsonb,notnull"               json:"raw_l11_filter,omitempty"`
	EquipmentTypeMap    map[string]string   `bun:"equipment_type_map,type:jsonb,notnull"           json:"equipment_type_map,omitempty"`
	IncludeSegments     bool                `bun:"include_segments,type:boolean,notnull"           json:"include_segments,omitempty"`
	EmitISODateTime     bool                `bun:"emit_iso_datetime,type:boolean,notnull"          json:"emit_iso_datetime,omitempty"`
	Timezone            string              `bun:"timezone,type:varchar(255),notnull"              json:"timezone,omitempty"`
	ServiceLevelQuals   []string            `bun:"service_level_quals,type:jsonb,notnull"          json:"service_level_quals,omitempty"`
	ServiceLevelMap     map[string]string   `bun:"service_level_map,type:jsonb,notnull"            json:"service_level_map,omitempty"`
	AccessorialQuals    []string            `bun:"accessorial_quals,type:jsonb,notnull"            json:"accessorial_quals,omitempty"`
	AccessorialMap      map[string]string   `bun:"accessorial_map,type:jsonb,notnull"              json:"accessorial_map,omitempty"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT"                                                                  json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

// Validate implements domain.Validatable for PartnerConfig.
func (pc *PartnerConfig) Validate(_ context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStruct(
		pc,
		validation.Field(&pc.Name,
			validation.Required.Error("Name is required"),
			validation.RuneLength(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&pc.SchemaPath,
			validation.Required.Error("Schema path is required"),
			validation.RuneLength(1, 255).Error("Schema path must be between 1 and 255 characters"),
		),
		// Delimiters: if provided, must be exactly 1 char
		validation.Field(
			&pc.Delims.Element,
			validation.RuneLength(0, 1).Error("Element delimiter must be a single character"),
		),
		validation.Field(
			&pc.Delims.Component,
			validation.RuneLength(0, 1).Error("Component delimiter must be a single character"),
		),
		validation.Field(
			&pc.Delims.Segment,
			validation.RuneLength(0, 1).Error("Segment delimiter must be a single character"),
		),
		validation.Field(
			&pc.Delims.Repetition,
			validation.RuneLength(0, 1).Error("Repetition delimiter must be a single character"),
		),
		// Strictness enum
		validation.Field(
			&pc.Validation.Strictness,
			validation.In(configtypes.Strict, configtypes.Lenient).
				Error("Strictness must be 'strict' or 'lenient'"),
		),
		// Shipment ID mode
		validation.Field(&pc.ShipmentIDMode,
			validation.When(
				pc.ShipmentIDMode != "",
				validation.In("ref_first", "b2_first", "ref_only", "b2_only").
					Error("Invalid shipment_id_mode"),
			),
		),
		// Timezone
		validation.Field(&pc.Timezone, validation.By(domain.ValidateTimezone)),
	)
	if err != nil {
		var ve validation.Errors
		if eris.As(err, &ve) {
			errors.FromOzzoErrors(ve, multiErr)
		}
	}
}

func (pc *PartnerConfig) GetTableName() string { return "edi_profiles" }
func (pc *PartnerConfig) GetID() string        { return pc.ID.String() }

func (pc *PartnerConfig) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if pc.ID.IsNil() {
			pc.ID = pulid.MustNew("edp_")
		}
		pc.CreatedAt = now
	case *bun.UpdateQuery:
		pc.UpdatedAt = now
	}
	return nil
}

// ToConfigTypes converts the domain PartnerConfig into the portable
// shared EDI config shape used across services (configtypes.PartnerConfig).
func (pc *PartnerConfig) ToConfigTypes() *configtypes.PartnerConfig {
	out := &configtypes.PartnerConfig{
		Name:       pc.Name,
		SchemaPath: pc.SchemaPath,
		Delims: configtypes.DelimiterConfig{
			Element:    pc.Delims.Element,
			Component:  pc.Delims.Component,
			Segment:    pc.Delims.Segment,
			Repetition: pc.Delims.Repetition,
		},
		Validation: configtypes.ValidationConfig{
			Strictness:               pc.Validation.Strictness,
			EnforceSECount:           pc.Validation.EnforceSECount,
			RequirePickupAndDelivery: pc.Validation.RequirePickupAndDelivery,
			RequireB2ShipID:          pc.Validation.RequireB2ShipID,
			RequireN1SH:              pc.Validation.RequireN1SH,
			RequireN1ST:              pc.Validation.RequireN1ST,
		},
		References:          pc.References,
		PartyRoles:          pc.PartyRoles,
		StopTypeMap:         pc.StopTypeMap,
		ShipmentIDQuals:     pc.ShipmentIDQuals,
		ShipmentIDMode:      pc.ShipmentIDMode,
		CarrierSCACFallback: pc.CarrierSCACFallback,
		IncludeRawL11:       pc.IncludeRawL11,
		RawL11Filter:        pc.RawL11Filter,
		EquipmentTypeMap:    pc.EquipmentTypeMap,
		IncludeSegments:     pc.IncludeSegments,
		EmitISODateTime:     pc.EmitISODateTime,
		Timezone:            pc.Timezone,
		ServiceLevelQuals:   pc.ServiceLevelQuals,
		ServiceLevelMap:     pc.ServiceLevelMap,
		AccessorialQuals:    pc.AccessorialQuals,
		AccessorialMap:      pc.AccessorialMap,
	}
	return out
}
