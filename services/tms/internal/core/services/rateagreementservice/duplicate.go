package rateagreementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// DuplicateRateAgreementRequest copies a contract as the basis for a new one —
// the renewal workflow. Code and Name are optional; left empty, the copy names
// itself after the original.
type DuplicateRateAgreementRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	RateAgreementID pulid.ID              `json:"-"`

	Code string `json:"code"`
	Name string `json:"name"`
}

// Duplicate copies an agreement — lanes with their breaks, the accessorial
// schedule, the fuel binding — as a fresh Draft with none of the original's
// identity, history, or approvals. The copy goes through Create, so it is
// validated like anything typed in by hand and starts its own version chain
// at one.
func (s *Service) Duplicate(
	ctx context.Context,
	req *DuplicateRateAgreementRequest,
	userID pulid.ID,
) (*rateagreement.RateAgreement, error) {
	log := s.l.With(
		zap.String("operation", "Duplicate"),
		zap.String("sourceId", req.RateAgreementID.String()),
		zap.String("userId", userID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: req.RateAgreementID,
		TenantInfo:      req.TenantInfo,
		IncludeChildren: true,
	})
	if err != nil {
		log.Error("failed to load rate agreement to duplicate", zap.Error(err))
		return nil, err
	}

	copied, err := s.Create(ctx, copyAgreement(original, req), userID)
	if err != nil {
		return nil, err
	}

	s.audit(
		log, copied, nil, permission.OpDuplicate, userID,
		"Duplicated from "+original.Code,
	)

	return copied, nil
}

// copyAgreement builds the duplicate: the negotiated terms verbatim, every
// child re-minted without its identity, and nothing of the original's review
// history — a copy has not been approved by anyone.
func copyAgreement(
	original *rateagreement.RateAgreement,
	req *DuplicateRateAgreementRequest,
) *rateagreement.RateAgreement {
	duplicate := *original

	duplicate.ID = pulid.Nil
	duplicate.Code = req.Code
	if duplicate.Code == "" {
		duplicate.Code = original.Code + "-COPY"
	}
	duplicate.Name = req.Name
	if duplicate.Name == "" {
		duplicate.Name = original.Name + " (Copy)"
	}

	duplicate.Status = rateagreement.StatusDraft
	duplicate.SubmittedByID = nil
	duplicate.SubmittedAt = nil
	duplicate.ApprovedByID = nil
	duplicate.ApprovedAt = nil
	duplicate.ReviewComment = ""
	duplicate.CurrentVersionNumber = 1
	duplicate.Version = 0
	duplicate.CreatedAt = 0
	duplicate.UpdatedAt = 0

	duplicate.Rules = copyRules(original.Rules)
	duplicate.Accessorials = copyAccessorials(original.Accessorials)
	duplicate.FuelBinding = copyFuelBinding(original.FuelBinding)
	duplicate.Versions = nil

	return &duplicate
}

func copyRules(rules []*rateagreement.RateAgreementRule) []*rateagreement.RateAgreementRule {
	copied := make([]*rateagreement.RateAgreementRule, 0, len(rules))

	for _, rule := range rules {
		if rule == nil {
			continue
		}

		clone := *rule
		clone.ID = pulid.Nil
		clone.RateAgreementID = pulid.Nil
		// A copy has no lineage: its history starts here, in the new contract.
		clone.SupersedesRuleID = nil
		clone.Version = 0
		clone.CreatedAt = 0
		clone.UpdatedAt = 0

		clone.Breaks = make([]*rateagreement.RateAgreementRuleBreak, 0, len(rule.Breaks))
		for _, ruleBreak := range rule.Breaks {
			if ruleBreak == nil {
				continue
			}

			clonedBreak := *ruleBreak
			clonedBreak.ID = pulid.Nil
			clonedBreak.RateAgreementRuleID = pulid.Nil
			clonedBreak.CreatedAt = 0
			clonedBreak.UpdatedAt = 0
			clone.Breaks = append(clone.Breaks, &clonedBreak)
		}

		copied = append(copied, &clone)
	}

	return copied
}

func copyAccessorials(
	accessorials []*rateagreement.RateAgreementAccessorial,
) []*rateagreement.RateAgreementAccessorial {
	copied := make([]*rateagreement.RateAgreementAccessorial, 0, len(accessorials))

	for _, accessorial := range accessorials {
		if accessorial == nil {
			continue
		}

		clone := *accessorial
		clone.ID = pulid.Nil
		clone.RateAgreementID = pulid.Nil
		clone.Version = 0
		clone.CreatedAt = 0
		clone.UpdatedAt = 0
		copied = append(copied, &clone)
	}

	return copied
}

func copyFuelBinding(
	binding *rateagreement.RateAgreementFuelBinding,
) *rateagreement.RateAgreementFuelBinding {
	if binding == nil {
		return nil
	}

	clone := *binding
	clone.ID = pulid.Nil
	clone.RateAgreementID = pulid.Nil
	clone.Version = 0
	clone.CreatedAt = 0
	clone.UpdatedAt = 0

	return &clone
}
