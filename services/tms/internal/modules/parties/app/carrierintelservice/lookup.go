package carrierintelservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierinteldiff"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelrule"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelusage"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type FetchRequest struct {
	TenantInfo    pagination.TenantInfo
	Subject       carrierintelports.CarrierIntelSubject
	MinDepth      carrierintel.LookupDepth
	MaxAgeSeconds int64
	Purpose       carrierintel.Purpose
	Force         bool
	Source        carrierintelevent.EventSource
	Control       *carrierintelcontrol.CarrierIntelControl
}

type FetchResult struct {
	Snapshot     *carrierintelsnapshot.CarrierIntelSnapshot
	FromCache    bool
	UsedFallback bool
	Changes      []carrierinteldiff.FieldChange
	Raised       []carrierintelrule.Finding
}

func depthEndpoint(depth carrierintel.LookupDepth) carrierintelusage.Endpoint {
	switch depth {
	case carrierintel.LookupDepthFull:
		return carrierintelusage.EndpointProfileFull
	case carrierintel.LookupDepthLite:
		return carrierintelusage.EndpointProfileLite
	default:
		return carrierintelusage.EndpointProfileFMCSA
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
		carrierintelports.CarrierIntelSubjectRef{
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
			maxAge = control.TTLSecondsForDepth(wanted)
		}
		if current.IsFreshForDepth(s.now(), maxAge, wanted) {
			return &FetchResult{Snapshot: current, FromCache: true}, nil
		}
	}

	if err = s.requirePaidLookups(ctx, req.TenantInfo); err != nil {
		return nil, err
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
		source = carrierintelevent.EventSourceSnapshotDiff
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
	control *carrierintelcontrol.CarrierIntelControl
	primary *boundProvider
	subject carrierintelports.CarrierIntelSubject
	wanted  carrierintel.LookupDepth
}

func (s *Service) lookupWithFallback(
	ctx context.Context,
	attempt *lookupAttempt,
) (*carrierintelports.CarrierIntelLookupResult, *boundProvider, error) {
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
	).WithInternal(&carrierintelports.CarrierIntelProviderError{
		Provider: provider,
		Kind:     carrierintelports.CarrierIntelErrorUnavailable,
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

	price := bound.prices.Price(carrierintelusage.EndpointProfileFull)
	dedupe := carrierintelusage.BillingDedupeKey(
		carrierintelusage.EndpointProfileFull, price, attempt.subject.DOTNumber, s.now(),
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
) (*carrierintelports.CarrierIntelLookupResult, error) {
	depth := s.chooseDepth(ctx, attempt, bound)
	if depth == "" {
		return nil, &carrierintelports.CarrierIntelProviderError{
			Provider: bound.provider,
			Kind:     carrierintelports.CarrierIntelErrorUnsupported,
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

	request := &carrierintelports.CarrierIntelLookupRequest{
		Identifier: carrierintelports.CarrierIntelIdentifier{
			DOTNumber:    attempt.subject.DOTNumber,
			DocketNumber: attempt.subject.DocketNumber,
		},
		Depth: depth,
	}
	if request.Identifier.DOTNumber != "" {
		request.Identifier.DocketNumber = ""
	}

	var result *carrierintelports.CarrierIntelLookupResult
	err := s.withRateLimitWait(ctx, func() error {
		var lookupErr error
		result, lookupErr = bound.client.Lookup(ctx, request)
		return lookupErr
	})
	if err != nil {
		if carrierintelports.IsCarrierIntelErrorKind(
			err,
			carrierintelports.CarrierIntelErrorNotFound,
		) {
			return &carrierintelports.CarrierIntelLookupResult{
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
