// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/ent/accountingcontrol"
	"github.com/emoss08/trenova/internal/ent/billingcontrol"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/dispatchcontrol"
	"github.com/emoss08/trenova/internal/ent/emailcontrol"
	"github.com/emoss08/trenova/internal/ent/feasibilitytoolcontrol"
	"github.com/emoss08/trenova/internal/ent/googleapi"
	"github.com/emoss08/trenova/internal/ent/invoicecontrol"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/routecontrol"
	"github.com/emoss08/trenova/internal/ent/shipmentcontrol"
	"github.com/google/uuid"
)

// Organization is the model entity for the Organization schema.
type Organization struct {
	config `json:"-" validate:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// The time that this entity was created.
	CreatedAt time.Time `json:"createdAt" validate:"omitempty"`
	// The last time that this entity was updated.
	UpdatedAt time.Time `json:"updatedAt" validate:"omitempty"`
	// BusinessUnitID holds the value of the "business_unit_id" field.
	BusinessUnitID uuid.UUID `json:"businessUnitId"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// ScacCode holds the value of the "scac_code" field.
	ScacCode string `json:"scacCode"`
	// DotNumber holds the value of the "dot_number" field.
	DotNumber string `json:"dotNumber"`
	// LogoURL holds the value of the "logo_url" field.
	LogoURL string `json:"logoUrl"`
	// OrgType holds the value of the "org_type" field.
	OrgType organization.OrgType `json:"orgType"`
	// Timezone holds the value of the "timezone" field.
	Timezone string `json:"timezone" validate:"required,timezone"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the OrganizationQuery when eager-loading is set.
	Edges        OrganizationEdges `json:"edges"`
	selectValues sql.SelectValues
}

// OrganizationEdges holds the relations/edges for other nodes in the graph.
type OrganizationEdges struct {
	// BusinessUnit holds the value of the business_unit edge.
	BusinessUnit *BusinessUnit `json:"business_unit,omitempty"`
	// OrganizationFeatureFlag holds the value of the organization_feature_flag edge.
	OrganizationFeatureFlag []*OrganizationFeatureFlag `json:"organization_feature_flag,omitempty"`
	// Shipments holds the value of the shipments edge.
	Shipments []*Shipment `json:"shipments,omitempty"`
	// AccountingControl holds the value of the accounting_control edge.
	AccountingControl *AccountingControl `json:"accounting_control,omitempty"`
	// BillingControl holds the value of the billing_control edge.
	BillingControl *BillingControl `json:"billing_control,omitempty"`
	// DispatchControl holds the value of the dispatch_control edge.
	DispatchControl *DispatchControl `json:"dispatch_control,omitempty"`
	// FeasibilityToolControl holds the value of the feasibility_tool_control edge.
	FeasibilityToolControl *FeasibilityToolControl `json:"feasibility_tool_control,omitempty"`
	// InvoiceControl holds the value of the invoice_control edge.
	InvoiceControl *InvoiceControl `json:"invoice_control,omitempty"`
	// RouteControl holds the value of the route_control edge.
	RouteControl *RouteControl `json:"route_control,omitempty"`
	// ShipmentControl holds the value of the shipment_control edge.
	ShipmentControl *ShipmentControl `json:"shipment_control,omitempty"`
	// EmailControl holds the value of the email_control edge.
	EmailControl *EmailControl `json:"email_control,omitempty"`
	// GoogleAPI holds the value of the google_api edge.
	GoogleAPI *GoogleApi `json:"google_api,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes                  [12]bool
	namedOrganizationFeatureFlag map[string][]*OrganizationFeatureFlag
	namedShipments               map[string][]*Shipment
}

// BusinessUnitOrErr returns the BusinessUnit value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) BusinessUnitOrErr() (*BusinessUnit, error) {
	if e.BusinessUnit != nil {
		return e.BusinessUnit, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: businessunit.Label}
	}
	return nil, &NotLoadedError{edge: "business_unit"}
}

// OrganizationFeatureFlagOrErr returns the OrganizationFeatureFlag value or an error if the edge
// was not loaded in eager-loading.
func (e OrganizationEdges) OrganizationFeatureFlagOrErr() ([]*OrganizationFeatureFlag, error) {
	if e.loadedTypes[1] {
		return e.OrganizationFeatureFlag, nil
	}
	return nil, &NotLoadedError{edge: "organization_feature_flag"}
}

// ShipmentsOrErr returns the Shipments value or an error if the edge
// was not loaded in eager-loading.
func (e OrganizationEdges) ShipmentsOrErr() ([]*Shipment, error) {
	if e.loadedTypes[2] {
		return e.Shipments, nil
	}
	return nil, &NotLoadedError{edge: "shipments"}
}

// AccountingControlOrErr returns the AccountingControl value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) AccountingControlOrErr() (*AccountingControl, error) {
	if e.AccountingControl != nil {
		return e.AccountingControl, nil
	} else if e.loadedTypes[3] {
		return nil, &NotFoundError{label: accountingcontrol.Label}
	}
	return nil, &NotLoadedError{edge: "accounting_control"}
}

// BillingControlOrErr returns the BillingControl value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) BillingControlOrErr() (*BillingControl, error) {
	if e.BillingControl != nil {
		return e.BillingControl, nil
	} else if e.loadedTypes[4] {
		return nil, &NotFoundError{label: billingcontrol.Label}
	}
	return nil, &NotLoadedError{edge: "billing_control"}
}

// DispatchControlOrErr returns the DispatchControl value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) DispatchControlOrErr() (*DispatchControl, error) {
	if e.DispatchControl != nil {
		return e.DispatchControl, nil
	} else if e.loadedTypes[5] {
		return nil, &NotFoundError{label: dispatchcontrol.Label}
	}
	return nil, &NotLoadedError{edge: "dispatch_control"}
}

// FeasibilityToolControlOrErr returns the FeasibilityToolControl value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) FeasibilityToolControlOrErr() (*FeasibilityToolControl, error) {
	if e.FeasibilityToolControl != nil {
		return e.FeasibilityToolControl, nil
	} else if e.loadedTypes[6] {
		return nil, &NotFoundError{label: feasibilitytoolcontrol.Label}
	}
	return nil, &NotLoadedError{edge: "feasibility_tool_control"}
}

// InvoiceControlOrErr returns the InvoiceControl value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) InvoiceControlOrErr() (*InvoiceControl, error) {
	if e.InvoiceControl != nil {
		return e.InvoiceControl, nil
	} else if e.loadedTypes[7] {
		return nil, &NotFoundError{label: invoicecontrol.Label}
	}
	return nil, &NotLoadedError{edge: "invoice_control"}
}

// RouteControlOrErr returns the RouteControl value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) RouteControlOrErr() (*RouteControl, error) {
	if e.RouteControl != nil {
		return e.RouteControl, nil
	} else if e.loadedTypes[8] {
		return nil, &NotFoundError{label: routecontrol.Label}
	}
	return nil, &NotLoadedError{edge: "route_control"}
}

// ShipmentControlOrErr returns the ShipmentControl value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) ShipmentControlOrErr() (*ShipmentControl, error) {
	if e.ShipmentControl != nil {
		return e.ShipmentControl, nil
	} else if e.loadedTypes[9] {
		return nil, &NotFoundError{label: shipmentcontrol.Label}
	}
	return nil, &NotLoadedError{edge: "shipment_control"}
}

// EmailControlOrErr returns the EmailControl value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) EmailControlOrErr() (*EmailControl, error) {
	if e.EmailControl != nil {
		return e.EmailControl, nil
	} else if e.loadedTypes[10] {
		return nil, &NotFoundError{label: emailcontrol.Label}
	}
	return nil, &NotLoadedError{edge: "email_control"}
}

// GoogleAPIOrErr returns the GoogleAPI value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e OrganizationEdges) GoogleAPIOrErr() (*GoogleApi, error) {
	if e.GoogleAPI != nil {
		return e.GoogleAPI, nil
	} else if e.loadedTypes[11] {
		return nil, &NotFoundError{label: googleapi.Label}
	}
	return nil, &NotLoadedError{edge: "google_api"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Organization) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case organization.FieldName, organization.FieldScacCode, organization.FieldDotNumber, organization.FieldLogoURL, organization.FieldOrgType, organization.FieldTimezone:
			values[i] = new(sql.NullString)
		case organization.FieldCreatedAt, organization.FieldUpdatedAt:
			values[i] = new(sql.NullTime)
		case organization.FieldID, organization.FieldBusinessUnitID:
			values[i] = new(uuid.UUID)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Organization fields.
func (o *Organization) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case organization.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				o.ID = *value
			}
		case organization.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				o.CreatedAt = value.Time
			}
		case organization.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				o.UpdatedAt = value.Time
			}
		case organization.FieldBusinessUnitID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field business_unit_id", values[i])
			} else if value != nil {
				o.BusinessUnitID = *value
			}
		case organization.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				o.Name = value.String
			}
		case organization.FieldScacCode:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field scac_code", values[i])
			} else if value.Valid {
				o.ScacCode = value.String
			}
		case organization.FieldDotNumber:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field dot_number", values[i])
			} else if value.Valid {
				o.DotNumber = value.String
			}
		case organization.FieldLogoURL:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field logo_url", values[i])
			} else if value.Valid {
				o.LogoURL = value.String
			}
		case organization.FieldOrgType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field org_type", values[i])
			} else if value.Valid {
				o.OrgType = organization.OrgType(value.String)
			}
		case organization.FieldTimezone:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field timezone", values[i])
			} else if value.Valid {
				o.Timezone = value.String
			}
		default:
			o.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Organization.
// This includes values selected through modifiers, order, etc.
func (o *Organization) Value(name string) (ent.Value, error) {
	return o.selectValues.Get(name)
}

// QueryBusinessUnit queries the "business_unit" edge of the Organization entity.
func (o *Organization) QueryBusinessUnit() *BusinessUnitQuery {
	return NewOrganizationClient(o.config).QueryBusinessUnit(o)
}

// QueryOrganizationFeatureFlag queries the "organization_feature_flag" edge of the Organization entity.
func (o *Organization) QueryOrganizationFeatureFlag() *OrganizationFeatureFlagQuery {
	return NewOrganizationClient(o.config).QueryOrganizationFeatureFlag(o)
}

// QueryShipments queries the "shipments" edge of the Organization entity.
func (o *Organization) QueryShipments() *ShipmentQuery {
	return NewOrganizationClient(o.config).QueryShipments(o)
}

// QueryAccountingControl queries the "accounting_control" edge of the Organization entity.
func (o *Organization) QueryAccountingControl() *AccountingControlQuery {
	return NewOrganizationClient(o.config).QueryAccountingControl(o)
}

// QueryBillingControl queries the "billing_control" edge of the Organization entity.
func (o *Organization) QueryBillingControl() *BillingControlQuery {
	return NewOrganizationClient(o.config).QueryBillingControl(o)
}

// QueryDispatchControl queries the "dispatch_control" edge of the Organization entity.
func (o *Organization) QueryDispatchControl() *DispatchControlQuery {
	return NewOrganizationClient(o.config).QueryDispatchControl(o)
}

// QueryFeasibilityToolControl queries the "feasibility_tool_control" edge of the Organization entity.
func (o *Organization) QueryFeasibilityToolControl() *FeasibilityToolControlQuery {
	return NewOrganizationClient(o.config).QueryFeasibilityToolControl(o)
}

// QueryInvoiceControl queries the "invoice_control" edge of the Organization entity.
func (o *Organization) QueryInvoiceControl() *InvoiceControlQuery {
	return NewOrganizationClient(o.config).QueryInvoiceControl(o)
}

// QueryRouteControl queries the "route_control" edge of the Organization entity.
func (o *Organization) QueryRouteControl() *RouteControlQuery {
	return NewOrganizationClient(o.config).QueryRouteControl(o)
}

// QueryShipmentControl queries the "shipment_control" edge of the Organization entity.
func (o *Organization) QueryShipmentControl() *ShipmentControlQuery {
	return NewOrganizationClient(o.config).QueryShipmentControl(o)
}

// QueryEmailControl queries the "email_control" edge of the Organization entity.
func (o *Organization) QueryEmailControl() *EmailControlQuery {
	return NewOrganizationClient(o.config).QueryEmailControl(o)
}

// QueryGoogleAPI queries the "google_api" edge of the Organization entity.
func (o *Organization) QueryGoogleAPI() *GoogleApiQuery {
	return NewOrganizationClient(o.config).QueryGoogleAPI(o)
}

// Update returns a builder for updating this Organization.
// Note that you need to call Organization.Unwrap() before calling this method if this Organization
// was returned from a transaction, and the transaction was committed or rolled back.
func (o *Organization) Update() *OrganizationUpdateOne {
	return NewOrganizationClient(o.config).UpdateOne(o)
}

// Unwrap unwraps the Organization entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (o *Organization) Unwrap() *Organization {
	_tx, ok := o.config.driver.(*txDriver)
	if !ok {
		panic("ent: Organization is not a transactional entity")
	}
	o.config.driver = _tx.drv
	return o
}

// String implements the fmt.Stringer.
func (o *Organization) String() string {
	var builder strings.Builder
	builder.WriteString("Organization(")
	builder.WriteString(fmt.Sprintf("id=%v, ", o.ID))
	builder.WriteString("created_at=")
	builder.WriteString(o.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(o.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("business_unit_id=")
	builder.WriteString(fmt.Sprintf("%v", o.BusinessUnitID))
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(o.Name)
	builder.WriteString(", ")
	builder.WriteString("scac_code=")
	builder.WriteString(o.ScacCode)
	builder.WriteString(", ")
	builder.WriteString("dot_number=")
	builder.WriteString(o.DotNumber)
	builder.WriteString(", ")
	builder.WriteString("logo_url=")
	builder.WriteString(o.LogoURL)
	builder.WriteString(", ")
	builder.WriteString("org_type=")
	builder.WriteString(fmt.Sprintf("%v", o.OrgType))
	builder.WriteString(", ")
	builder.WriteString("timezone=")
	builder.WriteString(o.Timezone)
	builder.WriteByte(')')
	return builder.String()
}

// NamedOrganizationFeatureFlag returns the OrganizationFeatureFlag named value or an error if the edge was not
// loaded in eager-loading with this name.
func (o *Organization) NamedOrganizationFeatureFlag(name string) ([]*OrganizationFeatureFlag, error) {
	if o.Edges.namedOrganizationFeatureFlag == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := o.Edges.namedOrganizationFeatureFlag[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (o *Organization) appendNamedOrganizationFeatureFlag(name string, edges ...*OrganizationFeatureFlag) {
	if o.Edges.namedOrganizationFeatureFlag == nil {
		o.Edges.namedOrganizationFeatureFlag = make(map[string][]*OrganizationFeatureFlag)
	}
	if len(edges) == 0 {
		o.Edges.namedOrganizationFeatureFlag[name] = []*OrganizationFeatureFlag{}
	} else {
		o.Edges.namedOrganizationFeatureFlag[name] = append(o.Edges.namedOrganizationFeatureFlag[name], edges...)
	}
}

// NamedShipments returns the Shipments named value or an error if the edge was not
// loaded in eager-loading with this name.
func (o *Organization) NamedShipments(name string) ([]*Shipment, error) {
	if o.Edges.namedShipments == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := o.Edges.namedShipments[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (o *Organization) appendNamedShipments(name string, edges ...*Shipment) {
	if o.Edges.namedShipments == nil {
		o.Edges.namedShipments = make(map[string][]*Shipment)
	}
	if len(edges) == 0 {
		o.Edges.namedShipments[name] = []*Shipment{}
	} else {
		o.Edges.namedShipments[name] = append(o.Edges.namedShipments[name], edges...)
	}
}

// Organizations is a parsable slice of Organization.
type Organizations []*Organization
