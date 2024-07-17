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

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type CustomerMasterKeyGeneration struct {
	bun.BaseModel `bun:"table:worker_master_key_generations,alias:wmkg" json:"-"`
	CreatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID            uuid.UUID  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Pattern       string     `bun:"type:VARCHAR(255),notnull" json:"pattern"`
	MasterKeyID   *uuid.UUID `bun:"type:uuid" json:"masterKeyGenerationId"`

	MasterKey *MasterKeyGeneration `bun:"rel:belongs-to,join:master_key_id=id" json:"masterKeyGeneration"`
}

func QueryCustomerMasterKeyGenerationByOrgID(ctx context.Context, db *bun.DB, orgID uuid.UUID) (*CustomerMasterKeyGeneration, error) {
	var customerMasterKeyGeneration CustomerMasterKeyGeneration
	err := db.NewSelect().Model(&customerMasterKeyGeneration).Relation("MasterKey").Where("master_key.organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &customerMasterKeyGeneration, nil
}

var _ bun.BeforeAppendModelHook = (*CustomerMasterKeyGeneration)(nil)

func (c *CustomerMasterKeyGeneration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
