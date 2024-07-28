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

package audit

import (
	"encoding/json"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Log struct {
	bun.BaseModel `bun:"table:audit_logs,alias:al"`

	ID           uuid.UUID               `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	TableName    string                  `bun:"type:varchar(255),notnull" json:"tableName"`
	EntityID     string                  `bun:"type:varchar(255),notnull" json:"entityId"`
	Action       property.AuditLogAction `bun:"type:audit_log_status_enum,notnull" json:"action"`
	Changes      json.RawMessage         `bun:"type:jsonb" json:"changes"`
	Timestamp    time.Time               `bun:"default:current_timestamp" json:"timestamp"`
	Description  string                  `bun:"type:text" json:"description"`
	ErrorMessage string                  `bun:"type:text" json:"errorMessage"`
	Status       property.LogStatus      `bun:"type:log_status_enum,notnull,default:'ATTEMPTED'" json:"status"`
	Username     string                  `bun:"type:varchar(255),notnull" json:"username"`

	UserID         uuid.UUID `bun:"type:uuid" json:"userId"`
	OrganizationID uuid.UUID `bun:"type:uuid" json:"orgId"`
	BusinessUnitID uuid.UUID `bun:"type:uuid" json:"businessUnitId"`
}
