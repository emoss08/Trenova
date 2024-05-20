package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/config"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/models"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/imroc/req/v3"
	"github.com/rs/zerolog"
)

// ReportService represents the report service.
type ReportService struct {
	Client       *ent.Client
	Logger       *zerolog.Logger
	Config       *config.Server
	QueryService *models.QueryService // QueryService provides methods for querying the database.
}

// NewReportService creates a new report service.
func NewReportService(s *api.Server) *ReportService {
	return &ReportService{
		Client: s.Client,
		Logger: s.Logger,
		Config: &s.Config,
	}
}

// GetColumnsByTableName returns the column names and relationships for a given table name.
//
// This function is used to retrieve the column names and relationships for a given table name. It will exclude any columns
func (r *ReportService) GetColumnsByTableName(ctx context.Context, tableName string) ([]types.ColumnValue, []types.Relationship, int, error) {
	excludedTableNames := map[string]bool{
		"table_change_alerts":       true,
		"shipment_controls":         true,
		"billing_controls":          true,
		"sessions":                  true,
		"organizations":             true,
		"business_units":            true,
		"feasibility_tool_controls": true,
		"users":                     true,
		"user_favorites":            true,
		"us_states":                 true,
		"invoice_controls":          true,
		"email_controls":            true,
		"route_controls":            true,
		"accounting_controls":       true,
		"email_profiles":            true,
	}

	excludedColumns := map[string]bool{
		"id":               true,
		"business_unit_id": true,
		"organization_id":  true,
	}

	columns, relationships, _, err := r.QueryService.GetColumnsAndRelationships(ctx, tableName, excludedTableNames, excludedColumns)
	if err != nil {
		r.Logger.Err(err).Msg("Failed to get columns and relationships")
		return nil, nil, 0, err
	}

	return columns, relationships, len(columns), nil
}

// GenerateReport creates a report based on the provided payload.
//
// This function is used to generate a report based on the provided payload. The report will be generated
// based on the table name, columns, file format, and delivery method provided in the payload. The report
// will be sent to the user via the delivery method specified in the payload.
func (r *ReportService) GenerateReport(
	ctx context.Context, payload types.GenerateReportRequest, userID, orgID, buID uuid.UUID,
) (types.GenerateReportResponse, error) {
	client := req.C().
		SetTimeout(10 * time.Second)

	var result types.GenerateReportResponse

	resp, err := client.R().
		SetBody(&payload).
		SetSuccessResult(&result).
		Post(r.Config.Integration.GenerateReportEndpoint)
	if err != nil {
		return types.GenerateReportResponse{}, err
	}

	if resp.IsErrorState() {
		return types.GenerateReportResponse{}, nil
	}

	if resp.IsSuccessState() {
		err = util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
			message := Message{
				Type:     "success",
				Content:  "Report job completed successfully. Check your inbox for the requested report.",
				ClientID: userID.String(),
			}

			err = r.addReportToUser(ctx, tx, userID, orgID, buID, result.ReportURL)
			if err != nil {
				r.Logger.Err(err).Msg("Failed to add report to user")
				return err
			}

			// Send the Notify client message in goroutine
			go func() {
				// Notify the client that the report has been generated
				NewWebsocketService(r.Client, r.Logger).NotifyClient(userID.String(), message)
			}()

			return nil
		})
		if err != nil {
			return types.GenerateReportResponse{}, err
		}

		return result, nil
	}

	return types.GenerateReportResponse{}, nil
}

// addReportToUser adds the report URL to the user's report list.
//
// This function is used to add the report URL to the user's report list. The report URL is the URL where the
// report can be downloaded from. The report URL is stored in the user_report table in the database.
func (r *ReportService) addReportToUser(
	ctx context.Context, tx *ent.Tx, userID, orgID, buID uuid.UUID, reportURL string,
) error {
	return tx.UserReport.Create().
		SetOrganizationID(orgID).
		SetBusinessUnitID(buID).
		SetUserID(userID).
		SetReportURL(reportURL).
		Exec(ctx)
}
