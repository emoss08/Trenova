package edimessageview

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListMessages(
	ctx context.Context,
	req *edimessageports.ListEDIMessagesRequest,
) (*pagination.ListResult[*edimessage.EDIMessage], error) {
	return s.messageRepo.ListMessages(ctx, req)
}

func (s *Service) GetMessage(
	ctx context.Context,
	req edimessageports.GetEDIMessageByIDRequest,
) (*edimessage.EDIMessage, error) {
	return s.messageRepo.GetMessageByID(ctx, req)
}

func (s *Service) InspectX12(
	ctx context.Context,
	req *InspectX12Request,
) (*edix12inspect.InspectX12Result, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"inspection",
			errortypes.ErrRequired,
			"Inspection request is required",
		)
	}
	return edix12inspect.InspectX12(ctx, &edix12inspect.InspectX12Request{
		RawX12:         req.RawX12,
		TransactionSet: req.TransactionSet,
		X12Version:     req.X12Version,
		Envelope:       req.Envelope,
		Diagnostics:    req.Diagnostics,
	})
}

func (s *Service) InspectMessage(
	ctx context.Context,
	req edimessageports.GetEDIMessageByIDRequest,
) (*EDIMessageInspection, error) {
	message, err := s.messageRepo.GetMessageByID(ctx, req)
	if err != nil {
		return nil, err
	}
	diagnostics := make([]edidocument.Diagnostic, 0, len(message.ValidationErrors))
	for _, diagnostic := range message.ValidationErrors {
		if strings.HasPrefix(diagnostic.Code, edix12inspect.FindingCodePrefix) {
			continue
		}
		diagnostics = append(diagnostics, edidocument.Diagnostic{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
			SuggestedFix:    diagnostic.SuggestedFix,
		})
	}
	inspection, err := edix12inspect.InspectX12(ctx, &edix12inspect.InspectX12Request{
		RawX12:         message.RawX12,
		TransactionSet: message.TransactionSet,
		X12Version:     message.X12Version,
		Diagnostics:    diagnostics,
	})
	if err != nil {
		return nil, err
	}
	return &EDIMessageInspection{
		Message:    message,
		Inspection: *inspection,
		Provenance: EDIInspectionProvenance{
			MessageID:       message.ID,
			LadingVersionID: message.LadingVersionID,
			FlowInstanceID:  message.FlowInstanceID,
			GeneratedAt:     message.GeneratedAt,
			GeneratedByID:   message.GeneratedByID,
		},
	}, nil
}
