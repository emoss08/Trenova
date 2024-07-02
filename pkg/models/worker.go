package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type WorkerPermission string

const (
	// PermissionWorkerView is the permission to view worker details
	PermissionWorkerView = WorkerPermission("worker.view")

	// PermissionWorkerEdit is the permission to edit worker details
	PermissionWorkerEdit = WorkerPermission("worker.edit")

	// PermissionWorkerAdd is the permission to add a necw worker
	PermissionWorkerAdd = WorkerPermission("worker.add")

	// PermissionWorkerDelete is the permission to delete an worker
	PermissionWorkerDelete = WorkerPermission("worker.delete")
)

// String returns the string representation of the WorkerPermission
func (p WorkerPermission) String() string {
	return string(p)
}

type Worker struct {
	bun.BaseModel  `bun:"table:workers,alias:wk" json:"-"`
	CreatedAt      time.Time           `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time           `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID           `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status     `bun:"status,type:status" json:"status"`
	Code           string              `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	WorkerType     property.WorkerType `bun:"type:worker_type_enum,default:'Employee',notnull" json:"workerType"`
	FirstName      string              `bun:"type:VARCHAR(255),notnull" json:"firstName"`
	LastName       string              `bun:"type:VARCHAR(255),notnull" json:"lastName"`
	AddressLine1   string              `bun:"address_line_1,type:VARCHAR(150),notnull" json:"addressLine1"`
	AddressLine2   string              `bun:"address_line_2,type:VARCHAR(150)" json:"addressLine2"`
	City           string              `bun:"type:VARCHAR(150),notnull" json:"city"`
	PostalCode     string              `bun:"type:VARCHAR(10),notnull" json:"postalCode"`
	StateID        *uuid.UUID          `bun:"type:uuid" json:"stateId"`
	FleetCodeID    *uuid.UUID          `bun:"type:uuid" json:"fleetCodeId"`
	ManagerID      *uuid.UUID          `bun:"type:uuid" json:"managerId"`
	BusinessUnitID uuid.UUID           `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID           `bun:"type:uuid,notnull" json:"organizationId"`

	WorkerProfile *WorkerProfile `bun:"rel:has-one,join:id=worker_id" json:"workerProfile"`
	State         *UsState       `bun:"rel:belongs-to,join:state_id=id" json:"-"`
	FleetCode     *FleetCode     `bun:"rel:belongs-to,join:fleet_code_id=id" json:"-"`
	Manager       *User          `bun:"rel:belongs-to,join:manager_id=id" json:"-"`
	BusinessUnit  *BusinessUnit  `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization  *Organization  `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (w Worker) Validate() error {
	return validation.ValidateStruct(
		&w,
		validation.Field(&w.Code, validation.Required, validation.Length(10, 10).Error("Code must be 4 characters")),
		validation.Field(&w.BusinessUnitID, validation.Required),
		validation.Field(&w.OrganizationID, validation.Required),
	)
}

func (w *Worker) InsertWorker(ctx context.Context, tx bun.Tx, codeGen *gen.CodeGenerator, pattern string) error {
	if w.WorkerProfile == nil {
		return validator.DBValidationError{
			Field:   "workerProfile",
			Message: "worker profile is required",
		}
	}

	code, err := codeGen.GenerateUniqueCode(ctx, w, pattern, w.OrganizationID)
	if err != nil {
		return fmt.Errorf("error generating unique code: %w", err)
	}
	w.Code = code

	if err = w.Validate(); err != nil {
		return fmt.Errorf("worker validation failed: %w", err)
	}

	if err = w.WorkerProfile.Validate(); err != nil {
		return fmt.Errorf("worker profile validation failed: %w", err)
	}

	_, err = tx.NewInsert().Model(w).Exec(ctx)
	if err != nil {
		return fmt.Errorf("error inserting worker: %w", err)
	}

	w.WorkerProfile.WorkerID = w.ID
	w.WorkerProfile.BusinessUnitID = w.BusinessUnitID
	w.WorkerProfile.OrganizationID = w.OrganizationID

	_, err = tx.NewInsert().Model(w.WorkerProfile).Exec(ctx)
	if err != nil {
		return fmt.Errorf("error inserting worker profile: %w", err)
	}

	return nil
}

func (w *Worker) UpdateWorker(ctx context.Context, tx bun.Tx) error {
	if w.WorkerProfile == nil {
		return validator.DBValidationError{
			Field:   "workerProfile",
			Message: "worker profile is required",
		}
	}

	if err := w.Validate(); err != nil {
		return fmt.Errorf("worker validation failed: %w", err)
	}

	if err := w.WorkerProfile.Validate(); err != nil {
		return fmt.Errorf("worker profile validation failed: %w", err)
	}

	_, err := tx.NewUpdate().Model(w).Where("id = ?", w.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("error updating worker: %w", err)
	}

	_, err = tx.NewUpdate().Model(w.WorkerProfile).Where("worker_id = ?", w.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("error updating worker profile: %w", err)
	}

	return nil
}

func (w *Worker) TableName() string {
	return "workers"
}

func (w *Worker) GetCodePrefix(pattern string) string {
	switch pattern {
	case "TYPE-LASTNAME-COUNTER":
		return fmt.Sprintf("%c%s", w.WorkerType[0], utils.TruncateString(strings.ToUpper(w.LastName), 3))
	case "INITIAL-LASTNAME-COUNTER":
		return fmt.Sprintf("%c%s", w.FirstName[0], utils.TruncateString(strings.ToUpper(w.LastName), 3))
	case "LASTNAME-COUNTER":
		return utils.TruncateString(strings.ToUpper(w.LastName), 4)
	default:
		return utils.TruncateString(strings.ToUpper(w.LastName), 4)
	}
}

func (w *Worker) GenerateCode(pattern string, counter int) string {
	switch pattern {
	case "TYPE-LASTNAME-COUNTER":
		return fmt.Sprintf("%c%s%04d", w.WorkerType[0], utils.TruncateString(strings.ToUpper(w.LastName), 3), counter)
	case "INITIAL-LASTNAME-COUNTER":
		return fmt.Sprintf("%c%s%04d", w.FirstName[0], utils.TruncateString(strings.ToUpper(w.LastName), 3), counter)
	case "LASTNAME-COUNTER":
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(w.LastName), 4), counter)
	default:
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(w.LastName), 4), counter)
	}
}

var _ bun.BeforeAppendModelHook = (*Worker)(nil)

func (w *Worker) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		w.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		w.UpdatedAt = time.Now()
	}
	return nil
}
