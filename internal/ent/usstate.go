// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/ent/usstate"
	"github.com/google/uuid"
)

// UsState is the model entity for the UsState schema.
type UsState struct {
	config `json:"-" validate:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// The time that this entity was created.
	CreatedAt time.Time `json:"createdAt" validate:"omitempty"`
	// The last time that this entity was updated.
	UpdatedAt time.Time `json:"updatedAt" validate:"omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name"`
	// Abbreviation holds the value of the "abbreviation" field.
	Abbreviation string `json:"abbreviation"`
	// CountryName holds the value of the "country_name" field.
	CountryName string `json:"countryName"`
	// CountryIso3 holds the value of the "country_iso3" field.
	CountryIso3  string `json:"countryIso3"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*UsState) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case usstate.FieldName, usstate.FieldAbbreviation, usstate.FieldCountryName, usstate.FieldCountryIso3:
			values[i] = new(sql.NullString)
		case usstate.FieldCreatedAt, usstate.FieldUpdatedAt:
			values[i] = new(sql.NullTime)
		case usstate.FieldID:
			values[i] = new(uuid.UUID)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the UsState fields.
func (us *UsState) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case usstate.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				us.ID = *value
			}
		case usstate.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				us.CreatedAt = value.Time
			}
		case usstate.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				us.UpdatedAt = value.Time
			}
		case usstate.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				us.Name = value.String
			}
		case usstate.FieldAbbreviation:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field abbreviation", values[i])
			} else if value.Valid {
				us.Abbreviation = value.String
			}
		case usstate.FieldCountryName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field country_name", values[i])
			} else if value.Valid {
				us.CountryName = value.String
			}
		case usstate.FieldCountryIso3:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field country_iso3", values[i])
			} else if value.Valid {
				us.CountryIso3 = value.String
			}
		default:
			us.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the UsState.
// This includes values selected through modifiers, order, etc.
func (us *UsState) Value(name string) (ent.Value, error) {
	return us.selectValues.Get(name)
}

// Update returns a builder for updating this UsState.
// Note that you need to call UsState.Unwrap() before calling this method if this UsState
// was returned from a transaction, and the transaction was committed or rolled back.
func (us *UsState) Update() *UsStateUpdateOne {
	return NewUsStateClient(us.config).UpdateOne(us)
}

// Unwrap unwraps the UsState entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (us *UsState) Unwrap() *UsState {
	_tx, ok := us.config.driver.(*txDriver)
	if !ok {
		panic("ent: UsState is not a transactional entity")
	}
	us.config.driver = _tx.drv
	return us
}

// String implements the fmt.Stringer.
func (us *UsState) String() string {
	var builder strings.Builder
	builder.WriteString("UsState(")
	builder.WriteString(fmt.Sprintf("id=%v, ", us.ID))
	builder.WriteString("created_at=")
	builder.WriteString(us.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(us.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(us.Name)
	builder.WriteString(", ")
	builder.WriteString("abbreviation=")
	builder.WriteString(us.Abbreviation)
	builder.WriteString(", ")
	builder.WriteString("country_name=")
	builder.WriteString(us.CountryName)
	builder.WriteString(", ")
	builder.WriteString("country_iso3=")
	builder.WriteString(us.CountryIso3)
	builder.WriteByte(')')
	return builder.String()
}

// UsStates is a parsable slice of UsState.
type UsStates []*UsState
