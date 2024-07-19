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
	"encoding/json"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AuditLog struct {
	bun.BaseModel `bun:"table:audit_logs,alias:al"`

	ID        uuid.UUID               `bun:",pk,type:uuid,default:uuid_generate_v4()"`
	TableName string                  `bun:"type:varchar(255),notnull"`
	EntityID  string                  `bun:"type:varchar(255),notnull"`
	Action    property.AuditLogAction `bun:"type:audit_log_status_enum,notnull"`
	Data      json.RawMessage         `bun:"type:jsonb"`
	Timestamp time.Time               `bun:"default:current_timestamp"`

	UserID         uuid.UUID `bun:"type:uuid"`
	OrganizationID uuid.UUID `bun:"type:uuid"`
	BusinessUnitID uuid.UUID `bun:"type:uuid"`

	User         *User         `bun:"rel:belongs-to,join:user_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}
