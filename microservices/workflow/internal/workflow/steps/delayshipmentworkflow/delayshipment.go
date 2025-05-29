package delayshipmentworkflow

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/microservices/workflow/internal/email"
	"github.com/emoss08/trenova/microservices/workflow/internal/workflow/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/uptrace/bun"
)

func DelayShipments(
	db *bun.DB,
	emailClient *email.Client,
) func(worker.HatchetContext, *types.DelayShipmentsInput) (*types.DelayShipmentsOutput, error) {
	return func(hc worker.HatchetContext, _ *types.DelayShipmentsInput) (*types.DelayShipmentsOutput, error) {
		var queryStopsResults types.QueryStopsOutput
		if err := hc.StepOutput("query-stops", &queryStopsResults); err != nil {
			return nil, err
		}

		var queryControlsResults types.QueryShipmentControlsOutput
		if err := hc.StepOutput("get-shipment-controls", &queryControlsResults); err != nil {
			return nil, err
		}

		if len(queryStopsResults.PastDueStops) == 0 {
			hc.Log("no past due stops found, no shipments to delay")
			return &types.DelayShipmentsOutput{DelayedShipments: 0}, nil
		}

		// * Get unique shipment IDs from the moves
		type ShipmentMove struct {
			ShipmentID     string `bun:"shipment_id"`
			OrganizationID string `bun:"organization_id"`
		}

		// * Collect moveIDs from past due stops
		moveIDs := make([]string, 0, len(queryStopsResults.PastDueStops))
		for _, stop := range queryStopsResults.PastDueStops {
			moveIDs = append(moveIDs, stop.ShipmentMoveID)
		}

		hc.Log(fmt.Sprintf("querying for %d shipment moves", len(moveIDs)))

		// * Query to get the shipment IDs from the moves
		shipmentMoves := make([]ShipmentMove, 0)
		err := db.NewSelect().
			Table("shipment_moves").
			Column("shipment_id", "organization_id").
			Where("id IN (?)", bun.In(moveIDs)).
			Scan(hc, &shipmentMoves)
		if err != nil {
			hc.Log(fmt.Sprintf("error querying shipment moves: %v", err))
			return nil, err
		}

		hc.Log(fmt.Sprintf("found %d shipment moves", len(shipmentMoves)))

		if len(shipmentMoves) == 0 {
			hc.Log("no shipment moves found, no shipments to delay")
			return &types.DelayShipmentsOutput{DelayedShipments: 0}, nil
		}

		// Create a WITH clause for the shipment IDs and organization IDs
		// to use in the bulk update
		subquery := db.NewValues(&shipmentMoves)

		// Perform a bulk update for all shipments at once
		res, err := db.NewUpdate().
			With("_data", subquery).
			Table("shipments").
			Set("status = ?", "Delayed").
			Set("updated_at = ?", time.Now().Unix()).
			Where("id IN (SELECT shipment_id FROM _data)").
			Where("organization_id IN (SELECT organization_id FROM _data)").
			// Only update if currently in a status that can be changed to delayed
			Where("status IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit')").
			Exec(hc)
		if err != nil {
			hc.Log(fmt.Sprintf("error updating shipments: %v", err))
			return nil, err
		}

		rowsAffected, _ := res.RowsAffected()
		hc.Log(fmt.Sprintf("updated %d shipments to 'Delayed' status", rowsAffected))

		// Get organization details
		type Organization struct {
			ID   string `bun:"id"`
			Name string `bun:"name"`
		}

		// Build a map of organization IDs to organization details
		organizationsByID := make(map[string]string)
		for _, org := range queryControlsResults.Organizations {
			// Get organization name from database
			var organization Organization
			err = db.NewSelect().
				Table("organizations").
				Column("id", "name").
				Where("id = ?", org.OrganizationID).
				Scan(hc, &organization)

			if err != nil {
				hc.Log(fmt.Sprintf("error querying organization: %v", err))
				// Continue even if we can't get the organization name
				organizationsByID[org.OrganizationID] = org.OrganizationID
			} else {
				organizationsByID[org.OrganizationID] = organization.Name
			}
		}

		// If we have shipments that were delayed, send notification emails
		if rowsAffected > 0 && emailClient != nil {
			// Get admin users to notify for each organization
			for orgID, orgName := range organizationsByID {
				// Query admin users for this organization who should receive notifications
				type User struct {
					ID           string `bun:"id"`
					EmailAddress string `bun:"email_address"`
					Name         string `bun:"name"`
				}

				users := make([]User, 0)
				err = db.NewSelect().
					Table("users").
					Column("id", "email_address", "name").
					Where("current_organization_id = ?", orgID).
					Where("email_address IS NOT NULL").
					Limit(10). // Limit to avoid too many notifications
					Scan(hc, &users)

				if err != nil {
					hc.Log(fmt.Sprintf("error querying users for notifications: %v", err))
					continue
				}

				// If we found users to notify
				if len(users) > 0 {
					// Extract email addresses
					validEmails := make([]string, 0, len(users))
					for _, user := range users {
						if user.EmailAddress != "" {
							validEmails = append(validEmails, user.EmailAddress)
						}
					}

					// Skip if no valid emails were found
					if len(validEmails) == 0 {
						hc.Log(
							fmt.Sprintf(
								"no valid email addresses found for organization %s",
								orgID,
							),
						)
						continue
					}

					// Create notification data
					emailData := map[string]any{
						"DelayedShipments": rowsAffected,
						"Date":             time.Now().Format("January 2, 2006"),
						"OrganizationName": orgName,
						"Year":             time.Now().Year(),
						"LoginURL":         "https://app.trenova.io/login",
					}

					// Send the email notification
					if err = emailClient.SendEmail(
						hc,
						orgID,
						"workflow-system",
						"shipment",
						"shipment-delayed",
						fmt.Sprintf("Alert: %d Shipments Automatically Delayed", rowsAffected),
						validEmails,
						emailData,
					); err != nil {
						hc.Log(fmt.Sprintf("failed to send email notification: %v", err))
					} else {
						hc.Log(fmt.Sprintf("sent delayed shipments notification to %d users", len(validEmails)))
					}
				} else {
					hc.Log(fmt.Sprintf("no users found to notify for organization %s", orgID))
				}
			}
		}

		return &types.DelayShipmentsOutput{
			DelayedShipments: int(rowsAffected),
		}, nil
	}
}
