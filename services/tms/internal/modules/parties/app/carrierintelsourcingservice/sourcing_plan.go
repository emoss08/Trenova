package carrierintelsourcingservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// ImportPlan is the carrier importing a prospect would make, as far as it can
// be known without asking the provider: the DOT number, checked for a carrier
// already holding it.
type ImportPlan struct {
	DOTNumber        string
	Code             string
	Provider         integration.Type
	EnrollMonitoring bool
}

func (s *Service) PlanImportProspect(
	ctx context.Context,
	req *ImportProspectRequest,
) (*ImportPlan, error) {
	dot := stringutils.DigitsOnly(req.DOTNumber)
	if dot == "" {
		return nil, errortypes.NewValidationError("dotNumber", errortypes.ErrRequired,
			"A DOT number is required to import a carrier")
	}
	existing, err := s.subjectRepo.ListExistingCarrierDOTs(ctx, req.TenantInfo, []string{dot})
	if err != nil {
		return nil, err
	}
	if id, found := existing[dot]; found {
		return nil, errortypes.NewBusinessError(
			"A carrier with DOT {0} already exists", dot,
		).WithParam("carrierId", id.String())
	}
	control, err := s.controlRepo.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return nil, errortypes.NewBusinessError(
			"Enable a carrier intelligence provider before importing carriers",
		)
	}

	return &ImportPlan{
		DOTNumber:        dot,
		Code:             strings.TrimSpace(req.Code),
		Provider:         provider,
		EnrollMonitoring: req.EnrollMonitoring,
	}, nil
}

type LookupProspectRequest struct {
	TenantInfo   pagination.TenantInfo
	DOTNumber    string
	DocketNumber string
	Depth        carrierintel.LookupDepth
}

func (s *Service) LookupProspect(
	ctx context.Context,
	req *LookupProspectRequest,
) (*carrierintelservice.FetchResult, map[string]pulid.ID, error) {
	dot := stringutils.DigitsOnly(req.DOTNumber)
	docket := stringutils.DigitsOnly(req.DocketNumber)
	if dot == "" && docket == "" {
		return nil, nil, errortypes.NewValidationError("dotNumber", errortypes.ErrRequired,
			"Enter a DOT or MC number")
	}

	subjectID := carrierintelsnapshot.ProspectSubjectID(dot)
	if dot == "" {
		subjectID = "MC" + docket
	}
	depth := req.Depth
	if !depth.IsValid() {
		depth = carrierintel.LookupDepthLite
	}

	result, err := s.intel.Fetch(ctx, &carrierintelservice.FetchRequest{
		TenantInfo: req.TenantInfo,
		Subject: carrierintelports.CarrierIntelSubject{
			SubjectType:  carrierintel.SubjectTypeProspect,
			SubjectID:    subjectID,
			DOTNumber:    dot,
			DocketNumber: docket,
		},
		MinDepth: depth,
		Purpose:  carrierintel.PurposeSourcing,
		Source:   carrierintelevent.EventSourceSnapshotDiff,
	})
	if err != nil {
		return nil, nil, err
	}

	existing := map[string]pulid.ID{}
	if dotFound := result.Snapshot.DOTNumber; dotFound != "" {
		existing, err = s.subjectRepo.ListExistingCarrierDOTs(
			ctx,
			req.TenantInfo,
			[]string{dotFound},
		)
		if err != nil {
			return nil, nil, err
		}
	}
	return result, existing, nil
}
