// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/invoicecontrol"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

// InvoiceControl is the model entity for the InvoiceControl schema.
type InvoiceControl struct {
	config `json:"-" validate:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// The time that this entity was created.
	CreatedAt time.Time `json:"createdAt" validate:"omitempty"`
	// The last time that this entity was updated.
	UpdatedAt time.Time `json:"updatedAt" validate:"omitempty"`
	// InvoiceNumberPrefix holds the value of the "invoice_number_prefix" field.
	InvoiceNumberPrefix string `json:"invoiceNumberPrefix"`
	// CreditMemoNumberPrefix holds the value of the "credit_memo_number_prefix" field.
	CreditMemoNumberPrefix string `json:"creditMemoNumberPrefix"`
	// InvoiceTerms holds the value of the "invoice_terms" field.
	InvoiceTerms string `json:"invoiceTerms"`
	// InvoiceFooter holds the value of the "invoice_footer" field.
	InvoiceFooter string `json:"invoiceFooter"`
	// InvoiceLogoURL holds the value of the "invoice_logo_url" field.
	InvoiceLogoURL string `json:"invoiceLogoUrl"`
	// InvoiceDateFormat holds the value of the "invoice_date_format" field.
	InvoiceDateFormat invoicecontrol.InvoiceDateFormat `json:"invoiceDateFormat"`
	// InvoiceDueAfterDays holds the value of the "invoice_due_after_days" field.
	InvoiceDueAfterDays uint8 `json:"invoiceDueAfterDays"`
	// InvoiceLogoWidth holds the value of the "invoice_logo_width" field.
	InvoiceLogoWidth uint16 `json:"invoiceLogoWidth"`
	// ShowAmountDue holds the value of the "show_amount_due" field.
	ShowAmountDue bool `json:"showAmountDue"`
	// AttachPdf holds the value of the "attach_pdf" field.
	AttachPdf bool `json:"attachPdf"`
	// ShowInvoiceDueDate holds the value of the "show_invoice_due_date" field.
	ShowInvoiceDueDate bool `json:"showInvoiceDueDate"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the InvoiceControlQuery when eager-loading is set.
	Edges            InvoiceControlEdges `json:"edges"`
	business_unit_id *uuid.UUID
	organization_id  *uuid.UUID
	selectValues     sql.SelectValues
}

// InvoiceControlEdges holds the relations/edges for other nodes in the graph.
type InvoiceControlEdges struct {
	// Organization holds the value of the organization edge.
	Organization *Organization `json:"organization,omitempty"`
	// BusinessUnit holds the value of the business_unit edge.
	BusinessUnit *BusinessUnit `json:"business_unit,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// OrganizationOrErr returns the Organization value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e InvoiceControlEdges) OrganizationOrErr() (*Organization, error) {
	if e.Organization != nil {
		return e.Organization, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: organization.Label}
	}
	return nil, &NotLoadedError{edge: "organization"}
}

// BusinessUnitOrErr returns the BusinessUnit value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e InvoiceControlEdges) BusinessUnitOrErr() (*BusinessUnit, error) {
	if e.BusinessUnit != nil {
		return e.BusinessUnit, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: businessunit.Label}
	}
	return nil, &NotLoadedError{edge: "business_unit"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*InvoiceControl) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case invoicecontrol.FieldShowAmountDue, invoicecontrol.FieldAttachPdf, invoicecontrol.FieldShowInvoiceDueDate:
			values[i] = new(sql.NullBool)
		case invoicecontrol.FieldInvoiceDueAfterDays, invoicecontrol.FieldInvoiceLogoWidth:
			values[i] = new(sql.NullInt64)
		case invoicecontrol.FieldInvoiceNumberPrefix, invoicecontrol.FieldCreditMemoNumberPrefix, invoicecontrol.FieldInvoiceTerms, invoicecontrol.FieldInvoiceFooter, invoicecontrol.FieldInvoiceLogoURL, invoicecontrol.FieldInvoiceDateFormat:
			values[i] = new(sql.NullString)
		case invoicecontrol.FieldCreatedAt, invoicecontrol.FieldUpdatedAt:
			values[i] = new(sql.NullTime)
		case invoicecontrol.FieldID:
			values[i] = new(uuid.UUID)
		case invoicecontrol.ForeignKeys[0]: // business_unit_id
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case invoicecontrol.ForeignKeys[1]: // organization_id
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the InvoiceControl fields.
func (ic *InvoiceControl) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case invoicecontrol.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				ic.ID = *value
			}
		case invoicecontrol.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				ic.CreatedAt = value.Time
			}
		case invoicecontrol.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				ic.UpdatedAt = value.Time
			}
		case invoicecontrol.FieldInvoiceNumberPrefix:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_number_prefix", values[i])
			} else if value.Valid {
				ic.InvoiceNumberPrefix = value.String
			}
		case invoicecontrol.FieldCreditMemoNumberPrefix:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field credit_memo_number_prefix", values[i])
			} else if value.Valid {
				ic.CreditMemoNumberPrefix = value.String
			}
		case invoicecontrol.FieldInvoiceTerms:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_terms", values[i])
			} else if value.Valid {
				ic.InvoiceTerms = value.String
			}
		case invoicecontrol.FieldInvoiceFooter:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_footer", values[i])
			} else if value.Valid {
				ic.InvoiceFooter = value.String
			}
		case invoicecontrol.FieldInvoiceLogoURL:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_logo_url", values[i])
			} else if value.Valid {
				ic.InvoiceLogoURL = value.String
			}
		case invoicecontrol.FieldInvoiceDateFormat:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_date_format", values[i])
			} else if value.Valid {
				ic.InvoiceDateFormat = invoicecontrol.InvoiceDateFormat(value.String)
			}
		case invoicecontrol.FieldInvoiceDueAfterDays:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_due_after_days", values[i])
			} else if value.Valid {
				ic.InvoiceDueAfterDays = uint8(value.Int64)
			}
		case invoicecontrol.FieldInvoiceLogoWidth:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_logo_width", values[i])
			} else if value.Valid {
				ic.InvoiceLogoWidth = uint16(value.Int64)
			}
		case invoicecontrol.FieldShowAmountDue:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field show_amount_due", values[i])
			} else if value.Valid {
				ic.ShowAmountDue = value.Bool
			}
		case invoicecontrol.FieldAttachPdf:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field attach_pdf", values[i])
			} else if value.Valid {
				ic.AttachPdf = value.Bool
			}
		case invoicecontrol.FieldShowInvoiceDueDate:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field show_invoice_due_date", values[i])
			} else if value.Valid {
				ic.ShowInvoiceDueDate = value.Bool
			}
		case invoicecontrol.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field business_unit_id", values[i])
			} else if value.Valid {
				ic.business_unit_id = new(uuid.UUID)
				*ic.business_unit_id = *value.S.(*uuid.UUID)
			}
		case invoicecontrol.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field organization_id", values[i])
			} else if value.Valid {
				ic.organization_id = new(uuid.UUID)
				*ic.organization_id = *value.S.(*uuid.UUID)
			}
		default:
			ic.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the InvoiceControl.
// This includes values selected through modifiers, order, etc.
func (ic *InvoiceControl) Value(name string) (ent.Value, error) {
	return ic.selectValues.Get(name)
}

// QueryOrganization queries the "organization" edge of the InvoiceControl entity.
func (ic *InvoiceControl) QueryOrganization() *OrganizationQuery {
	return NewInvoiceControlClient(ic.config).QueryOrganization(ic)
}

// QueryBusinessUnit queries the "business_unit" edge of the InvoiceControl entity.
func (ic *InvoiceControl) QueryBusinessUnit() *BusinessUnitQuery {
	return NewInvoiceControlClient(ic.config).QueryBusinessUnit(ic)
}

// Update returns a builder for updating this InvoiceControl.
// Note that you need to call InvoiceControl.Unwrap() before calling this method if this InvoiceControl
// was returned from a transaction, and the transaction was committed or rolled back.
func (ic *InvoiceControl) Update() *InvoiceControlUpdateOne {
	return NewInvoiceControlClient(ic.config).UpdateOne(ic)
}

// Unwrap unwraps the InvoiceControl entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (ic *InvoiceControl) Unwrap() *InvoiceControl {
	_tx, ok := ic.config.driver.(*txDriver)
	if !ok {
		panic("ent: InvoiceControl is not a transactional entity")
	}
	ic.config.driver = _tx.drv
	return ic
}

// String implements the fmt.Stringer.
func (ic *InvoiceControl) String() string {
	var builder strings.Builder
	builder.WriteString("InvoiceControl(")
	builder.WriteString(fmt.Sprintf("id=%v, ", ic.ID))
	builder.WriteString("created_at=")
	builder.WriteString(ic.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(ic.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("invoice_number_prefix=")
	builder.WriteString(ic.InvoiceNumberPrefix)
	builder.WriteString(", ")
	builder.WriteString("credit_memo_number_prefix=")
	builder.WriteString(ic.CreditMemoNumberPrefix)
	builder.WriteString(", ")
	builder.WriteString("invoice_terms=")
	builder.WriteString(ic.InvoiceTerms)
	builder.WriteString(", ")
	builder.WriteString("invoice_footer=")
	builder.WriteString(ic.InvoiceFooter)
	builder.WriteString(", ")
	builder.WriteString("invoice_logo_url=")
	builder.WriteString(ic.InvoiceLogoURL)
	builder.WriteString(", ")
	builder.WriteString("invoice_date_format=")
	builder.WriteString(fmt.Sprintf("%v", ic.InvoiceDateFormat))
	builder.WriteString(", ")
	builder.WriteString("invoice_due_after_days=")
	builder.WriteString(fmt.Sprintf("%v", ic.InvoiceDueAfterDays))
	builder.WriteString(", ")
	builder.WriteString("invoice_logo_width=")
	builder.WriteString(fmt.Sprintf("%v", ic.InvoiceLogoWidth))
	builder.WriteString(", ")
	builder.WriteString("show_amount_due=")
	builder.WriteString(fmt.Sprintf("%v", ic.ShowAmountDue))
	builder.WriteString(", ")
	builder.WriteString("attach_pdf=")
	builder.WriteString(fmt.Sprintf("%v", ic.AttachPdf))
	builder.WriteString(", ")
	builder.WriteString("show_invoice_due_date=")
	builder.WriteString(fmt.Sprintf("%v", ic.ShowInvoiceDueDate))
	builder.WriteByte(')')
	return builder.String()
}

// InvoiceControls is a parsable slice of InvoiceControl.
type InvoiceControls []*InvoiceControl
