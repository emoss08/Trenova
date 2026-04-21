package versionservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrNoReleaseFound = errors.New("no release found")

type Params struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
}

type Service struct {
	l          *zap.Logger
	cfg        *config.Config
	httpClient *http.Client

	mu           sync.RWMutex
	cachedStatus *serviceports.UpdateStatus
	lastCheck    time.Time
}

func New(p Params) *Service {
	return &Service{
		l:   p.Logger.Named("service.version"),
		cfg: p.Config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *Service) GetVersionInfo(_ context.Context) (*serviceports.VersionInfo, error) {
	return &serviceports.VersionInfo{
		Version:     s.cfg.App.Version,
		Environment: s.cfg.App.Env,
	}, nil
}

func (s *Service) GetUpdateStatus(ctx context.Context) (*serviceports.UpdateStatus, error) {
	if !s.cfg.Update.Enabled {
		return &serviceports.UpdateStatus{
			CurrentVersion:  s.cfg.App.Version,
			UpdateAvailable: false,
			LastChecked:     0,
		}, nil
	}

	s.mu.RLock()
	cached := s.cachedStatus
	lastCheck := s.lastCheck
	s.mu.RUnlock()

	if cached != nil && time.Since(lastCheck) < s.cfg.Update.GetCheckInterval() {
		return cached, nil
	}

	return s.CheckForUpdates(ctx)
}

func (s *Service) CheckForUpdates(ctx context.Context) (*serviceports.UpdateStatus, error) {
	if !s.cfg.Update.Enabled {
		return &serviceports.UpdateStatus{
			CurrentVersion:  s.cfg.App.Version,
			UpdateAvailable: false,
			LastChecked:     timeutils.NowUnix(),
		}, nil
	}

	if s.cfg.Update.OfflineMode {
		s.l.Debug("offline mode enabled, skipping remote update check")
		return &serviceports.UpdateStatus{
			CurrentVersion:  s.cfg.App.Version,
			UpdateAvailable: false,
			LastChecked:     timeutils.NowUnix(),
		}, nil
	}

	release, err := s.fetchLatestRelease(ctx)
	if err != nil {
		if errors.Is(err, ErrNoReleaseFound) {
			return &serviceports.UpdateStatus{
				CurrentVersion:  s.cfg.App.Version,
				UpdateAvailable: false,
				LastChecked:     timeutils.NowUnix(),
			}, nil
		}
		s.l.Warn("failed to fetch latest release", zap.Error(err))
		return &serviceports.UpdateStatus{
			CurrentVersion:  s.cfg.App.Version,
			UpdateAvailable: false,
			LastChecked:     timeutils.NowUnix(),
		}, nil
	}

	status := &serviceports.UpdateStatus{
		CurrentVersion:  s.cfg.App.Version,
		LatestVersion:   release.Version,
		UpdateAvailable: s.isNewerVersion(release.Version, s.cfg.App.Version),
		LatestRelease:   release,
		LastChecked:     timeutils.NowUnix(),
	}

	s.mu.Lock()
	s.cachedStatus = status
	s.lastCheck = time.Now()
	s.mu.Unlock()

	return status, nil
}

func (s *Service) fetchLatestRelease(ctx context.Context) (*serviceports.ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		s.cfg.Update.GetGitHubOwner(),
		s.cfg.Update.GetGitHubRepo(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("Trenova/%s", s.cfg.App.Version))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNoReleaseFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var ghRelease serviceports.GitHubRelease
	if err = sonic.Unmarshal(body, &ghRelease); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if ghRelease.Draft {
		return nil, ErrNoReleaseFound
	}

	if ghRelease.Prerelease && !s.cfg.Update.AllowPrerelease {
		return nil, ErrNoReleaseFound
	}

	publishedAt, _ := time.Parse(time.RFC3339, ghRelease.PublishedAt)

	var downloadURL string
	for _, asset := range ghRelease.Assets {
		if strings.Contains(asset.Name, "linux") || strings.Contains(asset.Name, "docker") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return &serviceports.ReleaseInfo{
		Version:      strings.TrimPrefix(ghRelease.TagName, "v"),
		TagName:      ghRelease.TagName,
		PublishedAt:  publishedAt.Unix(),
		ReleaseNotes: ghRelease.Body,
		DownloadURL:  downloadURL,
		HTMLURL:      ghRelease.HTMLURL,
		IsPrerelease: ghRelease.Prerelease,
	}, nil
}

func (s *Service) isNewerVersion(latest, current string) bool {
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	if latest == current {
		return false
	}

	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		latestNum, err := strconv.Atoi(latestParts[i])
		if err != nil {
			latestNum = 0
		}
		currentNum, err := strconv.Atoi(currentParts[i])
		if err != nil {
			currentNum = 0
		}

		if latestNum > currentNum {
			return true
		}
		if latestNum < currentNum {
			return false
		}
	}

	return len(latestParts) > len(currentParts)
}
