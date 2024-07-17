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
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Location struct {
	bun.BaseModel `bun:"table:locations,alias:lc" json:"-"`

	ID           uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status       property.Status `bun:"status,type:status_enum" json:"status"`
	Code         string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Name         string          `bun:"type:VARCHAR(255),notnull" json:"name"`
	AddressLine1 string          `bun:"address_line_1,type:VARCHAR(150),notnull" json:"addressLine1"`
	AddressLine2 string          `bun:"address_line_2,type:VARCHAR(150),notnull" json:"addressLine2"`
	City         string          `bun:"type:VARCHAR(150),notnull" json:"city"`
	PostalCode   string          `bun:"type:VARCHAR(10),notnull" json:"postalCode"`
	Longitude    float64         `bun:"type:float" json:"longitude"`
	Latitude     float64         `bun:"type:float" json:"latitude"`
	PlaceID      string          `bun:"type:VARCHAR(255)" json:"placeId"`
	IsGeocoded   bool            `bun:"type:boolean" json:"isGeocoded"`
	Description  string          `bun:"type:TEXT" json:"description"`
	Version      int64           `bun:"type:BIGINT" json:"version"`
	CreatedAt    time.Time       `bun:",notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt    time.Time       `bun:",notnull,default:current_timestamp" json:"updatedAt"`

	LocationCategoryID uuid.UUID `bun:"type:uuid,notnull" json:"locationCategoryId"`
	StateID            uuid.UUID `bun:"type:uuid" json:"stateId"`
	BusinessUnitID     uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID     uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	LocationCategory *LocationCategory  `bun:"rel:belongs-to,join:location_category_id=id" json:"locationCategory"`
	State            *UsState           `bun:"rel:belongs-to,join:state_id=id" json:"state"`
	BusinessUnit     *BusinessUnit      `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization     *Organization      `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	Comments         []*LocationComment `bun:"rel:has-many,join:id=location_id" json:"comments"`
	Contacts         []*LocationContact `bun:"rel:has-many,join:id=location_id" json:"contacts"`
}

func (l Location) Validate() error {
	return validation.ValidateStruct(
		&l,
		validation.Field(&l.BusinessUnitID, validation.Required),
		validation.Field(&l.OrganizationID, validation.Required),
		validation.Field(&l.LocationCategoryID, validation.Required),
		validation.Field(&l.Name, validation.Required, validation.Length(0, 255)),
		validation.Field(&l.AddressLine1, validation.Required, validation.Length(0, 150)),
		validation.Field(&l.AddressLine2, validation.Length(0, 150)),
		validation.Field(&l.City, validation.Required, validation.Length(0, 150)),
		validation.Field(&l.PostalCode, validation.Required, validation.Length(0, 10)),
		validation.Field(&l.StateID, validation.Required, is.UUIDv4.Error("State ID must be a valid UUID.")),
		validation.Field(&l.Longitude, is.Longitude.Error("Longitude must be between -180 and 180.")),
		validation.Field(&l.Latitude, is.Latitude.Error("Latitude must be between -90 and 90.")),
		validation.Field(&l.Contacts),
		validation.Field(&l.Comments),
	)
}

func (l Location) TableName() string {
	return "locations"
}

func (l Location) GetCodePrefix(pattern string) string {
	switch pattern {
	case "NAME-COUNTER":
		return utils.TruncateString(strings.ToUpper(l.Name), 4)
	case "CITY-COUNTER":
		return utils.TruncateString(strings.ToUpper(l.City), 4)
	default:
		return utils.TruncateString(strings.ToUpper(l.Name), 4)
	}
}

func (l Location) GenerateCode(pattern string, counter int) string {
	switch pattern {
	case "NAME-COUNTER":
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(l.Name), 4), counter)
	case "CITY-COUNTER":
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(l.City), 4), counter)
	default:
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(l.Name), 4), counter)
	}
}

func (l *Location) BeforeUpdate(_ context.Context) error {
	l.Version++

	return nil
}

func (l *Location) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := l.Version

	if err := l.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(l).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return &validator.BusinessLogicError{
			Message: fmt.Sprintf("Version mismatch. The Location (ID: %s) has been updated by another user. Please refresh and try again.", l.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*Location)(nil)

func (l *Location) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		l.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		l.UpdatedAt = time.Now()
	}
	return nil
}

// InsertLocation creates a new location record
func (l *Location) InsertLocation(ctx context.Context, tx bun.IDB, codeGen *gen.CodeGenerator, pattern string) error {
	code, err := codeGen.GenerateUniqueCode(ctx, l, pattern, l.OrganizationID)
	if err != nil {
		return err
	}
	l.Code = code

	_, err = tx.NewInsert().Model(l).Exec(ctx)
	if err != nil {
		return err
	}

	if err = l.syncLocationContacts(ctx, tx); err != nil {
		return err
	}

	return l.syncLocationComments(ctx, tx)
}

// UpdateLocation updates an existing location record
func (l *Location) UpdateLocation(ctx context.Context, tx bun.IDB) error {
	if err := l.OptimisticUpdate(ctx, tx); err != nil {
		return err
	}

	if err := l.syncLocationContacts(ctx, tx); err != nil {
		return err
	}

	return l.syncLocationComments(ctx, tx)
}

// syncLocationComments synchronizes the comments associated with a location
func (l *Location) syncLocationComments(ctx context.Context, tx bun.IDB) error {
	return tx.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := l.deleteRemovedComments(ctx, tx); err != nil {
			return err
		}
		return l.upsertComments(ctx, tx)
	})
}

func (l *Location) deleteRemovedComments(ctx context.Context, tx bun.Tx) error {
	if len(l.Comments) == 0 {
		// If there are no comments provided, delete all existing comments
		_, err := tx.NewDelete().
			Model((*LocationComment)(nil)).
			Where("location_id = ?", l.ID).
			Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	}

	// Create a slice of IDs for comments that should be kept
	keepIDs := make([]uuid.UUID, 0, len(l.Comments))
	for _, comment := range l.Comments {
		if comment.ID != uuid.Nil {
			keepIDs = append(keepIDs, comment.ID)
		}
	}

	// Delete comments that are not in the keepIDs slice
	_, err := tx.NewDelete().
		Model((*LocationComment)(nil)).
		Where("location_id = ? AND id NOT IN (?)", l.ID, bun.In(keepIDs)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (l *Location) upsertComments(ctx context.Context, tx bun.Tx) error {
	for _, comment := range l.Comments {
		comment.LocationID = l.ID
		comment.OrganizationID = l.OrganizationID
		comment.BusinessUnitID = l.BusinessUnitID
		comment.UpdatedAt = time.Now()

		_, err := tx.NewInsert().
			Model(comment).
			On("CONFLICT (id) DO UPDATE").
			Set("comment = EXCLUDED.comment").
			Set("comment_type_id = EXCLUDED.comment_type_id").
			Set("user_id = EXCLUDED.user_id").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// syncLocationContacts synchronizes the contacts associated with a location
func (l *Location) syncLocationContacts(ctx context.Context, tx bun.IDB) error {
	return tx.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := l.deleteRemovedContacts(ctx, tx); err != nil {
			return err
		}
		return l.upsertContacts(ctx, tx)
	})
}

func (l *Location) deleteRemovedContacts(ctx context.Context, tx bun.Tx) error {
	if len(l.Contacts) == 0 {
		// If there are no contacts provided, delete all existing contacts
		_, err := tx.NewDelete().
			Model((*LocationContact)(nil)).
			Where("location_id = ?", l.ID).
			Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	}

	// Create a slice of IDs for contacts that should be kept
	keepIDs := make([]uuid.UUID, 0, len(l.Contacts))
	for _, contact := range l.Contacts {
		if contact.ID != uuid.Nil {
			keepIDs = append(keepIDs, contact.ID)
		}
	}

	// Delete contacts that are not in the keepIDs slice
	_, err := tx.NewDelete().
		Model((*LocationContact)(nil)).
		Where("location_id = ? AND id NOT IN (?)", l.ID, bun.In(keepIDs)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (l *Location) upsertContacts(ctx context.Context, tx bun.Tx) error {
	for _, contact := range l.Contacts {
		contact.LocationID = l.ID
		contact.OrganizationID = l.OrganizationID
		contact.BusinessUnitID = l.BusinessUnitID
		contact.UpdatedAt = time.Now()

		_, err := tx.NewInsert().
			Model(contact).
			On("CONFLICT (id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("email_address = EXCLUDED.email_address").
			Set("phone_number = EXCLUDED.phone_number").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
