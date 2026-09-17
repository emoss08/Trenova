package agentjobs

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const maxSubjectNotesChars = 12000

type SubjectContext struct {
	billingQueue serviceports.BillingQueueService
	shipments    serviceports.ShipmentService
	console      repositories.DispatchConsoleRepository
	logger       *zap.Logger
}

func (s *SubjectContext) Describe(
	ctx context.Context,
	tenant pagination.TenantInfo,
	subjectType agent.SubjectType,
	subjectID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	switch subjectType {
	case agent.SubjectBillingQueueItem:
		return s.billingQueueItem(ctx, tenant, subjectID)
	case agent.SubjectShipmentMove:
		return s.shipmentMove(ctx, tenant, subjectID)
	case agent.SubjectOrganization, "":
		return nil, nil
	default:
		return &agentdefinition.RuntimeSubject{
			Type:  subjectType,
			ID:    subjectID.String(),
			Label: string(subjectType),
		}, nil
	}
}

func (s *SubjectContext) billingQueueItem(
	ctx context.Context,
	tenant pagination.TenantInfo,
	itemID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	item, err := s.billingQueue.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		TenantInfo:            tenant,
		ItemID:                itemID,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, fmt.Errorf("load billing queue item: %w", err)
	}

	sections := map[string]any{"billingQueueItem": item}
	if item.ShipmentID.IsNotNil() && s.shipments != nil {
		readiness, rErr := s.shipments.GetBillingReadiness(ctx, item.ShipmentID, tenant)
		if rErr != nil {
			s.logger.Warn("agent subject: billing readiness unavailable", zap.Error(rErr))
		} else {
			sections["billingReadiness"] = map[string]any{
				"validationFailures":  readiness.ValidationFailures,
				"missingRequirements": readiness.MissingRequirements,
				"warnings":            readiness.Warnings,
				"serviceFailures":     readiness.ServiceFailureContext,
			}
		}
	}
	notes := strings.TrimSpace(
		strings.Join([]string{item.ReviewNotes, item.ExceptionNotes, item.CancelReason}, "\n"),
	)
	if notes != "" {
		sections["notes"] = notes
	}

	label := "Billing queue item"
	if item.ShipmentID.IsNotNil() {
		label = "Billing queue item for shipment " + item.ShipmentID.String()
	}

	return &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectBillingQueueItem,
		ID:    itemID.String(),
		Label: label,
		Notes: marshalNotes(sections),
	}, nil
}

func (s *SubjectContext) shipmentMove(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moveID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	if s.console == nil {
		return &agentdefinition.RuntimeSubject{
			Type:  agent.SubjectShipmentMove,
			ID:    moveID.String(),
			Label: "Shipment move",
		}, nil
	}

	moves, err := s.console.ListBoardMoves(ctx, &repositories.DispatchBoardFilter{
		TenantInfo: tenant,
		MoveIDs:    []pulid.ID{moveID},
		Limit:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("load shipment move: %w", err)
	}

	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectShipmentMove,
		ID:    moveID.String(),
		Label: "Shipment move",
	}
	if len(moves) > 0 {
		subject.Label = "Shipment move for PRO " + moves[0].ProNumber
		subject.Notes = marshalNotes(moves[0])
	}

	return subject, nil
}

func marshalNotes(value any) string {
	encoded, err := sonic.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	notes := string(encoded)
	if len(notes) > maxSubjectNotesChars {
		return notes[:maxSubjectNotesChars] + "\n…(truncated)"
	}

	return notes
}
