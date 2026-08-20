package rateagreementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// headerVersionIgnoreFields keeps the version diff to the negotiated terms:
// identity and bookkeeping belong to the version row, not to the contract.
var headerVersionIgnoreFields = []string{
	"id",
	"createdAt",
	"createdById",
	"versionNumber",
	"effectiveFrom",
	"effectiveTo",
	"changeMessage",
	"changeSummary",
	"status",
}

// headerTermsChanged diffs two readings of the same agreement down to the
// terms a version records. Rules are deliberately absent: they carry their own
// effective windows, and minting a header version for every lane edit would
// bury the renegotiations under the housekeeping.
func headerTermsChanged(
	original, updated *rateagreement.RateAgreement,
) (map[string]jsonutils.FieldChange, bool) {
	before := rateagreement.NewVersionFromAgreement(original, 0, 0, pulid.Nil, "", nil)
	after := rateagreement.NewVersionFromAgreement(updated, 0, 0, pulid.Nil, "", nil)

	changes, err := jsonutils.JSONDiff(before, after, &jsonutils.DiffOptions{
		IgnoreFields: headerVersionIgnoreFields,
	})
	if err != nil || len(changes) == 0 {
		return nil, false
	}

	return changes, true
}

// recordVersion snapshots the header terms as they now stand.
//
// A failure is logged rather than returned: the save itself has already
// committed, and unwinding a contract change because its bookkeeping row did
// not write would leave the caller believing terms that are live are not.
func (s *Service) recordVersion(
	ctx context.Context,
	log *zap.Logger,
	agreement *rateagreement.RateAgreement,
	effectiveFrom int64,
	userID pulid.ID,
	changeMessage string,
	changeSummary map[string]jsonutils.FieldChange,
) {
	version := rateagreement.NewVersionFromAgreement(
		agreement,
		agreement.CurrentVersionNumber,
		effectiveFrom,
		userID,
		changeMessage,
		changeSummary,
	)

	if _, err := s.repo.CreateVersion(ctx, version); err != nil {
		log.Error("failed to record rate agreement version", zap.Error(err))
	}
}
