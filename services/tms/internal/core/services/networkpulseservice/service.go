package networkpulseservice

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// WindowDays is the reporting window for the on-time figure: "on-time this week".
const WindowDays = 7

// LaneLimit caps how many lanes the panel is handed. The chip band only has room for a
// dozen or so before it repeats, and a smaller set is also less to disclose.
const LaneLimit = 12

const secondsPerDay = int64(24 * time.Hour / time.Second)

var ErrDisabled = errortypes.NewNotFoundError("Network pulse is not enabled")

type Params struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
	Repo   repositories.NetworkPulseRepository
}

type Service struct {
	l    *zap.Logger
	cfg  *config.Config
	repo repositories.NetworkPulseRepository

	// The endpoint is unauthenticated, so every call is a database aggregate an
	// anonymous caller can trigger. A single shared snapshot behind a short TTL keeps
	// repeated page loads off the database entirely.
	mu       sync.RWMutex
	cached   *services.NetworkPulse
	cachedAt time.Time
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.network-pulse"),
		cfg:  p.Config,
		repo: p.Repo,
	}
}

func (s *Service) Enabled() bool {
	return s.cfg.System.NetworkPulse.Enabled
}

func (s *Service) GetNetworkPulse(ctx context.Context) (*services.NetworkPulse, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}

	ttl := s.cfg.System.NetworkPulse.GetCacheTTL()
	if cached := s.readCache(ttl); cached != nil {
		return cached, nil
	}

	since := timeutils.NowUnix() - int64(WindowDays)*secondsPerDay
	counts, err := s.repo.GetNetworkPulse(ctx, since, LaneLimit)
	if err != nil {
		return nil, err
	}

	pulse := &services.NetworkPulse{
		LoadsInMotion: counts.LoadsInMotion,
		OnTimePercent: onTimePercent(counts.OnTimeCount, counts.OnTimeTotal),
		SampleSize:    counts.OnTimeTotal,
		WindowDays:    WindowDays,
		Lanes:         toLanes(counts.Lanes),
	}

	s.writeCache(pulse)
	return pulse, nil
}

func (s *Service) readCache(ttl time.Duration) *services.NetworkPulse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cached == nil || time.Since(s.cachedAt) >= ttl {
		return nil
	}
	return s.cached
}

func (s *Service) writeCache(pulse *services.NetworkPulse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cached = pulse
	s.cachedAt = time.Now()
}

// toLanes always yields a non-nil slice: the field is rendered by a client that treats
// an absent list and an empty one the same way, and a JSON null would make that a
// distinction it has to defend against for no reason.
func toLanes(lanes []repositories.NetworkPulseLane) []services.NetworkPulseLane {
	result := make([]services.NetworkPulseLane, 0, len(lanes))
	for _, lane := range lanes {
		result = append(result, services.NetworkPulseLane{
			From:   lane.OriginState,
			To:     lane.DestinationState,
			Status: lane.Status,
			Count:  lane.Count,
		})
	}
	return result
}

// onTimePercent reports to one decimal place, matching the in-app service-level card.
// An empty window scores zero, which the SampleSize of 0 alongside it marks as absent
// rather than as a perfect miss.
func onTimePercent(onTime, total int) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(onTime)/float64(total)*1000) / 10
}
