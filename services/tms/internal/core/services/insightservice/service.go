// Package insightservice runs the refresh that produces home-screen insights
// and serves what it produced.
//
// The order is fixed and the isolation is deliberate: detectors measure, the
// narrator writes, storage replaces. A detector that throws does not stop the
// others; a narrator that is unavailable does not stop anything at all. The
// worst outcome of any failure here is fewer insights or plainer wording, never
// a wrong number.
package insightservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/emoss08/trenova/internal/core/services/insightservice/narrator"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// DefaultWindow is the period a refresh examines. A month is long enough for
	// a service trend to be real rather than weather, and short enough that a
	// problem fixed three weeks ago has left the numbers.
	DefaultWindow = 30 * 24 * time.Hour
	// StaleAfter is how long an insight's numbers are worth trusting. It is
	// longer than the refresh interval on purpose: a missed run should make a
	// card say it is old, not make it vanish.
	StaleAfter = 36 * time.Hour
	// widgetLimit is what a home widget gets if it does not say.
	widgetLimit = 5
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Repo        repositories.InsightRepository
	Detectors   *detector.Registry
	Narrator    *narrator.Service
	Permissions services.PermissionEngine
}

// permissionChecker is the slice of PermissionEngine this service uses. It asks
// one question about one reader and needs nothing else.
type permissionChecker interface {
	Check(
		ctx context.Context,
		req *services.PermissionCheckRequest,
	) (*services.PermissionCheckResult, error)
}

type Service struct {
	l           *zap.Logger
	repo        repositories.InsightRepository
	detectors   *detector.Registry
	narrator    *narrator.Service
	permissions permissionChecker
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.insight"),
		repo:        p.Repo,
		detectors:   p.Detectors,
		narrator:    p.Narrator,
		permissions: p.Permissions,
	}
}

// RefreshRequest runs every detector for one tenant.
type RefreshRequest struct {
	TenantInfo pagination.TenantInfo
	Timezone   string
	// Now is the instant the run is anchored to. It is passed rather than read
	// from the clock so a scheduled run and a test agree on what "the last 30
	// days" means.
	Now int64
}

// RefreshResult reports what the run did, per detector, so a scheduled job logs
// work done rather than merely "ran".
type RefreshResult struct {
	Created    int
	Superseded int
	Resolved   int
	Suppressed int
	// Failed names the detectors that errored. The run still succeeds: partial
	// insight is worth more than none, and a broken detector must not take the
	// working ones off someone's home screen.
	Failed []string
	// Narrated counts how many findings a model actually wrote prose for, which
	// is the number that says whether narration is working in this deployment.
	Narrated int
}

// Refresh recomputes every detector's findings for one tenant.
func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (RefreshResult, error) {
	now := req.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}

	params := detector.Params{
		TenantInfo:  req.TenantInfo,
		WindowStart: now - int64(DefaultWindow.Seconds()),
		WindowEnd:   now,
		Timezone:    req.Timezone,
	}

	var result RefreshResult
	for _, d := range s.detectors.All() {
		s.refreshDetector(ctx, refreshParams{
			detector: d,
			params:   params,
			now:      now,
			result:   &result,
		})
	}

	s.l.Info("insight refresh complete",
		zap.String("organization", req.TenantInfo.OrgID.String()),
		zap.Int("created", result.Created),
		zap.Int("resolved", result.Resolved),
		zap.Int("suppressed", result.Suppressed),
		zap.Int("narrated", result.Narrated),
		zap.Strings("failed", result.Failed),
	)

	return result, nil
}

type refreshParams struct {
	detector detector.Detector
	params   detector.Params
	now      int64
	result   *RefreshResult
}

// refreshDetector runs one detector and stores what it found.
//
// Every failure mode here is contained to this detector. A query that errors, a
// finding that will not validate, a storage failure — all of them cost this one
// detector's cards and leave the rest of the panel intact, because a home screen
// that goes blank because one aggregate broke is worse than one that is missing
// a section.
func (s *Service) refreshDetector(ctx context.Context, p refreshParams) {
	log := s.l.With(zap.String("detector", p.detector.Key()))

	findings, err := p.detector.Detect(ctx, p.params)
	if err != nil {
		log.Error("detector failed", zap.Error(err))
		p.result.Failed = append(p.result.Failed, p.detector.Key())

		return
	}

	valid := make([]detector.Finding, 0, len(findings))
	for _, finding := range findings {
		if vErr := finding.Validate(); vErr != nil {
			// A malformed finding is a programming error in the detector. It is
			// dropped rather than stored, and logged loudly enough to be fixed.
			log.Error("dropping malformed finding", zap.Error(vErr))

			continue
		}
		valid = append(valid, finding)
	}

	narrations := s.narrator.Narrate(ctx, &narrator.NarrateRequest{
		TenantInfo: p.params.TenantInfo,
		Findings:   valid,
		WindowDays: p.params.WindowDays(),
	})

	entities := s.buildInsights(p, valid, narrations)

	stored, err := s.repo.ReplaceDetectorFindings(
		ctx,
		repositories.ReplaceDetectorFindingsRequest{
			TenantInfo:       p.params.TenantInfo,
			DetectorKey:      p.detector.Key(),
			Insights:         entities,
			SuppressedBefore: p.now - int64(insight.DismissalSuppression.Seconds()),
		},
	)
	if err != nil {
		log.Error("failed to store findings", zap.Error(err))
		p.result.Failed = append(p.result.Failed, p.detector.Key())

		return
	}

	p.result.Created += stored.Created
	p.result.Superseded += stored.Superseded
	p.result.Resolved += stored.Resolved
	p.result.Suppressed += stored.Suppressed
}

func (s *Service) buildInsights(
	p refreshParams,
	findings []detector.Finding,
	narrations map[string]narrator.Narration,
) []*insight.Insight {
	entities := make([]*insight.Insight, 0, len(findings))

	for _, finding := range findings {
		narration := narrations[finding.DedupeKey]
		if narration.Narrated {
			p.result.Narrated++
		}

		entity := &insight.Insight{
			OrganizationID: p.params.TenantInfo.OrgID,
			BusinessUnitID: p.params.TenantInfo.BuID,
			DetectorKey:    p.detector.Key(),
			Category:       p.detector.Category(),
			Severity:       finding.Severity,
			Status:         insight.StatusActive,
			DedupeKey:      finding.DedupeKey,
			Subject:        finding.Subject,
			// Wording comes from the narration, which is the detector's own
			// headline when no model wrote one. Everything else on this entity
			// comes from the finding.
			Headline:        narration.Headline,
			Narrative:       narration.Narrative,
			Recommendation:  narration.Recommendation,
			Narrated:        narration.Narrated,
			ModelIdentifier: narration.ModelIdentifier,
			ProviderID:      narration.ProviderID,
			Metrics:         finding.Metrics,
			Links:           finding.Links,
			WindowStart:     p.params.WindowStart,
			WindowEnd:       p.params.WindowEnd,
			DetectedAt:      p.now,
			StaleAt:         p.now + int64(StaleAfter.Seconds()),
		}

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		if multiErr.HasErrors() {
			s.l.Error("dropping insight that failed validation",
				zap.String("detector", p.detector.Key()),
				zap.String("dedupeKey", finding.DedupeKey),
				zap.Error(multiErr),
			)

			continue
		}

		entities = append(entities, entity)
	}

	return entities
}

// ListActive returns what a reader is allowed to see.
//
// Filtering by permission here rather than at render time is the point: an
// insight about customer profitability names a customer and a figure in its
// headline, and a person who cannot read customers must not receive it in a
// payload at all.
func (s *Service) ListActive(
	ctx context.Context,
	req services.ListInsightsRequest,
) ([]*insight.Insight, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = widgetLimit
	}

	found, err := s.repo.ListActive(ctx, repositories.ListActiveInsightsRequest{
		TenantInfo: req.TenantInfo,
		Categories: req.Categories,
		// Read more than asked for, because permission filtering below removes
		// some and a widget asking for five should still get five where five are
		// visible to that reader.
		Limit: limit * 2,
	})
	if err != nil {
		return nil, err
	}

	visible := s.filterByPermission(ctx, req, found)
	if len(visible) > limit {
		visible = visible[:limit]
	}

	return visible, nil
}

func (s *Service) filterByPermission(
	ctx context.Context,
	req services.ListInsightsRequest,
	found []*insight.Insight,
) []*insight.Insight {
	visible := make([]*insight.Insight, 0, len(found))
	// Detectors repeat across insights, so the permission for a given detector is
	// resolved once rather than per row.
	decided := make(map[string]bool, len(s.detectors.All()))

	for _, entity := range found {
		allowed, seen := decided[entity.DetectorKey]
		if !seen {
			allowed = s.readerMaySee(ctx, req, entity.DetectorKey)
			decided[entity.DetectorKey] = allowed
		}

		if allowed {
			visible = append(visible, entity)
		}
	}

	return visible
}

// readerMaySee answers whether this reader holds the permission the detector's
// subject matter requires.
//
// A detector that is no longer registered — turned off in a release, while its
// findings are still stored — is refused rather than allowed. There is nothing
// left to say what permission it needed, and defaulting to visible is how a
// retired detector's cards outlive the check that guarded them.
func (s *Service) readerMaySee(
	ctx context.Context,
	req services.ListInsightsRequest,
	detectorKey string,
) bool {
	d, ok := s.detectors.Get(detectorKey)
	if !ok {
		return false
	}

	result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    req.UserID,
		UserID:         req.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Resource:       d.Resource().String(),
		Operation:      d.Operation(),
	})
	if err != nil {
		s.l.Error("failed to check insight permission",
			zap.String("detector", detectorKey),
			zap.Error(err),
		)

		return false
	}

	return result.Allowed
}

// Dismiss records that a person judged a finding not worth acting on.
func (s *Service) Dismiss(
	ctx context.Context,
	req services.DismissInsightRequest,
) (*insight.Insight, error) {
	return s.repo.Dismiss(ctx, repositories.DismissInsightRequest{
		ID:         req.ID,
		UserID:     req.UserID,
		Reason:     req.Reason,
		TenantInfo: req.TenantInfo,
	})
}
