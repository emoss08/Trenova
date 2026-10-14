package editenderrepairservice

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/shared/pulid"
)

const defaultBatchSize = 200

type Deps struct {
	Repo ediports.TenderReferenceRepairRepository
}

type Service struct {
	repo ediports.TenderReferenceRepairRepository
}

type Request struct {
	OrganizationID pulid.ID
	DryRun         bool
	BatchSize      int
	OnRepair       func(Repair)
	OnSkip         func(Skip)
}

type Reference struct {
	ShipmentID    string
	PaymentMethod string
}

type Repair struct {
	MessageID      pulid.ID
	OrganizationID pulid.ID
	From           Reference
	To             Reference
}

type Skip struct {
	MessageID      pulid.ID
	OrganizationID pulid.ID
	Reason         string
}

type Report struct {
	Repaired          int
	Skipped           int
	Unchanged         int
	Unreadable        int
	TransfersRepaired int
	ShipmentsRepaired int
}

func New(deps Deps) *Service {
	return &Service{repo: deps.Repo}
}

func (s *Service) Repair(ctx context.Context, req *Request) (*Report, error) {
	batchSize := req.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	report := &Report{}
	afterID := pulid.Nil
	for {
		page, err := s.repo.ListTenderReferenceCandidates(
			ctx,
			&ediports.ListTenderReferenceCandidatesRequest{
				OrganizationID: req.OrganizationID,
				AfterID:        afterID,
				Limit:          batchSize,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("list inbound tenders to re-read: %w", err)
		}
		if len(page) == 0 {
			break
		}
		for _, candidate := range page {
			if err = s.repairOne(ctx, req, candidate, report); err != nil {
				return nil, err
			}
		}
		afterID = page[len(page)-1].MessageID
	}

	unreadable, err := s.repo.CountUnreadableTenders(
		ctx,
		&ediports.CountUnreadableTendersRequest{OrganizationID: req.OrganizationID},
	)
	if err != nil {
		return nil, fmt.Errorf("count inbound tenders whose X12 is gone: %w", err)
	}
	report.Unreadable = unreadable
	return report, nil
}

func (s *Service) repairOne(
	ctx context.Context,
	req *Request,
	candidate *ediports.TenderReferenceCandidate,
	report *Report,
) error {
	read, found := ediinboundservice.TenderReferences(
		candidate.RawX12,
		separatorsFor(candidate),
	)
	if !found {
		report.Skipped++
		if req.OnSkip != nil {
			req.OnSkip(Skip{
				MessageID:      candidate.MessageID,
				OrganizationID: candidate.OrganizationID,
				Reason:         "the kept transaction has no B2 segment",
			})
		}
		return nil
	}

	stored := Reference{ShipmentID: candidate.ShipmentID, PaymentMethod: candidate.PaymentMethod}
	reread := Reference{ShipmentID: read.ShipmentID, PaymentMethod: read.PaymentMethod}
	if stored == reread {
		report.Unchanged++
		return nil
	}

	report.Repaired++
	if req.OnRepair != nil {
		req.OnRepair(Repair{
			MessageID:      candidate.MessageID,
			OrganizationID: candidate.OrganizationID,
			From:           stored,
			To:             reread,
		})
	}
	if req.DryRun {
		return nil
	}

	result, err := s.repo.RepairTenderReference(ctx, &ediports.RepairTenderReferenceRequest{
		Candidate:     candidate,
		ShipmentID:    reread.ShipmentID,
		PaymentMethod: reread.PaymentMethod,
	})
	if err != nil {
		return fmt.Errorf("repair inbound tender %s: %w", candidate.MessageID, err)
	}
	report.TransfersRepaired += result.Transfers
	report.ShipmentsRepaired += result.Shipments
	return nil
}

func separatorsFor(candidate *ediports.TenderReferenceCandidate) *edi.X12EnvelopeSettings {
	if head := strings.TrimLeftFunc(candidate.FileHead, unicode.IsSpace); head != "" {
		separators, _ := edix12inspect.DetectSeparators(head, nil)
		if separators.Source == edix12inspect.SeparatorSourceISA {
			return &edi.X12EnvelopeSettings{
				ElementSeparator:    separators.Element,
				SegmentTerminator:   separators.Segment,
				ComponentSeparator:  separators.Component,
				RepetitionSeparator: separators.Repetition,
			}
		}
	}
	return transactionSeparators(candidate.RawX12)
}

func transactionSeparators(raw string) *edi.X12EnvelopeSettings {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 3 || !strings.HasPrefix(trimmed, "ST") {
		return nil
	}
	defaults := edi.DefaultX12EnvelopeSettings()
	return &edi.X12EnvelopeSettings{
		ElementSeparator:    trimmed[2:3],
		SegmentTerminator:   trimmed[len(trimmed)-1:],
		ComponentSeparator:  defaults.ComponentSeparator,
		RepetitionSeparator: defaults.RepetitionSeparator,
	}
}
