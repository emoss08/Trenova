package models

import (
	"github.com/google/uuid"
)

type DateFormatType string

const (
	MmDdYyyy DateFormatType = "01/02/2006" // MM/DD/YYYY
	DdMmYyyy DateFormatType = "02/01/2006" // DD/MM/YYYY
	YyyyDdMm DateFormatType = "2006/02/01" // YYYY/DD/MM
	YyyyMmDd DateFormatType = "2006/01/02" // YYYY/MM/DD
)

type InvoiceControl struct {
	TimeStampedModel
	BusinessUnitID         uuid.UUID      `gorm:"type:uuid;not null;index"                                            json:"businessUnitId"`
	OrganizationID         uuid.UUID      `gorm:"type:uuid;not null;unique"                                           json:"organizationId"`
	InvoiceNumberPrefix    string         `gorm:"type:varchar(10);not null;default:'INV-'"                            json:"invoiceNumberPrefix"    validate:"required,max=10"`
	CreditMemoNumberPrefix string         `gorm:"type:varchar(10);not null;default:'CM-'"                             json:"creditMemoNumberPrefix" validate:"required,max=10"`
	InvoiceTerms           string         `gorm:"type:text"                                                           json:"invoiceTerms"           validate:"omitempty"`
	InvoiceFooter          string         `gorm:"type:text"                                                           json:"invoiceFooter"          validate:"omitempty"`
	InvoiceLogoURL         string         `gorm:"type:varchar(255);"                                                  json:"invoiceLogoUrl"         validate:"omitempty,url"`
	InvoiceDateFormat      DateFormatType `gorm:"type:varchar(10);not null;default:'01/02/2006'"                      json:"invoiceDateFormat"      validate:"required,oneof=01/02/2006 02/01/2006 2006/02/01 2006/01/02"`
	InvoiceDueAfterDays    uint8          `gorm:"type:smallint;check:invoice_due_after_days >= 0;not null;default:30" json:"invoiceDueAfterDays"    validate:"required"`
	InvoiceLogoWidth       uint16         `gorm:"type:smallint;check:invoice_logo_width >= 0;not null;default:100"    json:"invoiceLogoWidth"       validate:"required"`
	ShowAmountDue          bool           `gorm:"type:boolean;not null;default:true"                                  json:"showAmountDue"          validate:"omitempty"`
	AttachPDF              bool           `gorm:"type:boolean;not null;default:true"                                  json:"attachPdf"              validate:"omitempty"`
	ShowInvoiceDueDate     bool           `gorm:"type:boolean;not null;default:true"                                  json:"showInvoiceDueDate"     validate:"omitempty"`
}
