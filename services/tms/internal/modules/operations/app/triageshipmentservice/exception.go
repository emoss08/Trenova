package triageshipmentservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	ExceptionHoldReasonCode = "OSD"
	maxHoldNoteRunes        = 2000
)

var exceptionLabels = map[inboundmessage.ExceptionKind]string{
	inboundmessage.ExceptionShortage: "Shortage",
	inboundmessage.ExceptionOverage:  "Overage",
	inboundmessage.ExceptionDamage:   "Damage",
}

func (s *Service) LogException(
	ctx context.Context,
	req *inboundmessageports.LogTriageExceptionRequest,
) error {
	reasonID, err := s.lookup.HoldReasonByCode(ctx, req.TenantInfo, ExceptionHoldReasonCode)
	if err != nil {
		return err
	}
	if reasonID.IsNil() {
		return errortypes.NewBusinessError(
			"Add an active hold reason with the code {0} to log shortages, overages and damage, "+
				"so billing holds the invoice until it is settled.",
			ExceptionHoldReasonCode,
		)
	}

	blocksBilling := true
	_, err = s.holds.Create(ctx, &shipmentholdports.CreateShipmentHoldRequest{
		TenantInfo:    req.TenantInfo,
		ShipmentID:    req.ShipmentID,
		HoldReasonID:  reasonID,
		Notes:         holdNote(req),
		BlocksBilling: &blocksBilling,
	}, req.Actor)

	return err
}

func holdNote(req *inboundmessageports.LogTriageExceptionRequest) string {
	lines := []string{
		fmt.Sprintf(
			"%s: %d %s",
			exceptionLabels[req.Kind],
			req.Quantity,
			strings.TrimSpace(req.Unit),
		),
	}
	if note := strings.TrimSpace(req.Note); note != "" {
		lines = append(lines, note)
	}
	if source := strings.TrimSpace(req.Source); source != "" {
		lines = append(lines, source)
	}

	note := strings.Join(lines, "\n")
	if runes := []rune(note); len(runes) > maxHoldNoteRunes {
		note = string(runes[:maxHoldNoteRunes])
	}

	return note
}
