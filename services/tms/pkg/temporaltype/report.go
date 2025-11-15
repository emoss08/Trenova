package temporaltype

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GenerateReportPayload struct {
	ReportID       pulid.ID                `json:"reportId"`
	OrganizationID pulid.ID                `json:"organizationId"`
	BusinessUnitID pulid.ID                `json:"businessUnitId"`
	UserID         pulid.ID                `json:"userId"`
	UserEmail      string                  `json:"userEmail"`
	ResourceType   string                  `json:"resourceType"`
	Format         report.Format           `json:"format"`
	DeliveryMethod report.DeliveryMethod   `json:"deliveryMethod"`
	FilterState    pagination.QueryOptions `json:"filterState"`
	EmailProfileID *pulid.ID               `json:"emailProfileId,omitempty"`
	Metadata       map[string]any          `json:"metadata,omitempty"`
}

type ReportResult struct {
	ReportID       pulid.ID      `json:"reportId"`
	OrganizationID pulid.ID      `json:"organizationId"`
	BusinessUnitID pulid.ID      `json:"businessUnitId"`
	UserID         pulid.ID      `json:"userId"`
	FilePath       string        `json:"filePath"`
	FileName       string        `json:"fileName"`
	FileSize       int64         `json:"fileSize"`
	RowCount       int           `json:"rowCount"`
	Status         report.Status `json:"status"`
	ErrorMsg       string        `json:"errorMsg,omitempty"`
	Timestamp      int64         `json:"timestamp"`
}

type QueryExecutionResult struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Total   int              `json:"total"`
}
