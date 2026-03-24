package services

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type APIKeyUsageEvent struct {
	APIKeyID       pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	OccurredAt     time.Time
	IPAddress      string
	UserAgent      string
}

type UsageRecorder interface {
	RecordUsage(event APIKeyUsageEvent)
}

type APIKeyPermissionInput struct {
	Resource   string                 `json:"resource"`
	Operations []permission.Operation `json:"operations"`
	DataScope  permission.DataScope   `json:"dataScope"`
}

type APIKeyResponse struct {
	ID              string                  `json:"id"`
	BusinessUnitID  string                  `json:"businessUnitId"`
	OrganizationID  string                  `json:"organizationId"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description"`
	KeyPrefix       string                  `json:"keyPrefix"`
	Status          string                  `json:"status"`
	ExpiresAt       int64                   `json:"expiresAt"`
	LastUsedAt      int64                   `json:"lastUsedAt"`
	CreatedAt       int64                   `json:"createdAt"`
	UpdatedAt       int64                   `json:"updatedAt"`
	PermissionScope string                  `json:"permissionScope"`
	Permissions     []APIKeyPermissionInput `json:"permissions,omitempty"`
}

type APIKeySecretResponse struct {
	APIKeyResponse
	Token string `json:"token"`
}

type CreateAPIKeyRequest struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	ExpiresAt   int64                   `json:"expiresAt"`
	Permissions []APIKeyPermissionInput `json:"permissions"`
}

func (r *CreateAPIKeyRequest) Validate() error {
	me := errortypes.NewMultiError()
	valErr := validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&r.Permissions,
			validation.Required.Error("At least one permission is required"),
			validation.Length(1, 100),
		),
	)
	me.AddOzzoError(valErr)

	for i := range r.Permissions {
		item := &r.Permissions[i]
		if strings.TrimSpace(item.Resource) == "" {
			me.Add("permissions", errortypes.ErrRequired, "Each permission must include a resource")
		}
		if len(item.Operations) == 0 {
			me.Add(
				"permissions",
				errortypes.ErrRequired,
				"Each permission must include at least one operation",
			)
			break
		}
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

type UpdateAPIKeyRequest struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	ExpiresAt   int64                   `json:"expiresAt"`
	Permissions []APIKeyPermissionInput `json:"permissions"`
}

func (r *UpdateAPIKeyRequest) Validate() error {
	me := errortypes.NewMultiError()
	valErr := validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&r.Permissions,
			validation.Required.Error("At least one permission is required"),
			validation.Length(1, 100),
		),
	)
	me.AddOzzoError(valErr)
	for i := range r.Permissions {
		item := &r.Permissions[i]
		if strings.TrimSpace(item.Resource) == "" {
			me.Add("permissions", errortypes.ErrRequired, "Each permission must include a resource")
		}
		if len(item.Operations) == 0 {
			me.Add(
				"permissions",
				errortypes.ErrRequired,
				"Each permission must include at least one operation",
			)
			break
		}
	}
	if me.HasErrors() {
		return me
	}
	return nil
}
