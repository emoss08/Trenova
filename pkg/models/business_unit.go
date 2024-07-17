// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package models

import (
	"context"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type BusinessUnit struct {
	bun.BaseModel `bun:"business_units"`
	CreatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID            uuid.UUID  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name          string     `bun:"type:VARCHAR(100),notnull" json:"name" queryField:"true"`
	AddressLine1  string     `json:"addressLine1"`
	AddressLine2  string     `json:"addressLine2"`
	City          string     `json:"city"`
	StateID       *uuid.UUID `bun:"type:uuid" json:"stateId"`
	State         *UsState   `bun:"rel:belongs-to,join:state_id=id" json:"state"`
	PostalCode    string     `json:"postalCode"`
	PhoneNumber   string     `bun:"type:VARCHAR(15)" json:"phoneNumber"`
	ContactName   string     `json:"contactName"`
	ContactEmail  string     `json:"contactEmail"`
}

func (b BusinessUnit) Validate() error {
	return validation.ValidateStruct(
		&b,
		// Name is cannot be empty
		validation.Field(&b.Name, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*BusinessUnit)(nil)

func (b *BusinessUnit) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		b.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		b.UpdatedAt = time.Now()
	}
	return nil
}
