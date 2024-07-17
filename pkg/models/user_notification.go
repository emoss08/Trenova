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

type UserNotification struct {
	bun.BaseModel  `bun:"table:user_notifications,alias:un" json:"-"`
	CreatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	IsRead         bool      `bun:"is_read,default:false" json:"isRead"`
	Title          string    `bun:",notnull" json:"title"`
	Description    string    `bun:",notnull" json:"description"`
	ActionURL      string    `json:"actionUrl"`
	UserID         uuid.UUID `bun:"type:uuid,notnull" json:"userId"`
	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	User         *User         `bun:"rel:belongs-to,join:user_id=id" json:"user"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

var _ bun.BeforeAppendModelHook = (*UserNotification)(nil)

func (c *UserNotification) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
