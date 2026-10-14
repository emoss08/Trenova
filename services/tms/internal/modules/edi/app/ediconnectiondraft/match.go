package ediconnectiondraft

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
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
	for _, c := range e.byField[field] {
		if c.status == ediports.ValueStatusAccepted {
			return c.value
		}
	}
	if resolution, ok := e.resolutions[field]; ok {
		if normalized, err := fieldCatalog[field].normalize(resolution.Value); err == nil {
			return normalized
		}
	}
	return ""
}

func (e *evidence) hints() identityHints {
	return identityHints{
		code:         e.hint(fieldPartnerCode),
		name:         e.hint(fieldPartnerName),
		isaQualifier: e.hint(fieldPartnerQualifier),
		isaID:        e.hint(fieldPartnerID),
		as2ID:        e.hint("as2.partnerAs2Id"),
	}
}

type partnerMatch struct {
	draft ediports.DraftPartner
	setup *ediports.EDIPartnerSetup
}

func (s *Service) matchPartner(
	ctx context.Context,
	req *ediports.ConnectionDraftRequest,
	hints identityHints,
) (*partnerMatch, error) {
	if req.PartnerID.IsNotNil() {
		setup, err := s.repo.GetPartnerSetup(ctx, &ediports.GetEDIPartnerSetupRequest{
			TenantInfo: req.TenantInfo,
			PartnerID:  req.PartnerID,
		})
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, errortypes.NewValidationError(
					"partnerId",
					errortypes.ErrNotFound,
					"No EDI partner with that ID exists in this organization",
				)
			}
			return nil, err
		}
		if setup.Partner.Kind != edi.PartnerKindExternal {
			return nil, errortypes.NewValidationError(
				"partnerId",
				errortypes.ErrInvalidOperation,
				"Only an outside trading partner is set up from a spec",
			)
		}
		return &partnerMatch{
			setup: setup,
			draft: ediports.DraftPartner{
				Match:     ediports.PartnerMatchExisting,
				PartnerID: setup.Partner.ID,
				Version:   setup.Partner.Version,
				Status:    setup.Partner.Status,
				Reasons:   []string{"Named for this draft"},
			},
		}, nil
	}

	matches, err := s.repo.FindPartnerMatches(ctx, &ediports.FindEDIPartnerMatchesRequest{
		TenantInfo:   req.TenantInfo,
		Code:         hints.code,
		Name:         hints.name,
		ISAQualifier: hints.isaQualifier,
		ISAID:        hints.isaID,
		AS2ID:        hints.as2ID,
		CustomerID:   req.CustomerID,
	})
	if err != nil {
		return nil, err
	}

	strong := make([]*ediports.EDIPartnerMatch, 0, len(matches))
	byName := make([]*ediports.EDIPartnerMatch, 0, 1)
	candidates := make([]ediports.PartnerCandidate, 0, len(matches))
	for _, match := range matches {
		if match.Partner.Kind != edi.PartnerKindExternal {
			continue
		}
		reasons := matchReasons(match)
		candidates = append(candidates, ediports.PartnerCandidate{
			PartnerID: match.Partner.ID,
			Code:      match.Partner.Code,
			Name:      match.Partner.Name,
			Reasons:   reasons,
		})
		switch {
		case match.ByCode || match.ByInterchange || match.ByAS2:
			strong = append(strong, match)
		case match.ByName:
			byName = append(byName, match)
		}
	}

	chosen := strong
	if len(chosen) == 0 {
		chosen = byName
	}
	switch len(chosen) {
	case 0:
		return &partnerMatch{draft: ediports.DraftPartner{
			Match:      ediports.PartnerMatchNew,
			Candidates: candidates,
			Reasons:    newPartnerReasons(candidates),
		}}, nil
	case 1:
		setup, setupErr := s.repo.GetPartnerSetup(ctx, &ediports.GetEDIPartnerSetupRequest{
			TenantInfo: req.TenantInfo,
			PartnerID:  chosen[0].Partner.ID,
		})
		if setupErr != nil {
			return nil, setupErr
		}
		return &partnerMatch{
			setup: setup,
			draft: ediports.DraftPartner{
				Match:     ediports.PartnerMatchExisting,
				PartnerID: setup.Partner.ID,
				Version:   setup.Partner.Version,
				Status:    setup.Partner.Status,
				Reasons:   matchReasons(chosen[0]),
			},
		}, nil
	default:
		return &partnerMatch{draft: ediports.DraftPartner{
			Match:      ediports.PartnerMatchAmbiguous,
			Candidates: candidates,
			Reasons: []string{fmt.Sprintf(
				"%d partners match what the document says; name the one to update",
				len(chosen),
			)},
		}}, nil
	}
}

func matchReasons(match *ediports.EDIPartnerMatch) []string {
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

func newPartnerReasons(candidates []ediports.PartnerCandidate) []string {
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
