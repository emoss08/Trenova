package modeprofileservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingDeviationRepo struct {
	repositories.DeviationRepository

	liveKeys   []string
	recorded   []*modeprofile.Deviation
	superseded *repositories.SupersedeDeviationsRequest
}

func (r *recordingDeviationRepo) LiveRuleKeysForResource(
	_ context.Context,
	_ *repositories.LiveDeviationKeysRequest,
) ([]string, error) {
	return r.liveKeys, nil
}

func (r *recordingDeviationRepo) SupersedeOpenForResource(
	_ context.Context,
	req *repositories.SupersedeDeviationsRequest,
) error {
	r.superseded = req

	return nil
}

func (r *recordingDeviationRepo) RecordMany(
	_ context.Context,
	deviations []*modeprofile.Deviation,
) error {
	r.recorded = append(r.recorded, deviations...)

	return nil
}

func policyWith(keys ...modeprofile.RuleKey) *modeprofile.ResolvedPolicy {
	rules := make(map[modeprofile.RuleKey]modeprofile.ResolvedRule, len(keys))
	for _, key := range keys {
		rules[key] = modeprofile.ResolvedRule{
			Key:        key,
			Capability: modeprofile.CapabilityTemperatureControl,
		}
	}

	return &modeprofile.ResolvedPolicy{ProfileID: pulid.MustNew("mpf_"), Rules: rules}
}

func recordRequest(
	policy *modeprofile.ResolvedPolicy,
	keys ...modeprofile.RuleKey,
) *services.RecordDeviationsRequest {
	advisories := make([]*errortypes.AdvisoryError, 0, len(keys))
	for _, key := range keys {
		advisories = append(advisories, &errortypes.AdvisoryError{
			RuleKey:  string(key),
			Message:  "advisory for " + string(key),
			Severity: errortypes.SeverityWarn,
		})
	}

	return &services.RecordDeviationsRequest{
		TenantInfo:   pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		ResourceType: ResourceTypeShipment,
		ResourceID:   pulid.MustNew("shp_"),
		Policy:       policy,
		Advisories:   advisories,
	}
}

func newTestService(repo repositories.DeviationRepository) *service {
	return &service{deviationRepo: repo, l: zap.NewNop()}
}

// The rule keys that fired have to reach the repository, or every still-firing
// deviation is resolved and re-inserted, resetting OccurredAt and counting
// another occurrence in the ledger on each evaluation.
func TestRecordDeviations_KeepsTheRulesThatFiredAgain(t *testing.T) {
	repo := &recordingDeviationRepo{}
	policy := policyWith(modeprofile.RuleKeyTemperatureRange, modeprofile.RuleKeyHazmatSegregation)

	err := newTestService(repo).RecordDeviations(
		t.Context(),
		recordRequest(policy, modeprofile.RuleKeyTemperatureRange, modeprofile.RuleKeyHazmatSegregation),
	)
	require.NoError(t, err)
	require.NotNil(t, repo.superseded)

	assert.ElementsMatch(t,
		[]string{
			string(modeprofile.RuleKeyTemperatureRange),
			string(modeprofile.RuleKeyHazmatSegregation),
		},
		repo.superseded.KeepRuleKeys,
	)
}

// A deviation already on record keeps its row. Re-inserting it would leave the
// resource carrying the same finding twice — and where the operator had
// acknowledged it, the copy would come back unacknowledged.
func TestRecordDeviations_DoesNotReinsertADeviationAlreadyOnRecord(t *testing.T) {
	repo := &recordingDeviationRepo{
		liveKeys: []string{string(modeprofile.RuleKeyTemperatureRange)},
	}
	policy := policyWith(modeprofile.RuleKeyTemperatureRange, modeprofile.RuleKeyHazmatSegregation)

	err := newTestService(repo).RecordDeviations(
		t.Context(),
		recordRequest(policy, modeprofile.RuleKeyTemperatureRange, modeprofile.RuleKeyHazmatSegregation),
	)
	require.NoError(t, err)

	require.Len(t, repo.recorded, 1)
	assert.Equal(t, modeprofile.RuleKeyHazmatSegregation, repo.recorded[0].RuleKey)
}

func TestRecordDeviations_RecordsNothingWhenEveryRuleIsAlreadyOnRecord(t *testing.T) {
	repo := &recordingDeviationRepo{
		liveKeys: []string{string(modeprofile.RuleKeyTemperatureRange)},
	}
	policy := policyWith(modeprofile.RuleKeyTemperatureRange)

	err := newTestService(repo).RecordDeviations(
		t.Context(),
		recordRequest(policy, modeprofile.RuleKeyTemperatureRange),
	)
	require.NoError(t, err)

	assert.Empty(t, repo.recorded)
	// The supersede still runs: a rule that stopped firing must be resolved even
	// when nothing new is inserted.
	assert.NotNil(t, repo.superseded)
}

func TestRecordDeviations_RecordsAFirstTimeDeviation(t *testing.T) {
	repo := &recordingDeviationRepo{}
	policy := policyWith(modeprofile.RuleKeyTemperatureRange)

	err := newTestService(repo).RecordDeviations(
		t.Context(),
		recordRequest(policy, modeprofile.RuleKeyTemperatureRange),
	)
	require.NoError(t, err)

	require.Len(t, repo.recorded, 1)
	assert.Equal(t, modeprofile.DeviationStateOpen, repo.recorded[0].State)
}

// An advisory whose rule the resolved profile does not carry is not a deviation
// from that profile, so it must not be recorded against it.
func TestRecordDeviations_IgnoresAnAdvisoryTheProfileDoesNotGovern(t *testing.T) {
	repo := &recordingDeviationRepo{}

	err := newTestService(repo).RecordDeviations(
		t.Context(),
		recordRequest(policyWith(modeprofile.RuleKeyTemperatureRange), modeprofile.RuleKeyMoveRemoval),
	)
	require.NoError(t, err)

	assert.Empty(t, repo.recorded)
}

// A resource whose rules all stopped firing is the case that closes deviations
// out. Returning early because there is nothing to insert would leave every
// prior deviation open forever, so the supersede has to run on its own.
func TestRecordDeviations_ResolvesPriorDeviationsWhenNothingFiresAnyMore(t *testing.T) {
	repo := &recordingDeviationRepo{
		liveKeys: []string{string(modeprofile.RuleKeyTemperatureRange)},
	}

	err := newTestService(repo).RecordDeviations(
		t.Context(),
		recordRequest(policyWith(modeprofile.RuleKeyTemperatureRange)),
	)
	require.NoError(t, err)

	assert.Empty(t, repo.recorded)
	require.NotNil(t, repo.superseded, "prior deviations must still be resolved")
	assert.Empty(t, repo.superseded.KeepRuleKeys,
		"nothing fired, so nothing may be kept open")
}
