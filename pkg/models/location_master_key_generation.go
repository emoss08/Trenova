// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type LocationMasterKeyGeneration struct {
	bun.BaseModel `bun:"table:location_master_key_generations,alias:lmkg" json:"-"`
	CreatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID            uuid.UUID  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Pattern       string     `bun:"type:VARCHAR(255),notnull" json:"pattern"`
	MasterKeyID   *uuid.UUID `bun:"type:uuid" json:"masterKeyGenerationId"`

	MasterKey *MasterKeyGeneration `bun:"rel:belongs-to,join:master_key_id=id" json:"masterKeyGeneration"`
}

func QueryLocationMasterKeyGenerationByOrgID(ctx context.Context, db *bun.DB, orgID uuid.UUID) (*LocationMasterKeyGeneration, error) {
	var locationMasterKeyGeneration LocationMasterKeyGeneration
	err := db.NewSelect().Model(&locationMasterKeyGeneration).Relation("MasterKey").Where("master_key.organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &locationMasterKeyGeneration, nil
}

var _ bun.BeforeAppendModelHook = (*LocationMasterKeyGeneration)(nil)

func (m *LocationMasterKeyGeneration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		m.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		m.UpdatedAt = time.Now()
	}
	return nil
}
