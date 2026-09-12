package iftaservice

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type GenerateReturnRequest struct {
	TenantInfo pagination.TenantInfo
	Period     ifta.Period
	UserID     pulid.ID
}

type ReturnActionRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	UserID     pulid.ID
}

type ReopenReturnRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	UserID     pulid.ID
	Reason     string
}

type MarkFiledRequest struct {
	TenantInfo      pagination.TenantInfo
	ID              pulid.ID
	Version         int64
	UserID          pulid.ID
	FiledAt         int64
	FilingReference string
}

type AmendReturnRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Reason     string
	UserID     pulid.ID
}

type PeriodInfo struct {
	Period   ifta.Period
	Key      string
	Start    int64
	End      int64
	DueDate  int64
	Timezone string
}

func returnTenant(ret *ifta.Return) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: ret.OrganizationID, BuID: ret.BusinessUnitID}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListReturnsRequest,
) (*pagination.CursorListResult[*ifta.Return], error) {
	return s.repo.ListReturns(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*ifta.Return, error) {
	return s.repo.GetReturnByID(ctx, &repositories.GetReturnByIDRequest{
		ID:                   id,
		TenantInfo:           tenantInfo,
		IncludeLines:         true,
		IncludeJurisdictions: true,
	})
}

func (s *Service) GetReturnsByIDs(
	ctx context.Context,
	req *repositories.GetReturnsByIDsRequest,
) ([]*ifta.Return, error) {
	return s.repo.GetReturnsByIDs(ctx, req)
}

func (s *Service) GetForPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	period ifta.Period,
) (*ifta.Return, error) {
	if err := period.Validate(); err != nil {
		return nil, errortypes.NewValidationError("quarter", errortypes.ErrInvalid, err.Error())
	}
	latest, err := s.repo.GetLatestReturnForPeriod(ctx, &repositories.GetLatestReturnForPeriodRequest{
		TenantInfo: tenantInfo,
		Year:       period.Year,
		Quarter:    period.Quarter,
	})
	if err != nil || latest == nil {
		return nil, err
	}
	return s.Get(ctx, tenantInfo, latest.ID)
}

func (s *Service) CurrentPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (ifta.Period, error) {
	loc := s.tenantLocation(ctx, tenantInfo.OrgID)
	return ifta.PeriodOf(s.now(), loc).Previous(), nil
}

func (s *Service) PeriodInfo(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	year, quarter int,
) (PeriodInfo, error) {
	period := ifta.NewPeriod(year, quarter)
	if err := period.Validate(); err != nil {
		return PeriodInfo{}, errortypes.NewValidationError(
			"quarter",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}
	loc := s.tenantLocation(ctx, tenantInfo.OrgID)
	start, end := period.Bounds(loc)
	return PeriodInfo{
		Period:   period,
		Key:      period.Key(),
		Start:    start,
		End:      end,
		DueDate:  period.DueDate(loc),
		Timezone: loc.String(),
	}, nil
}

func (s *Service) loadReturn(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	version int64,
) (*ifta.Return, error) {
	ret, err := s.repo.GetReturnByID(ctx, &repositories.GetReturnByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if ret.Version != version {
		return nil, versionMismatch("return")
	}
	return ret, nil
}

func (s *Service) returnLocation(ctx context.Context, ret *ifta.Return) *time.Location {
	if ret.Timezone != "" {
		if loc, err := time.LoadLocation(ret.Timezone); err == nil {
			return loc
		}
	}
	return s.tenantLocation(ctx, ret.OrganizationID)
}

func (s *Service) tractorProfiles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]TractorProfile, error) {
	out := make(map[pulid.ID]TractorProfile, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	tractors, err := s.tractorRepo.GetByIDs(ctx, repositories.GetTractorsByIDsRequest{
		TenantInfo: tenantInfo,
		TractorIDs: ids,
	})
	if err != nil {
		return nil, err
	}
	for _, t := range tractors {
		out[t.ID] = TractorProfile{FuelType: t.FuelType, IFTAQualified: t.IFTAQualified}
	}
	return out, nil
}

type idSet struct {
	seen map[pulid.ID]struct{}
	ids  []pulid.ID
}

func newIDSet(capacity int) *idSet {
	return &idSet{seen: make(map[pulid.ID]struct{}, capacity), ids: make([]pulid.ID, 0, capacity)}
}

func (s *idSet) add(id pulid.ID) {
	if id.IsNil() {
		return
	}
	if _, ok := s.seen[id]; ok {
		return
	}
	s.seen[id] = struct{}{}
	s.ids = append(s.ids, id)
}

func (s *Service) recompute(ctx context.Context, ret *ifta.Return) (*ifta.Return, error) {
	tenantInfo := returnTenant(ret)
	loc := s.returnLocation(ctx, ret)

	miles, err := s.repo.AccumulateMiles(ctx, &repositories.AccumulateMilesRequest{
		TenantInfo: tenantInfo,
		Start:      ret.PeriodStart,
		End:        ret.PeriodEnd,
		Year:       ret.Year,
		Quarter:    ret.Quarter,
	})
	if err != nil {
		return nil, err
	}

	fuel, err := s.fuel.AccumulateFuel(ctx, &repositories.AccumulateFuelRequest{
		TenantInfo: tenantInfo,
		Start:      ret.PeriodStart,
		End:        ret.PeriodEnd,
	})
	if err != nil {
		return nil, err
	}

	rates, err := s.repo.ResolveRates(ctx, &repositories.ResolveRatesRequest{
		Year:    ret.Year,
		Quarter: ret.Quarter,
	})
	if err != nil {
		return nil, err
	}

	tractorIDs := newIDSet(len(miles.RouteRows) + len(fuel))
	jurisdictionIDs := newIDSet(len(miles.RouteRows) + len(miles.ManualRows) + len(fuel))
	for _, row := range miles.RouteRows {
		tractorIDs.add(row.TractorID)
		jurisdictionIDs.add(row.JurisdictionID)
	}
	for _, row := range miles.ManualRows {
		tractorIDs.add(row.TractorID)
		jurisdictionIDs.add(row.JurisdictionID)
	}
	for _, row := range fuel {
		tractorIDs.add(row.TractorID)
		jurisdictionIDs.add(row.JurisdictionID)
	}

	tractors, err := s.tractorProfiles(ctx, tenantInfo, tractorIDs.ids)
	if err != nil {
		return nil, err
	}
	jurisdictions, err := s.jurisdictionsByID(ctx, jurisdictionIDs.ids)
	if err != nil {
		return nil, err
	}

	result := Compute(ComputeInput{
		Period:        ret.Period(),
		Loc:           loc,
		Jurisdictions: jurisdictions,
		Tractors:      tractors,
		Miles:         miles,
		Fuel:          fuel,
		Rates:         rates,
	})
	applyComputation(ret, &result)

	return s.repo.ReplaceReturnLines(ctx, ret, result.Lines)
}

func applyComputation(ret *ifta.Return, result *ComputeResult) {
	ret.TotalMiles = result.Totals.TotalMiles
	ret.TotalTaxableMiles = result.Totals.TotalTaxableMiles
	ret.TotalGallons = result.Totals.TotalGallons
	ret.TotalTaxPaidGallons = result.Totals.TotalTaxPaidGallons
	ret.NetTaxableGallons = result.Totals.NetTaxableGallons
	ret.TaxDueMinor = result.Totals.TaxDueMinor
	ret.SurchargeDueMinor = result.Totals.SurchargeDueMinor
	ret.NetDueMinor = result.Totals.NetDueMinor
	ret.FleetMPGByFuelType = result.FleetMPG
	ret.UnattributedMiles = result.Unattributed.Miles.Round(milesScale)
	ret.UnattributedMoveCount = result.Unattributed.MoveCount
	ret.NoTractorMiles = result.NoTractor.Miles.Round(milesScale)
	ret.NoTractorMoveCount = result.NoTractor.MoveCount
	ret.Problems = result.Problems
	if ret.CurrencyCode == "" {
		ret.CurrencyCode = money.DefaultCurrencyCode
	}
}

func (s *Service) newDraft(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	period ifta.Period,
) (*ifta.Return, error) {
	if err := period.Validate(); err != nil {
		return nil, errortypes.NewValidationError("quarter", errortypes.ErrInvalid, err.Error())
	}

	open, err := s.repo.GetOpenReturnForPeriod(ctx, &repositories.GetOpenReturnForPeriodRequest{
		TenantInfo: tenantInfo,
		Year:       period.Year,
		Quarter:    period.Quarter,
	})
	if err != nil {
		return nil, err
	}
	if open != nil {
		return nil, errortypes.NewConflictError(
			"A {0} return already exists for {1}; recompute or reopen it instead of generating another", strings.ToLower(open.Status.Label()), period.Label(),
		)
	}

	loc := s.tenantLocation(ctx, tenantInfo.OrgID)
	start, end := period.Bounds(loc)
	return &ifta.Return{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Year:           period.Year,
		Quarter:        period.Quarter,
		Status:         ifta.ReturnStatusDraft,
		Timezone:       loc.String(),
		PeriodStart:    start,
		PeriodEnd:      end,
		CurrencyCode:   money.DefaultCurrencyCode,
	}, nil
}

func (s *Service) createAndCompute(
	ctx context.Context,
	draft *ifta.Return,
	userID pulid.ID,
	comment string,
) (*ifta.Return, error) {
	draft.Normalize()
	multiErr := errortypes.NewMultiError()
	draft.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateReturn(ctx, draft)
	if err != nil {
		return nil, err
	}

	computed, err := s.recompute(ctx, created)
	if err != nil {
		s.l.Error("failed to compute a newly generated ifta return",
			zap.String("returnId", created.ID.String()),
			zap.Error(err))
		return nil, err
	}

	tenantInfo := returnTenant(computed)
	s.audit(&auditParams{
		resource:   permission.ResourceIFTAReturn,
		resourceID: computed.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    computed,
		comment:    comment,
	})
	s.publish(ctx, tenantInfo, realtimeReturn, permission.OpCreate, computed.ID, userID)

	return computed, nil
}

func (s *Service) Generate(
	ctx context.Context,
	req *GenerateReturnRequest,
) (*ifta.Return, error) {
	latest, err := s.repo.GetLatestReturnForPeriod(ctx, &repositories.GetLatestReturnForPeriodRequest{
		TenantInfo: req.TenantInfo,
		Year:       req.Period.Year,
		Quarter:    req.Period.Quarter,
	})
	if err != nil {
		return nil, err
	}
	if latest != nil && latest.Status == ifta.ReturnStatusFiled {
		return nil, errortypes.NewConflictError(
			"The {0} return has been filed; amend it instead of generating a new one", req.Period.Label(),
		)
	}

	draft, err := s.newDraft(ctx, req.TenantInfo, req.Period)
	if err != nil {
		return nil, err
	}

	return s.createAndCompute(ctx, draft, req.UserID, "Generated the "+req.Period.Label()+" IFTA return")
}

func (s *Service) Recompute(ctx context.Context, req *ReturnActionRequest) (*ifta.Return, error) {
	ret, err := s.loadReturn(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if !ret.CanRecompute() {
		return nil, invalidOperation(
			"status",
			"Only a draft return can be recomputed; reopen or amend it first",
		)
	}

	previous := *ret
	computed, err := s.recompute(ctx, ret)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceIFTAReturn,
		resourceID: computed.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    computed,
		previous:   &previous,
		comment:    "Recomputed the " + computed.Period().Label() + " IFTA return",
	})
	s.publish(ctx, req.TenantInfo, realtimeReturn, permission.OpUpdate, computed.ID, req.UserID)

	return computed, nil
}

func (s *Service) Finalize(ctx context.Context, req *ReturnActionRequest) (*ifta.Return, error) {
	ret, err := s.loadReturn(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if !ret.CanRecompute() {
		return nil, invalidOperation("status", "Only a draft return can be finalized")
	}

	previous := *ret
	computed, err := s.recompute(ctx, ret)
	if err != nil {
		return nil, err
	}
	if !computed.CanFinalize() {
		blocking := computed.BlockingProblems()
		messages := make([]string, 0, len(blocking))
		for i := range blocking {
			messages = append(messages, blocking[i].Message)
		}
		return nil, invalidOperation(
			"problems",
			"The return cannot be finalized until every rate is on file: "+
				strings.Join(messages, "; "),
		)
	}

	now := s.now()
	computed.Status = ifta.ReturnStatusFinalized
	computed.FinalizedAt = &now
	computed.FinalizedByID = req.UserID
	computed.Normalize()
	multiErr := errortypes.NewMultiError()
	computed.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	finalized, err := s.repo.UpdateReturn(ctx, computed)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceIFTAReturn,
		resourceID: finalized.ID.String(),
		operation:  permission.OpApprove,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    finalized,
		previous:   &previous,
		comment:    "Finalized the " + finalized.Period().Label() + " IFTA return",
	})
	s.publish(ctx, req.TenantInfo, realtimeReturn, permission.OpApprove, finalized.ID, req.UserID)

	return finalized, nil
}

func requireReason(reason string) (string, error) {
	trimmed := strings.TrimSpace(reason)
	if len(trimmed) < ifta.MinReasonLength {
		return "", errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A reason of at least 10 characters is required",
		)
	}
	return trimmed, nil
}

func (s *Service) Reopen(ctx context.Context, req *ReopenReturnRequest) (*ifta.Return, error) {
	reason, err := requireReason(req.Reason)
	if err != nil {
		return nil, err
	}

	ret, err := s.loadReturn(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if !ret.CanReopen() {
		return nil, invalidOperation(
			"status",
			"Only a finalized return can be reopened; a filed return must be amended",
		)
	}

	previous := *ret
	now := s.now()
	ret.Status = ifta.ReturnStatusDraft
	ret.FinalizedAt = nil
	ret.FinalizedByID = pulid.Nil
	ret.ReopenedAt = &now
	ret.ReopenedByID = req.UserID
	ret.ReopenReason = reason
	ret.Normalize()
	multiErr := errortypes.NewMultiError()
	ret.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	reopened, err := s.repo.UpdateReturn(ctx, ret)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceIFTAReturn,
		resourceID: reopened.ID.String(),
		operation:  permission.OpReopen,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    reopened,
		previous:   &previous,
		comment:    "Reopened the " + reopened.Period().Label() + " IFTA return: " + reason,
	})
	s.publish(ctx, req.TenantInfo, realtimeReturn, permission.OpReopen, reopened.ID, req.UserID)

	return reopened, nil
}

func (s *Service) MarkFiled(ctx context.Context, req *MarkFiledRequest) (*ifta.Return, error) {
	ret, err := s.loadReturn(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if !ret.CanMarkFiled() {
		return nil, invalidOperation("status", "Only a finalized return can be marked as filed")
	}

	now := s.now()
	switch {
	case req.FiledAt <= 0:
		return nil, errortypes.NewValidationError(
			"filedAt",
			errortypes.ErrRequired,
			"The filing date is required",
		)
	case ret.FinalizedAt != nil && req.FiledAt < *ret.FinalizedAt:
		return nil, errortypes.NewValidationError(
			"filedAt",
			errortypes.ErrInvalid,
			"A return cannot be filed before it was finalized",
		)
	case req.FiledAt > now:
		return nil, errortypes.NewValidationError(
			"filedAt",
			errortypes.ErrInvalid,
			"The filing date cannot be in the future",
		)
	}

	previous := *ret
	ret.Status = ifta.ReturnStatusFiled
	ret.FiledAt = &req.FiledAt
	ret.FiledByID = req.UserID
	ret.FilingReference = strings.TrimSpace(req.FilingReference)
	ret.Normalize()
	multiErr := errortypes.NewMultiError()
	ret.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	filed, err := s.repo.UpdateReturn(ctx, ret)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceIFTAReturn,
		resourceID: filed.ID.String(),
		operation:  permission.OpSubmit,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    filed,
		previous:   &previous,
		comment:    "Marked the " + filed.Period().Label() + " IFTA return as filed",
	})
	s.publish(ctx, req.TenantInfo, realtimeReturn, permission.OpSubmit, filed.ID, req.UserID)

	return filed, nil
}

func (s *Service) Amend(ctx context.Context, req *AmendReturnRequest) (*ifta.Return, error) {
	reason, err := requireReason(req.Reason)
	if err != nil {
		return nil, err
	}

	filed, err := s.repo.GetReturnByID(ctx, &repositories.GetReturnByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !filed.CanAmend() {
		return nil, invalidOperation("status", "Only a filed return can be amended")
	}

	draft, err := s.newDraft(ctx, req.TenantInfo, filed.Period())
	if err != nil {
		return nil, err
	}
	amendsID := filed.ID
	draft.AmendmentNumber = filed.AmendmentNumber + 1
	draft.AmendsReturnID = &amendsID
	draft.Timezone = filed.Timezone
	draft.PeriodStart = filed.PeriodStart
	draft.PeriodEnd = filed.PeriodEnd
	draft.CurrencyCode = filed.CurrencyCode

	return s.createAndCompute(
		ctx,
		draft,
		req.UserID,
		"Opened amendment "+strconv.Itoa(draft.AmendmentNumber)+" of the "+
			filed.Period().Label()+" IFTA return: "+reason,
	)
}

func (s *Service) Delete(ctx context.Context, req *ReturnActionRequest) error {
	ret, err := s.loadReturn(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return err
	}
	if !ret.CanDelete() {
		return invalidOperation("status", "Only a draft return can be deleted")
	}

	if err = s.repo.DeleteReturn(ctx, &repositories.DeleteReturnRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		Version:    req.Version,
	}); err != nil {
		return err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceIFTAReturn,
		resourceID: ret.ID.String(),
		operation:  permission.OpDelete,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    ret,
		comment:    "Deleted the draft " + ret.Period().Label() + " IFTA return",
	})
	s.publish(ctx, req.TenantInfo, realtimeReturn, permission.OpDelete, ret.ID, req.UserID)

	return nil
}
