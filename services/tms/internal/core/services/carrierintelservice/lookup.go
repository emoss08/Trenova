package carrierintelservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type FetchRequest struct {
	TenantInfo    pagination.TenantInfo
	Subject       repositories.CarrierIntelSubject
	MinDepth      carrierintel.LookupDepth
	MaxAgeSeconds int64
	Purpose       carrierintel.Purpose
	Force         bool
	Source        carrierintel.EventSource
	Control       *carrierintel.CarrierIntelControl
}

type FetchResult struct {
	Snapshot     *carrierintel.CarrierIntelSnapshot
	FromCache    bool
	UsedFallback bool
	Changes      []carrierintel.FieldChange
	Raised       []carrierintel.Finding
}

func depthEndpoint(depth carrierintel.LookupDepth) carrierintel.Endpoint {
	switch depth {
	case carrierintel.LookupDepthFull:
		return carrierintel.EndpointProfileFull
	case carrierintel.LookupDepthLite:
		return carrierintel.EndpointProfileLite
	default:
		return carrierintel.EndpointProfileFMCSA
	}
}

func (s *Service) Fetch(ctx context.Context, req *FetchRequest) (*FetchResult, error) {
	if req.Subject.DOTNumber == "" && req.Subject.DocketNumber == "" {
		return nil, errortypes.NewValidationError("dotNumber", errortypes.ErrRequired,
			"A DOT or MC number is required to look up carrier intelligence")
	}

	purpose := req.Purpose
	if !purpose.IsValid() {
		purpose = carrierintel.PurposeRefresh
	}
	ctx = carrierintel.WithPurpose(ctx, purpose)

	control := req.Control
	if control == nil {
		var err error
		control, err = s.controlRepo.GetOrCreate(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
	}

	current, err := s.snapshotRepo.GetCurrent(
		ctx,
		req.TenantInfo,
		repositories.CarrierIntelSubjectRef{
			SubjectType: req.Subject.SubjectType,
			SubjectID:   req.Subject.SubjectID,
		},
	)
	if err != nil {
		return nil, err
	}

	wanted := req.MinDepth
	if !wanted.IsValid() {
		wanted = purpose.DefaultDepth()
	}

	if !req.Force && current != nil && current.Depth.Satisfies(wanted) {
		maxAge := req.MaxAgeSeconds
		if maxAge <= 0 {
			maxAge = control.SnapshotTTLSeconds()
			if wanted == carrierintel.LookupDepthFull {
				maxAge = control.FullProfileTTLSeconds()
			}
		}
		if current.IsFresh(s.now(), maxAge) {
			return &FetchResult{Snapshot: current, FromCache: true}, nil
		}
	}

	bound, err := s.resolvePrimary(ctx, req.TenantInfo, control)
	if err != nil {
		return nil, err
	}

	result, used, err := s.lookupWithFallback(ctx, &lookupAttempt{
		tenant:  req.TenantInfo,
		control: control,
		primary: bound,
		subject: req.Subject,
		wanted:  wanted,
	})
	if err != nil {
		return nil, err
	}

	source := req.Source
	if !source.IsValid() {
		source = carrierintel.EventSourceSnapshotDiff
	}

	ingested, err := s.ingest(ctx, &ingestInput{
		tenant:  req.TenantInfo,
		control: control,
		subject: req.Subject,
		bound:   used,
		result:  result,
		current: current,
		source:  source,
		purpose: purpose,
	})
	if err != nil {
		return nil, err
	}
	ingested.UsedFallback = used.fallback
	return ingested, nil
}

type lookupAttempt struct {
	tenant  pagination.TenantInfo
	control *carrierintel.CarrierIntelControl
	primary *boundProvider
	subject repositories.CarrierIntelSubject
	wanted  carrierintel.LookupDepth
}

func (s *Service) lookupWithFallback(
	ctx context.Context,
	attempt *lookupAttempt,
) (*services.CarrierIntelLookupResult, *boundProvider, error) {
	if s.providerAvailable(attempt.primary) {
		result, err := s.lookup(ctx, attempt, attempt.primary)
		if err == nil {
			return result, attempt.primary, nil
		}
		if !isOutageError(err) || isSpendCapError(err) {
			return nil, attempt.primary, s.lookupError(attempt.primary.provider, err)
		}
		s.l.Warn("primary carrier intelligence provider failed; trying fallback",
			zap.String("provider", attempt.primary.provider.String()), zap.Error(err))
	}

	fallback, ok := s.resolveFallback(ctx, attempt.tenant, attempt.control)
	if !ok || !s.providerAvailable(fallback) {
		return nil, attempt.primary, providerUnavailableError(attempt.primary.provider)
	}

	result, err := s.lookup(ctx, attempt, fallback)
	if err != nil {
		return nil, fallback, s.lookupError(fallback.provider, err)
	}
	return result, fallback, nil
}

func (s *Service) lookupError(provider integration.Type, err error) error {
	if isSpendCapError(err) {
		var business *errortypes.BusinessError
		if errors.As(err, &business) {
			return business
		}
	}
	return toBusinessError(provider, err)
}

func providerUnavailableError(provider integration.Type) error {
	return errortypes.NewBusinessError(
		"{0} is currently unavailable and no fallback provider could answer. Try again shortly",
		provider.String(),
	).WithInternal(&services.CarrierIntelProviderError{
		Provider: provider,
		Kind:     services.CarrierIntelErrorUnavailable,
	})
}

func (s *Service) chooseDepth(
	ctx context.Context,
	attempt *lookupAttempt,
	bound *boundProvider,
) carrierintel.LookupDepth {
	caps := bound.capabilities()
	depth, ok := caps.BestDepth(attempt.wanted)
	if !ok {
		return ""
	}
	if depth == carrierintel.LookupDepthFull || !caps.Has(carrierintel.CapabilityLookupFull) ||
		attempt.subject.DOTNumber == "" {
		return depth
	}

	price := bound.prices.Price(carrierintel.EndpointProfileFull)
	dedupe := carrierintel.BillingDedupeKey(
		carrierintel.EndpointProfileFull, price, attempt.subject.DOTNumber, s.now(),
	)
	if dedupe == "" {
		return depth
	}
	paid, err := s.usageRepo.ExistsDedupeKey(ctx, attempt.tenant, bound.provider, dedupe)
	if err != nil {
		s.l.Warn("failed to check existing full profile charge", zap.Error(err))
		return depth
	}
	if paid {
		return carrierintel.LookupDepthFull
	}
	return depth
}

func (s *Service) lookup(
	ctx context.Context,
	attempt *lookupAttempt,
	bound *boundProvider,
) (*services.CarrierIntelLookupResult, error) {
	depth := s.chooseDepth(ctx, attempt, bound)
	if depth == "" {
		return nil, &services.CarrierIntelProviderError{
			Provider: bound.provider,
			Kind:     services.CarrierIntelErrorUnsupported,
			Message:  "provider has no lookup capability",
		}
	}

	check := &budgetCheck{
		tenant:    attempt.tenant,
		control:   attempt.control,
		bound:     bound,
		endpoint:  depthEndpoint(depth),
		dotNumber: attempt.subject.DOTNumber,
	}
	if err := s.guardBudget(ctx, check); err != nil {
		logBudgetDenied(s.l, check, err)
		return nil, err
	}

	request := &services.CarrierIntelLookupRequest{
		Identifier: services.CarrierIntelIdentifier{
			DOTNumber:    attempt.subject.DOTNumber,
			DocketNumber: attempt.subject.DocketNumber,
		},
		Depth: depth,
	}
	if request.Identifier.DOTNumber != "" {
		request.Identifier.DocketNumber = ""
	}

	var result *services.CarrierIntelLookupResult
	err := s.withRateLimitWait(ctx, func() error {
		var lookupErr error
		result, lookupErr = bound.client.Lookup(ctx, request)
		return lookupErr
	})
	if err != nil {
		if services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorNotFound) {
			return &services.CarrierIntelLookupResult{
				NotFound: true,
				Depth:    depth,
				Endpoint: depthEndpoint(depth),
			}, nil
		}
		return nil, err
	}
	if result.Depth == "" {
		result.Depth = depth
	}
	return result, nil
}
