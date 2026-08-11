package rateconfirmationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type testDBConnection struct{}

func (testDBConnection) DB() *bun.DB                          { return nil }
func (testDBConnection) DBForContext(context.Context) bun.IDB { return nil }
func (testDBConnection) HealthCheck(context.Context) error    { return nil }
func (testDBConnection) IsHealthy(context.Context) bool       { return true }
func (testDBConnection) Close() error                         { return nil }
func (testDBConnection) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type fakeRateConRepo struct {
	repositories.RateConfirmationRepository

	byID               map[pulid.ID]*rateconfirmation.RateConfirmation
	activeByAssignment map[pulid.ID]*rateconfirmation.RateConfirmation
	tokensByHash       map[string]*rateconfirmation.RateConfirmationToken

	updated       []*rateconfirmation.RateConfirmation
	updateErr     error
	createdTokens []*rateconfirmation.RateConfirmationToken
	markedUsed    []pulid.ID
	revokedFor    []pulid.ID
}

func newFakeRateConRepo() *fakeRateConRepo {
	return &fakeRateConRepo{
		byID:               make(map[pulid.ID]*rateconfirmation.RateConfirmation),
		activeByAssignment: make(map[pulid.ID]*rateconfirmation.RateConfirmation),
		tokensByHash:       make(map[string]*rateconfirmation.RateConfirmationToken),
	}
}

func (f *fakeRateConRepo) add(entity *rateconfirmation.RateConfirmation) {
	f.byID[entity.ID] = entity
	if entity.Status.IsActive() {
		f.activeByAssignment[entity.CarrierAssignmentID] = entity
	}
}

func (f *fakeRateConRepo) GetByID(
	_ context.Context,
	req *repositories.GetRateConfirmationByIDRequest,
) (*rateconfirmation.RateConfirmation, error) {
	entity, ok := f.byID[req.RateConfirmationID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Rate confirmation not found")
	}
	return entity, nil
}

func (f *fakeRateConRepo) GetActiveByAssignmentID(
	_ context.Context,
	_ pagination.TenantInfo,
	assignmentID pulid.ID,
) (*rateconfirmation.RateConfirmation, error) {
	entity, ok := f.activeByAssignment[assignmentID]
	if !ok {
		return nil, nil //nolint:nilnil // mirrors the production contract
	}
	return entity, nil
}

func (f *fakeRateConRepo) Update(
	_ context.Context,
	entity *rateconfirmation.RateConfirmation,
) (*rateconfirmation.RateConfirmation, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	f.updated = append(f.updated, entity)
	f.byID[entity.ID] = entity
	if entity.Status.IsActive() {
		f.activeByAssignment[entity.CarrierAssignmentID] = entity
	} else {
		delete(f.activeByAssignment, entity.CarrierAssignmentID)
	}
	return entity, nil
}

func (f *fakeRateConRepo) CreateToken(
	_ context.Context,
	entity *rateconfirmation.RateConfirmationToken,
) (*rateconfirmation.RateConfirmationToken, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("rct_")
	}
	f.createdTokens = append(f.createdTokens, entity)
	f.tokensByHash[entity.TokenHash] = entity
	return entity, nil
}

func (f *fakeRateConRepo) GetTokenByHash(
	_ context.Context,
	tokenHash string,
) (*rateconfirmation.RateConfirmationToken, error) {
	token, ok := f.tokensByHash[tokenHash]
	if !ok {
		return nil, nil //nolint:nilnil // mirrors the production contract
	}
	return token, nil
}

func (f *fakeRateConRepo) MarkTokenUsed(
	_ context.Context,
	req *repositories.MarkRateConfirmationTokenUsedRequest,
) (bool, error) {
	f.markedUsed = append(f.markedUsed, req.TokenID)
	return true, nil
}

func (f *fakeRateConRepo) RevokeTokensForRateConfirmation(
	_ context.Context,
	_ pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	_ int64,
) error {
	f.revokedFor = append(f.revokedFor, rateConfirmationID)
	return nil
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
}
