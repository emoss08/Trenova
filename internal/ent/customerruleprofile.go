// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/customerruleprofile"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

// CustomerRuleProfile is the model entity for the CustomerRuleProfile schema.
type CustomerRuleProfile struct {
	config `json:"-" validate:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// BusinessUnitID holds the value of the "business_unit_id" field.
	BusinessUnitID uuid.UUID `json:"businessUnitId"`
	// OrganizationID holds the value of the "organization_id" field.
	OrganizationID uuid.UUID `json:"organizationId"`
	// The time that this entity was created.
	CreatedAt time.Time `json:"createdAt" validate:"omitempty"`
	// The last time that this entity was updated.
	UpdatedAt time.Time `json:"updatedAt" validate:"omitempty"`
	// The current version of this entity.
	Version int `json:"version" validate:"omitempty"`
	// CustomerID holds the value of the "customer_id" field.
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	// BillingCycle holds the value of the "billing_cycle" field.
	BillingCycle customerruleprofile.BillingCycle `json:"billingCycle" validate:"required,oneof=PER_JOB QUARTERLY MONTHLY ANNUALLY"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the CustomerRuleProfileQuery when eager-loading is set.
	Edges        CustomerRuleProfileEdges `json:"edges"`
	selectValues sql.SelectValues
}

// CustomerRuleProfileEdges holds the relations/edges for other nodes in the graph.
type CustomerRuleProfileEdges struct {
	// BusinessUnit holds the value of the business_unit edge.
	BusinessUnit *BusinessUnit `json:"business_unit,omitempty"`
	// Organization holds the value of the organization edge.
	Organization *Organization `json:"organization,omitempty"`
	// Customer holds the value of the customer edge.
	Customer *Customer `json:"customer,omitempty"`
	// DocumentClassifications holds the value of the document_classifications edge.
	DocumentClassifications []*DocumentClassification `json:"document_classifications,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes                  [4]bool
	namedDocumentClassifications map[string][]*DocumentClassification
}

// BusinessUnitOrErr returns the BusinessUnit value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CustomerRuleProfileEdges) BusinessUnitOrErr() (*BusinessUnit, error) {
	if e.BusinessUnit != nil {
		return e.BusinessUnit, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: businessunit.Label}
	}
	return nil, &NotLoadedError{edge: "business_unit"}
}

// OrganizationOrErr returns the Organization value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CustomerRuleProfileEdges) OrganizationOrErr() (*Organization, error) {
	if e.Organization != nil {
		return e.Organization, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: organization.Label}
	}
	return nil, &NotLoadedError{edge: "organization"}
}

// CustomerOrErr returns the Customer value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CustomerRuleProfileEdges) CustomerOrErr() (*Customer, error) {
	if e.Customer != nil {
		return e.Customer, nil
	} else if e.loadedTypes[2] {
		return nil, &NotFoundError{label: customer.Label}
	}
	return nil, &NotLoadedError{edge: "customer"}
}

// DocumentClassificationsOrErr returns the DocumentClassifications value or an error if the edge
// was not loaded in eager-loading.
func (e CustomerRuleProfileEdges) DocumentClassificationsOrErr() ([]*DocumentClassification, error) {
	if e.loadedTypes[3] {
		return e.DocumentClassifications, nil
	}
	return nil, &NotLoadedError{edge: "document_classifications"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*CustomerRuleProfile) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case customerruleprofile.FieldVersion:
			values[i] = new(sql.NullInt64)
		case customerruleprofile.FieldBillingCycle:
			values[i] = new(sql.NullString)
		case customerruleprofile.FieldCreatedAt, customerruleprofile.FieldUpdatedAt:
			values[i] = new(sql.NullTime)
		case customerruleprofile.FieldID, customerruleprofile.FieldBusinessUnitID, customerruleprofile.FieldOrganizationID, customerruleprofile.FieldCustomerID:
			values[i] = new(uuid.UUID)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the CustomerRuleProfile fields.
func (crp *CustomerRuleProfile) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case customerruleprofile.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				crp.ID = *value
			}
		case customerruleprofile.FieldBusinessUnitID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field business_unit_id", values[i])
			} else if value != nil {
				crp.BusinessUnitID = *value
			}
		case customerruleprofile.FieldOrganizationID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field organization_id", values[i])
			} else if value != nil {
				crp.OrganizationID = *value
			}
		case customerruleprofile.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				crp.CreatedAt = value.Time
			}
		case customerruleprofile.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				crp.UpdatedAt = value.Time
			}
		case customerruleprofile.FieldVersion:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field version", values[i])
			} else if value.Valid {
				crp.Version = int(value.Int64)
			}
		case customerruleprofile.FieldCustomerID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field customer_id", values[i])
			} else if value != nil {
				crp.CustomerID = *value
			}
		case customerruleprofile.FieldBillingCycle:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_cycle", values[i])
			} else if value.Valid {
				crp.BillingCycle = customerruleprofile.BillingCycle(value.String)
			}
		default:
			crp.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the CustomerRuleProfile.
// This includes values selected through modifiers, order, etc.
func (crp *CustomerRuleProfile) Value(name string) (ent.Value, error) {
	return crp.selectValues.Get(name)
}

// QueryBusinessUnit queries the "business_unit" edge of the CustomerRuleProfile entity.
func (crp *CustomerRuleProfile) QueryBusinessUnit() *BusinessUnitQuery {
	return NewCustomerRuleProfileClient(crp.config).QueryBusinessUnit(crp)
}

// QueryOrganization queries the "organization" edge of the CustomerRuleProfile entity.
func (crp *CustomerRuleProfile) QueryOrganization() *OrganizationQuery {
	return NewCustomerRuleProfileClient(crp.config).QueryOrganization(crp)
}

// QueryCustomer queries the "customer" edge of the CustomerRuleProfile entity.
func (crp *CustomerRuleProfile) QueryCustomer() *CustomerQuery {
	return NewCustomerRuleProfileClient(crp.config).QueryCustomer(crp)
}

// QueryDocumentClassifications queries the "document_classifications" edge of the CustomerRuleProfile entity.
func (crp *CustomerRuleProfile) QueryDocumentClassifications() *DocumentClassificationQuery {
	return NewCustomerRuleProfileClient(crp.config).QueryDocumentClassifications(crp)
}

// Update returns a builder for updating this CustomerRuleProfile.
// Note that you need to call CustomerRuleProfile.Unwrap() before calling this method if this CustomerRuleProfile
// was returned from a transaction, and the transaction was committed or rolled back.
func (crp *CustomerRuleProfile) Update() *CustomerRuleProfileUpdateOne {
	return NewCustomerRuleProfileClient(crp.config).UpdateOne(crp)
}

// Unwrap unwraps the CustomerRuleProfile entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (crp *CustomerRuleProfile) Unwrap() *CustomerRuleProfile {
	_tx, ok := crp.config.driver.(*txDriver)
	if !ok {
		panic("ent: CustomerRuleProfile is not a transactional entity")
	}
	crp.config.driver = _tx.drv
	return crp
}

// String implements the fmt.Stringer.
func (crp *CustomerRuleProfile) String() string {
	var builder strings.Builder
	builder.WriteString("CustomerRuleProfile(")
	builder.WriteString(fmt.Sprintf("id=%v, ", crp.ID))
	builder.WriteString("business_unit_id=")
	builder.WriteString(fmt.Sprintf("%v", crp.BusinessUnitID))
	builder.WriteString(", ")
	builder.WriteString("organization_id=")
	builder.WriteString(fmt.Sprintf("%v", crp.OrganizationID))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(crp.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(crp.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("version=")
	builder.WriteString(fmt.Sprintf("%v", crp.Version))
	builder.WriteString(", ")
	builder.WriteString("customer_id=")
	builder.WriteString(fmt.Sprintf("%v", crp.CustomerID))
	builder.WriteString(", ")
	builder.WriteString("billing_cycle=")
	builder.WriteString(fmt.Sprintf("%v", crp.BillingCycle))
	builder.WriteByte(')')
	return builder.String()
}

// NamedDocumentClassifications returns the DocumentClassifications named value or an error if the edge was not
// loaded in eager-loading with this name.
func (crp *CustomerRuleProfile) NamedDocumentClassifications(name string) ([]*DocumentClassification, error) {
	if crp.Edges.namedDocumentClassifications == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := crp.Edges.namedDocumentClassifications[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (crp *CustomerRuleProfile) appendNamedDocumentClassifications(name string, edges ...*DocumentClassification) {
	if crp.Edges.namedDocumentClassifications == nil {
		crp.Edges.namedDocumentClassifications = make(map[string][]*DocumentClassification)
	}
	if len(edges) == 0 {
		crp.Edges.namedDocumentClassifications[name] = []*DocumentClassification{}
	} else {
		crp.Edges.namedDocumentClassifications[name] = append(crp.Edges.namedDocumentClassifications[name], edges...)
	}
}

// CustomerRuleProfiles is a parsable slice of CustomerRuleProfile.
type CustomerRuleProfiles []*CustomerRuleProfile
