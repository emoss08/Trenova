package types

import "github.com/google/uuid"

// FileFormat represents the format of the report file.
type FileFormat string

// DeliveryMethod represents the method of delivery for the report.
type DeliveryMethod string

// Constants for the file format of the report.
const (
	CSV  = FileFormat("csv")
	XLS  = FileFormat("xls")
	XLSX = FileFormat("xlsx")
	PDF  = FileFormat("pdf")
)

// Constants for the delivery method of the report.
const (
	Email = DeliveryMethod("email")
	Local = DeliveryMethod("local")
)

type RelationshipRequest struct {
	ForeignKey      string   `json:"foreignKey" validate:"omitempty"`
	ReferencedTable string   `json:"referencedTable" validate:"omitempty"`
	Columns         []string `json:"columns" validate:"omitempty"`
}

// GenerateReportRequest represents the payload for generating a report.
type GenerateReportRequest struct {
	TableName      string                `json:"tableName" validate:"required"`
	Columns        []string              `json:"columns" validate:"required"`
	Relationships  []RelationshipRequest `json:"relationships" validate:"omitempty"`
	FileFormat     FileFormat            `json:"fileFormat" validate:"required"`
	DeliveryMethod DeliveryMethod        `json:"deliveryMethod" validate:"required"`
	OrganizationID uuid.UUID             `json:"organizationId"`
	BusinessUnitID uuid.UUID             `json:"businessUnitId"`
}

// GenerateReportResponse represents the response for generating a report.
type GenerateReportResponse struct {
	ReportURL string `json:"report_url"`
}

type ColumnValue struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// Relationship represents a foreign key relationship between tables.
type Relationship struct {
	ForeignKey       string        `json:"foreignKey"`
	ReferencedTable  string        `json:"referencedTable"`
	ReferencedColumn string        `json:"referencedColumn"`
	Columns          []ColumnValue `json:"columns"`
}
