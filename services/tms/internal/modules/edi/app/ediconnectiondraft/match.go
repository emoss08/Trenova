package ediconnectiondraft

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type identityHints struct {
	code         string
	name         string
	isaQualifier string
	isaID        string
	as2ID        string
}

func (e *evidence) hint(field string) string {
	spec, ok := fieldCatalog[field]
	if !ok {
		return ""
	}
	settled := resolveField(spec, e, "", nil, false)
	if settled.Status != ediconnectiondraftports.ValueStatusAccepted {
		return ""
	}
	return settled.Value
}

func (e *evidence) hints() *identityHints {
	return &identityHints{
		code:         e.hint(fieldPartnerCode),
		name:         e.hint(fieldPartnerName),
		isaQualifier: e.hint(fieldPartnerQualifier),
		isaID:        e.hint(fieldPartnerID),
		as2ID:        e.hint("as2.partnerAs2Id"),
	}
}

type partnerMatch struct {
	draft ediconnectiondraftports.DraftPartner
	setup *ediconnectiondraftports.EDIPartnerSetup
}

func (s *Service) matchPartner(
	ctx context.Context,
	req *ediconnectiondraftports.ConnectionDraftRequest,
	hints *identityHints,
) (*partnerMatch, error) {
	if req.PartnerID.IsNotNil() {
		return s.namedPartner(ctx, req)
	}

	matches, err := s.repo.FindPartnerMatches(
		ctx,
		&ediconnectiondraftports.FindEDIPartnerMatchesRequest{
			TenantInfo:   req.TenantInfo,
			Code:         hints.code,
			Name:         hints.name,
			ISAQualifier: hints.isaQualifier,
			ISAID:        hints.isaID,
			AS2ID:        hints.as2ID,
			CustomerID:   req.CustomerID,
		},
	)
	if err != nil {
		return nil, err
	}

	chosen, candidates := classifyMatches(matches)
	switch len(chosen) {
	case 0:
		return &partnerMatch{draft: ediconnectiondraftports.DraftPartner{
			Match:      ediconnectiondraftports.PartnerMatchNew,
			Candidates: candidates,
			Reasons:    newPartnerReasons(candidates),
		}}, nil
	case 1:
		match, matchErr := s.existingPartner(ctx, req, chosen[0].Partner.ID)
		if matchErr != nil {
			return nil, matchErr
		}
		match.draft.Reasons = matchReasons(chosen[0])
		return match, nil
	default:
		return &partnerMatch{draft: ediconnectiondraftports.DraftPartner{
			Match:      ediconnectiondraftports.PartnerMatchAmbiguous,
			Candidates: candidates,
			Reasons: []string{fmt.Sprintf(
				"%d partners match what the document says; name the one to update",
				len(chosen),
			)},
		}}, nil
	}
}

func (s *Service) namedPartner(
	ctx context.Context,
	req *ediconnectiondraftports.ConnectionDraftRequest,
) (*partnerMatch, error) {
	match, err := s.existingPartner(ctx, req, req.PartnerID)
	if err != nil {
		if dberror.IsNotFoundError(err) || errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"partnerId",
				errortypes.ErrNotFound,
				"No EDI partner with that ID exists in this organization",
			)
		}
		return nil, err
	}
	if match.setup.Partner.Kind != edi.PartnerKindExternal {
		return nil, errortypes.NewValidationError(
			"partnerId",
			errortypes.ErrInvalidOperation,
			"Only an outside trading partner is set up from a spec",
		)
	}
	match.draft.Reasons = []string{"Named for this draft"}
	return match, nil
}

func (s *Service) existingPartner(
	ctx context.Context,
	req *ediconnectiondraftports.ConnectionDraftRequest,
	partnerID pulid.ID,
) (*partnerMatch, error) {
	setup, err := s.repo.GetPartnerSetup(ctx, &ediconnectiondraftports.GetEDIPartnerSetupRequest{
		TenantInfo: req.TenantInfo,
		PartnerID:  partnerID,
	})
	if err != nil {
		return nil, err
	}
	return &partnerMatch{
		setup: setup,
		draft: ediconnectiondraftports.DraftPartner{
			Match:     ediconnectiondraftports.PartnerMatchExisting,
			PartnerID: setup.Partner.ID,
			Version:   setup.Partner.Version,
			Status:    setup.Partner.Status,
		},
	}, nil
}

func classifyMatches(
	matches []*ediconnectiondraftports.EDIPartnerMatch,
) (chosen []*ediconnectiondraftports.EDIPartnerMatch, candidates []ediconnectiondraftports.PartnerCandidate) {
	strong := make([]*ediconnectiondraftports.EDIPartnerMatch, 0, len(matches))
	byName := make([]*ediconnectiondraftports.EDIPartnerMatch, 0, 1)
	candidates = make([]ediconnectiondraftports.PartnerCandidate, 0, len(matches))
	for _, match := range matches {
		if match.Partner.Kind != edi.PartnerKindExternal {
			continue
		}
		candidates = append(candidates, ediconnectiondraftports.PartnerCandidate{
			PartnerID: match.Partner.ID,
			Code:      match.Partner.Code,
			Name:      match.Partner.Name,
			Reasons:   matchReasons(match),
		})
		switch {
		case match.ByCode || match.ByInterchange || match.ByAS2:
			strong = append(strong, match)
		case match.ByName:
			byName = append(byName, match)
		}
	}
	if len(strong) > 0 {
		return strong, candidates
	}
	return byName, candidates
}

func matchReasons(match *ediconnectiondraftports.EDIPartnerMatch) []string {
	reasons := make([]string, 0, 5)
	if match.ByCode {
		reasons = append(reasons, "Same partner code")
	}
	if match.ByInterchange {
		reasons = append(reasons, "Same ISA ID on a document profile")
	}
	if match.ByAS2 {
		reasons = append(reasons, "Same partner AS2 ID")
	}
	if match.ByName {
		reasons = append(reasons, "Same partner name")
	}
	if match.ByCustomer {
		reasons = append(reasons, "Linked to the same customer")
	}
	return reasons
}

func newPartnerReasons(candidates []ediconnectiondraftports.PartnerCandidate) []string {
	if len(candidates) == 0 {
		return []string{"No partner has this code, name, ISA ID or AS2 ID"}
	}
	return []string{
		"Other partners share this customer, but none has this code, name, ISA ID or AS2 ID",
	}
}

func (m *partnerMatch) partnerID() pulid.ID {
	if m.setup == nil {
		return pulid.Nil
	}
	return m.setup.Partner.ID
}
