package workerdrugalcoholservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// seedBytes is the length of a draw seed. Thirty-two bytes is far more entropy
// than a selection needs, and it costs nothing to make the seed impossible to
// guess ahead of the draw — which is what makes the round genuinely random
// rather than merely arbitrary.
const seedBytes = 32

func (s *Service) ListPools(
	ctx context.Context,
	req *repositories.ListDOTRandomPoolsRequest,
) (*pagination.CursorListResult[*worker.DOTRandomPool], error) {
	return s.repo.ListPools(ctx, req)
}

func (s *Service) GetPool(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.DOTRandomPool, error) {
	return s.repo.GetPoolByID(ctx, &repositories.GetDOTRandomPoolByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) preparePool(
	ctx context.Context,
	entity *worker.DOTRandomPool,
	excludeID pulid.ID,
) error {
	entity.Code = strings.ToUpper(strings.TrimSpace(entity.Code))
	entity.Name = strings.TrimSpace(entity.Name)
	entity.Description = strings.TrimSpace(entity.Description)
	if entity.IncludedDriverTypes == nil {
		entity.IncludedDriverTypes = []string{}
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
	exists, err := s.repo.PoolCodeExists(ctx, &repositories.DOTRandomPoolCodeExistsRequest{
		TenantInfo: tenantInfo,
		Code:       entity.Code,
		ExcludeID:  excludeID,
	})
	if err != nil {
		return err
	}
	if exists {
		return errortypes.NewValidationError(
			"code",
			errortypes.ErrDuplicate,
			"A pool with this code already exists",
		)
	}

	return nil
}

func (s *Service) CreatePool(
	ctx context.Context,
	entity *worker.DOTRandomPool,
	userID pulid.ID,
) (*worker.DOTRandomPool, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	if err := s.preparePool(ctx, entity, pulid.Nil); err != nil {
		return nil, err
	}
	if entity.IsDefault {
		if err := s.repo.ClearDefaultPool(ctx, tenantInfo, pulid.Nil); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.CreatePool(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceDOTRandomPool,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment:    "Created random testing pool " + created.Code,
	})
	s.publish(ctx, tenantInfo, realtimePool, permission.OpCreate, created.ID, userID)

	return created, nil
}

func (s *Service) UpdatePool(
	ctx context.Context,
	entity *worker.DOTRandomPool,
	userID pulid.ID,
) (*worker.DOTRandomPool, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	previous, err := s.repo.GetPoolByID(ctx, &repositories.GetDOTRandomPoolByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if err = s.preparePool(ctx, entity, entity.ID); err != nil {
		return nil, err
	}
	if entity.IsDefault {
		if err = s.repo.ClearDefaultPool(ctx, tenantInfo, entity.ID); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.UpdatePool(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceDOTRandomPool,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   previous,
		comment:    "Updated random testing pool " + updated.Code,
	})
	s.publish(ctx, tenantInfo, realtimePool, permission.OpUpdate, updated.ID, userID)

	return updated, nil
}

func (s *Service) ListDraws(
	ctx context.Context,
	req *repositories.ListDOTRandomDrawsRequest,
) ([]*worker.DOTRandomDraw, error) {
	return s.repo.ListDraws(ctx, req)
}

func (s *Service) GetDraw(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.DOTRandomDraw, error) {
	return s.repo.GetDrawByID(ctx, &repositories.GetDOTRandomDrawByIDRequest{
		ID:             id,
		TenantInfo:     tenantInfo,
		IncludeEntries: true,
		IncludeWorkers: true,
		IncludePool:    true,
	})
}

// RunDrawRequest asks for one selection round. The period is derived from the
// pool's own cadence unless one is named, so the ordinary case is a single
// button.
type RunDrawRequest struct {
	TenantInfo pagination.TenantInfo
	PoolID     pulid.ID
	// At is the moment the round is drawn for; zero means now. Naming it lets
	// an office catch up a period they missed without back-dating the clock.
	At     int64
	Notes  string
	UserID pulid.ID
}

// RunDraw selects the round. The names are chosen by a keyed hash of each
// worker id under a seed generated here and stored with the round, so the
// selection is unpredictable before the draw and reproducible after it: an
// auditor with the seed and the roster gets the same names back.
func (s *Service) RunDraw(
	ctx context.Context,
	req *RunDrawRequest,
) (*worker.DOTRandomDraw, error) {
	pool, err := s.resolvePool(ctx, req.TenantInfo, req.PoolID)
	if err != nil {
		return nil, err
	}

	at := req.At
	if at <= 0 {
		at = timeutils.NowUnix()
	}
	drawTime := time.Unix(at, 0).UTC()
	periodKey := worker.PeriodKeyFor(pool.Period, drawTime)
	periodStart, periodEnd := worker.PeriodBoundsFor(pool.Period, drawTime)

	existing, err := s.repo.GetDrawByPeriod(ctx, &repositories.GetDOTRandomDrawByPeriodRequest{
		TenantInfo: req.TenantInfo,
		PoolID:     pool.ID,
		PeriodKey:  periodKey,
	})
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errortypes.NewValidationError(
			"periodKey",
			errortypes.ErrDuplicate,
			"This pool has already been drawn for "+periodKey,
		)
	}

	candidates, err := s.repo.ListPoolCandidates(ctx, &repositories.ListPoolCandidatesRequest{
		TenantInfo:  req.TenantInfo,
		DriverTypes: pool.IncludedDriverTypes,
	})
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, errortypes.NewValidationError(
			"poolId",
			errortypes.ErrInvalid,
			"There is nobody in this pool to draw from",
		)
	}

	seed, err := newSeed()
	if err != nil {
		return nil, err
	}

	drugTarget, alcoholTarget := pool.TargetsFor(len(candidates))
	// The two substances draw independently: being picked for a drug test does
	// not exclude anybody from the alcohol draw, and the rates are separate
	// obligations. Salting the seed per substance keeps the two orders from
	// being the same list twice.
	drugPicks := worker.SelectRandom(candidates, seed+":drug", drugTarget)
	alcoholPicks := worker.SelectRandom(candidates, seed+":alcohol", alcoholTarget)

	draw := &worker.DOTRandomDraw{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		PoolID:          pool.ID,
		PeriodKey:       periodKey,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		Status:          worker.RandomDrawStatusDraft,
		PoolSize:        int32(len(candidates)),
		DrugTarget:      int32(drugTarget),
		AlcoholTarget:   int32(alcoholTarget),
		DrugSelected:    int32(len(drugPicks)),
		AlcoholSelected: int32(len(alcoholPicks)),
		Seed:            seed,
		Method:          worker.RandomSelectionMethod,
		Notes:           strings.TrimSpace(req.Notes),
		DrawnAt:         at,
		DrawnByID:       req.UserID,
	}

	multiErr := errortypes.NewMultiError()
	draw.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	entries := buildEntries(drugPicks, worker.DOTSubstanceDrug)
	entries = append(entries, buildEntries(alcoholPicks, worker.DOTSubstanceAlcohol)...)

	created, err := s.repo.CreateDrawWithEntries(ctx, draw, entries)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceDOTRandomPool,
		resourceID: created.ID.String(),
		operation:  permission.OpManage,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    created,
		comment: "Drew " + periodKey + " from " + pool.Code +
			" over " + strconv.Itoa(len(candidates)) + " drivers",
	})
	s.publish(ctx, req.TenantInfo, realtimeDraw, permission.OpCreate, created.ID, req.UserID)

	return created, nil
}

// FinalizeDraw locks a round. Until it is final the office can re-draw; once it
// is, the names are the evidence, and a mistake is corrected by cancelling the
// round rather than quietly reshuffling it.
func (s *Service) FinalizeDraw(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) (*worker.DOTRandomDraw, error) {
	entity, err := s.repo.GetDrawByID(ctx, &repositories.GetDOTRandomDrawByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if entity.Status != worker.RandomDrawStatusDraft {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only a draft round can be finalised",
		)
	}

	previous := *entity
	finalizedAt := timeutils.NowUnix()
	entity.Status = worker.RandomDrawStatusFinal
	entity.FinalizedAt = &finalizedAt

	updated, err := s.repo.UpdateDraw(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceDOTRandomPool,
		resourceID: updated.ID.String(),
		operation:  permission.OpManage,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Finalised round " + updated.PeriodKey,
	})
	s.publish(ctx, tenantInfo, realtimeDraw, permission.OpUpdate, updated.ID, userID)

	return updated, nil
}

// CancelDraw voids a round so the period can be drawn again. The row stays: a
// cancelled round and the reason for it are part of the evidence too.
func (s *Service) CancelDraw(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	reason string,
	userID pulid.ID,
) (*worker.DOTRandomDraw, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Cancelling a round needs a reason on the record",
		)
	}

	entity, err := s.repo.GetDrawByID(ctx, &repositories.GetDOTRandomDrawByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if entity.Status == worker.RandomDrawStatusCancelled {
		return entity, nil
	}

	previous := *entity
	entity.Status = worker.RandomDrawStatusCancelled
	entity.FinalizedAt = nil
	entity.Notes = strings.TrimSpace(entity.Notes + "\nCancelled: " + reason)

	updated, err := s.repo.UpdateDraw(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceDOTRandomPool,
		resourceID: updated.ID.String(),
		operation:  permission.OpManage,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    reason,
	})
	s.publish(ctx, tenantInfo, realtimeDraw, permission.OpUpdate, updated.ID, userID)

	return updated, nil
}

func (s *Service) ListDrawEntries(
	ctx context.Context,
	req *repositories.ListDOTRandomDrawEntriesRequest,
) ([]*worker.DOTRandomDrawEntry, error) {
	return s.repo.ListDrawEntries(ctx, req)
}

// UpdateEntryRequest moves one selection along: the driver has been told, or
// the selection is being excused or written off as missed.
type UpdateEntryRequest struct {
	TenantInfo   pagination.TenantInfo
	EntryID      pulid.ID
	Status       worker.RandomEntryStatus
	ExcuseReason string
	UserID       pulid.ID
}

func (s *Service) UpdateDrawEntry(
	ctx context.Context,
	req *UpdateEntryRequest,
) (*worker.DOTRandomDrawEntry, error) {
	entity, err := s.repo.GetDrawEntryByID(ctx, &repositories.GetDOTRandomDrawEntryByIDRequest{
		ID:         req.EntryID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	// Completion is not a status somebody sets by hand: it is what recording
	// the collection against the selection means, so leaving it to the test
	// keeps the entry and the test from ever disagreeing.
	if req.Status == worker.RandomEntryCompleted {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A selection completes when its test is recorded, not on its own",
		)
	}

	previous := *entity
	entity.Status = req.Status
	entity.ExcuseReason = strings.TrimSpace(req.ExcuseReason)
	if req.Status == worker.RandomEntryNotified && entity.NotifiedAt == nil {
		notifiedAt := timeutils.NowUnix()
		entity.NotifiedAt = &notifiedAt
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateDrawEntry(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceDOTRandomPool,
		resourceID: updated.ID.String(),
		operation:  permission.OpManage,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Selection is now " + string(updated.Status),
	})
	s.publish(ctx, req.TenantInfo, realtimeDraw, permission.OpUpdate, updated.DrawID, req.UserID)

	return updated, nil
}

func (s *Service) resolvePool(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	poolID pulid.ID,
) (*worker.DOTRandomPool, error) {
	if !poolID.IsNil() {
		return s.repo.GetPoolByID(ctx, &repositories.GetDOTRandomPoolByIDRequest{
			ID:         poolID,
			TenantInfo: tenantInfo,
		})
	}

	pool, err := s.repo.GetDefaultPool(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, errortypes.NewValidationError(
			"poolId",
			errortypes.ErrRequired,
			"Name a pool to draw from, or mark one as the default",
		)
	}

	return pool, nil
}

func buildEntries(
	picks []pulid.ID,
	substance worker.DOTTestSubstance,
) []*worker.DOTRandomDrawEntry {
	entries := make([]*worker.DOTRandomDrawEntry, 0, len(picks))
	for i, workerID := range picks {
		entries = append(entries, &worker.DOTRandomDrawEntry{
			WorkerID:  workerID,
			Substance: substance,
			Rank:      int32(i + 1),
			Status:    worker.RandomEntrySelected,
		})
	}
	return entries
}

func newSeed() (string, error) {
	buf := make([]byte, seedBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
