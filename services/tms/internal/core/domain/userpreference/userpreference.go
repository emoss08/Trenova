package userpreference

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*UserPreference)(nil)
	_ domain.Validatable        = (*UserPreference)(nil)
)

type PreferenceData struct {
	DismissedNotices []string       `json:"dismissedNotices"`
	DismissedDialogs []string       `json:"dismissedDialogs"`
	UISettings       map[string]any `json:"uiSettings"`
}

type UserPreference struct {
	bun.BaseModel `bun:"table:user_preferences,alias:up" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	UserID         pulid.ID       `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull,unique"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Preferences    PreferenceData `json:"preferences"    bun:"preferences,type:JSONB,notnull"`
	Version        int64          `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (up *UserPreference) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := utils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if up.ID.IsNil() {
			up.ID = pulid.MustNew("up_")
		}

		up.CreatedAt = now
	case *bun.UpdateQuery:
		up.UpdatedAt = now
		up.Version++
	}

	return nil
}

func (up *UserPreference) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(up,
		validation.Field(&up.UserID,
			validation.Required.Error("User ID is required"),
		),
		validation.Field(&up.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(&up.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (up *UserPreference) GetID() pulid.ID {
	return up.ID
}

func (up *UserPreference) GetTableName() string {
	return "user_preferences"
}

func (up *UserPreference) IsDismissed(key string, isDialog bool) bool {
	if isDialog {
		return slices.Contains(up.Preferences.DismissedDialogs, key)
	}

	return slices.Contains(up.Preferences.DismissedNotices, key)
}

func (up *UserPreference) AddDismissed(key string, isDialog bool) {
	if isDialog {
		if !up.IsDismissed(key, true) {
			up.Preferences.DismissedDialogs = append(up.Preferences.DismissedDialogs, key)
		}
	} else {
		if !up.IsDismissed(key, false) {
			up.Preferences.DismissedNotices = append(up.Preferences.DismissedNotices, key)
		}
	}
}

func (up *UserPreference) RemoveDismissed(key string, isDialog bool) {
	if isDialog {
		filtered := make([]string, 0)
		for _, d := range up.Preferences.DismissedDialogs {
			if d != key {
				filtered = append(filtered, d)
			}
		}
		up.Preferences.DismissedDialogs = filtered
	} else {
		filtered := make([]string, 0)
		for _, n := range up.Preferences.DismissedNotices {
			if n != key {
				filtered = append(filtered, n)
			}
		}
		up.Preferences.DismissedNotices = filtered
	}
}

func (up *UserPreference) GetUISetting(key string) (any, bool) {
	value, exists := up.Preferences.UISettings[key]
	return value, exists
}

func (up *UserPreference) SetUISetting(key string, value any) {
	if up.Preferences.UISettings == nil {
		up.Preferences.UISettings = make(map[string]any)
	}
	up.Preferences.UISettings[key] = value
}

func (up *UserPreference) RemoveUISetting(key string) {
	delete(up.Preferences.UISettings, key)
}
