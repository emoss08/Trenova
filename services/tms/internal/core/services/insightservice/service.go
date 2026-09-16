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
	"errors"
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

// ListActive returns the home screen's slice of what a reader may see.
func (s *Service) ListActive(
	ctx context.Context,
	req services.ListInsightsRequest,
) ([]*insight.Insight, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = widgetLimit
	}

	return s.repo.ListActive(ctx, repositories.ListActiveInsightsRequest{
		TenantInfo:          req.TenantInfo,
		AllowedDetectorKeys: s.allowedDetectorKeys(ctx, req),
		Categories:          req.Categories,
		Limit:               limit,
	})
}

// List browses the history a page at a time.
func (s *Service) List(
	ctx context.Context,
	req services.BrowseInsightsRequest,
) (*pagination.ListResult[*insight.Insight], error) {
	return s.repo.List(ctx, repositories.ListInsightsRequest{
		TenantInfo: req.TenantInfo,
		AllowedDetectorKeys: s.allowedDetectorKeys(ctx, services.ListInsightsRequest{
			TenantInfo: req.TenantInfo,
			UserID:     req.UserID,
		}),
		Categories: req.Categories,
		Severities: req.Severities,
		Statuses:   req.Statuses,
		Limit:      req.Limit,
		Offset:     req.Offset,
	})
}

// allowedDetectorKeys is the set of detectors whose findings this reader may
// receive.
//
// Resolving it here and pushing it into the query is what makes a paginated page
// honest: filtering rows after they come back gives short pages and a total that
// counts findings the reader will never see. It is also cheaper — one permission
// check per detector rather than per row.
//
// An insight names a customer, a location or a driver alongside a figure, so a
// reader who cannot see those records must not receive the finding at all, not
// merely be prevented from clicking through to it.
func (s *Service) allowedDetectorKeys(
	ctx context.Context,
	req services.ListInsightsRequest,
) repositories.AllowedDetectorKeys {
	detectors := s.detectors.All()
	allowed := make(repositories.AllowedDetectorKeys, 0, len(detectors))

	for _, d := range detectors {
		if s.readerMaySee(ctx, req, d) {
			allowed = append(allowed, d.Key())
		}
	}

	return allowed
}

// readerMaySee answers whether this reader holds the permission a detector's
// subject matter requires.
//
// Only registered detectors are ever asked about. A detector turned off in a
// release leaves its findings in the table, and nothing is left to say what
// permission they needed — so its key never joins the allowed set and those rows
// stay out of every read. Defaulting the other way is how a retired rule's cards
// outlive the check that guarded them.
func (s *Service) readerMaySee(
	ctx context.Context,
	req services.ListInsightsRequest,
	d detector.Detector,
) bool {
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
		// A permission service that cannot answer is not permission granted.
		s.l.Error("failed to check insight permission",
			zap.String("detector", d.Key()),
			zap.Error(err),
		)

		return false
	}

	return result.Allowed
}

// ErrInsightNotVisible reports a finding the reader may not see. It is returned
// rather than a not-found so the caller can decide how to phrase it; the handler
// turns it into the same 404 a missing insight produces, because telling someone
// a finding exists but is not for them is itself a disclosure.
var ErrInsightNotVisible = errors.New("insight is not visible to this reader")

// GetDetail reads one finding with its history and the rule behind it.
//
// The permission check happens here rather than being left to the query, because
// unlike a list this reads a single row by id: there is no filter to fold the
// restriction into, and a reader who guesses an id must not receive a finding
// their permissions exclude.
func (s *Service) GetDetail(
	ctx context.Context,
	req services.GetInsightDetailRequest,
) (*services.InsightDetail, error) {
	found, err := s.repo.GetByID(ctx, repositories.GetInsightByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	d, registered := s.detectors.Get(found.DetectorKey)
	if !registered {
		// A retired detector leaves findings with nothing left to say what
		// permission they needed. Refusing is the only safe reading.
		return nil, ErrInsightNotVisible
	}

	visible := s.readerMaySee(ctx, services.ListInsightsRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
	}, d)
	if !visible {
		return nil, ErrInsightNotVisible
	}

	history, err := s.repo.ListHistory(ctx, repositories.ListInsightHistoryRequest{
		DedupeKey:  found.DedupeKey,
		TenantInfo: req.TenantInfo,
		ExcludeID:  found.ID,
	})
	if err != nil {
		// A finding without its trend is still worth reading, so a failed history
		// read degrades the view rather than failing it.
		s.l.Error("failed to read insight history",
			zap.String("insight", found.ID.String()),
			zap.Error(err),
		)
		history = nil
	}

	explanation := d.Explain()

	return &services.InsightDetail{
		Insight: found,
		History: history,
		Explanation: services.InsightExplanation{
			Measures:  explanation.Measures,
			Threshold: explanation.Threshold,
			Excludes:  explanation.Excludes,
		},
	}, nil
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

// Restore undoes a dismissal.
func (s *Service) Restore(
	ctx context.Context,
	req services.RestoreInsightRequest,
) (*insight.Insight, error) {
	return s.repo.Restore(ctx, repositories.RestoreInsightRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
}
