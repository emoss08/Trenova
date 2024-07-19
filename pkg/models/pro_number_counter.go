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
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	nowFunc = time.Now
	mu      sync.Mutex
)

func SetNow(f func() time.Time) {
	nowFunc = f
}

// ProNumberCounter stores the last used pro_number for each organization
type ProNumberCounter struct {
	bun.BaseModel  `bun:"table:pro_number_counters,alias:pnc"`
	ID             uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull,unique"`
	LastUsedNumber int       `bun:"type:integer,notnull"`
	UpdatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// GenerateProNumber generates the next pro_number for a given organization
func GenerateProNumber(ctx context.Context, db *bun.DB, orgID uuid.UUID) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	currentYear := nowFunc().Year()

	var counter ProNumberCounter
	err := db.NewSelect().
		Model(&counter).
		Where("organization_id = ?", orgID).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		// Counter doesn't exist, create a new one
		counter = ProNumberCounter{
			OrganizationID: orgID,
			LastUsedNumber: 0,
		}
	}

	// Increment the counter
	counter.LastUsedNumber++

	_, err = db.NewInsert().
		Model(&counter).
		On("CONFLICT (organization_id) DO UPDATE").
		Set("last_used_number = EXCLUDED.last_used_number").
		Set("updated_at = CURRENT_TIMESTAMP").
		Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to update pro_number counter: %w", err)
	}

	// Generate pro_number in format SYYYY-NNNNNN (e.g., S2023-000001)
	proNumber := fmt.Sprintf("S%d-%06d", currentYear, counter.LastUsedNumber)

	return proNumber, nil
}
