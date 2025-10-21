package notification

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*NotificationPreference)(nil)
	_ domain.Validatable        = (*NotificationPreference)(nil)
)

// NotificationPreference represents a user's notification preferences for owned records
type NotificationPreference struct {
	bun.BaseModel `bun:"table:notification_preferences,alias:np" json:"-"`

	ID                     pulid.ID            `json:"id"                        bun:"id,pk,type:VARCHAR(100)"`
	UserID                 pulid.ID            `json:"userId"                    bun:"user_id,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID            `json:"organizationId"            bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID            `json:"businessUnitId"            bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Resource               permission.Resource `json:"resource"                  bun:"resource,type:VARCHAR(50),notnull"`
	UpdateTypes            []UpdateType        `json:"updateTypes"               bun:"update_types,type:TEXT[],notnull"`
	ExcludedUserIDs        []pulid.ID          `json:"excludedUserIds,omitempty" bun:"excluded_user_ids,type:VARCHAR(100)[]"`
	IncludedRoleIDs        []pulid.ID          `json:"includedRoleIds,omitempty" bun:"included_role_ids,type:VARCHAR(100)[]"`
	PreferredChannels      []Channel           `json:"preferredChannels"         bun:"preferred_channels,type:VARCHAR(20)[],notnull"`
	QuietHoursEnabled      bool                `json:"quietHoursEnabled"         bun:"quiet_hours_enabled,type:BOOLEAN,notnull,default:false"`
	NotifyOnAllUpdates     bool                `json:"notifyOnAllUpdates"        bun:"notify_on_all_updates,type:BOOLEAN,notnull,default:false"`
	NotifyOnlyOwnedRecords bool                `json:"notifyOnlyOwnedRecords"    bun:"notify_only_owned_records,type:BOOLEAN,notnull,default:true"`
	BatchNotifications     bool                `json:"batchNotifications"        bun:"batch_notifications,type:BOOLEAN,notnull,default:false"`
	IsActive               bool                `json:"isActive"                  bun:"is_active,type:BOOLEAN,notnull,default:true"`
	QuietHoursStart        string              `json:"quietHoursStart,omitempty" bun:"quiet_hours_start,type:TIME"`
	QuietHoursEnd          string              `json:"quietHoursEnd,omitempty"   bun:"quiet_hours_end,type:TIME"`
	Timezone               string              `json:"timezone"                  bun:"timezone,type:VARCHAR(50),notnull,default:'UTC'"`
	BatchIntervalMinutes   int                 `json:"batchIntervalMinutes"      bun:"batch_interval_minutes,type:INT,notnull,default:15"`
	Version                int64               `json:"version"                   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt              int64               `json:"createdAt"                 bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64               `json:"updatedAt"                 bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (np *NotificationPreference) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := utils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if np.ID.IsNil() {
			np.ID = pulid.MustNew("npref_")
		}
		np.CreatedAt = now
	case *bun.UpdateQuery:
		np.UpdatedAt = now
		np.Version++
	}

	return nil
}

func (np *NotificationPreference) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(np,
		// UserID is required
		validation.Field(&np.UserID,
			validation.Required.Error("User ID is required"),
		),

		// OrganizationID is required
		validation.Field(&np.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),

		// BusinessUnitID is required
		validation.Field(&np.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),

		// Resource is required and must be valid
		validation.Field(&np.Resource,
			validation.Required.Error("Resource is required"),
			// Validate against resources that support notifications
			validation.In(
				permission.ResourceShipment,
				permission.ResourceWorker,
				permission.ResourceCustomer,
				permission.ResourceTractor,
				permission.ResourceTrailer,
				permission.ResourceLocation,
				permission.ResourceCommodity,
				permission.ResourceFleetCode,
				permission.ResourceEquipmentType,
				permission.ResourceEquipmentManufacturer,
			).Error("Resource must be a valid resource type that supports notifications"),
		),

		// UpdateTypes must have at least one type if not notifying on all updates
		validation.Field(&np.UpdateTypes,
			validation.When(
				!np.NotifyOnAllUpdates,
				validation.Required.Error(
					"At least one update type is required when not notifying on all updates",
				),
			),
		),

		// PreferredChannels must have at least one channel
		validation.Field(&np.PreferredChannels,
			validation.Required.Error("At least one preferred channel is required"),
			validation.Each(validation.In(
				ChannelUser,
			).Error("Invalid channel type")),
		),

		// Quiet hours validation
		validation.Field(&np.QuietHoursStart,
			validation.When(
				np.QuietHoursEnabled,
				validation.Required.Error(
					"Quiet hours start time is required when quiet hours are enabled",
				),
			),
		),
		validation.Field(&np.QuietHoursEnd,
			validation.When(
				np.QuietHoursEnabled,
				validation.Required.Error(
					"Quiet hours end time is required when quiet hours are enabled",
				),
			),
		),

		// Batch interval must be positive
		validation.Field(&np.BatchIntervalMinutes,
			validation.By(func(value any) error {
				v, ok := value.(int)
				if !ok {
					return validation.NewError(
						"validation_type",
						"Batch interval must be an integer",
					)
				}
				if v < 1 {
					return validation.NewError(
						"validation_min",
						"Batch interval must be at least 1 minute",
					)
				}
				if v > 1440 {
					return validation.NewError(
						"validation_max",
						"Batch interval cannot exceed 24 hours",
					)
				}
				return nil
			}),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (np *NotificationPreference) GetID() pulid.ID {
	return np.ID
}

func (np *NotificationPreference) GetTableName() string {
	return "notification_preferences"
}

func (np *NotificationPreference) IsUpdateTypeEnabled(updateType UpdateType) bool {
	if np.NotifyOnAllUpdates {
		return true
	}

	for _, ut := range np.UpdateTypes {
		if ut == updateType || ut == UpdateTypeAny {
			return true
		}
	}

	return false
}

func (np *NotificationPreference) ShouldNotifyUser(updatedByUserID pulid.ID) bool {
	// Check if the updater is in the excluded list
	for _, excludedID := range np.ExcludedUserIDs {
		if excludedID == updatedByUserID {
			return false
		}
	}

	return true
}
