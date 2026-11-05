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
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/updateconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const failedCheckCooldown = 5 * time.Minute

var ErrNoReleaseFound = errors.New("no release found")

type Params struct {
	fx.In

	Logger       *zap.Logger
	AppConfig    *appconfig.Config
	UpdateConfig *updateconfig.Config
}

type Service struct {
	l            *zap.Logger
	appConfig    *appconfig.Config
	updateConfig *updateconfig.Config
	httpClient   *http.Client

	checks singleflight.Group

	mu           sync.RWMutex
	cachedStatus *kernelports.UpdateStatus
	nextCheck    time.Time
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.version"),
		appConfig:    p.AppConfig,
		updateConfig: p.UpdateConfig,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *Service) GetVersionInfo(_ context.Context) (*kernelports.VersionInfo, error) {
	return &kernelports.VersionInfo{
		Version:     s.appConfig.Version,
		Environment: s.appConfig.Env,
	}, nil
}

func (s *Service) GetUpdateStatus(ctx context.Context) (*kernelports.UpdateStatus, error) {
	if !s.updateConfig.Enabled {
		return &kernelports.UpdateStatus{
			CurrentVersion:  s.appConfig.Version,
			UpdateAvailable: false,
			LastChecked:     0,
		}, nil
	}

	if s.updateConfig.OfflineMode {
		s.l.Debug("offline mode enabled, skipping remote update check")
		return s.noUpdateStatus(), nil
	}

	if status := s.currentStatus(); status != nil {
		return status, nil
	}

	checked, err, _ := s.checks.Do("latest-release", func() (any, error) {
		if status := s.currentStatus(); status != nil {
			return status, nil
		}

		return s.refreshStatus(context.WithoutCancel(ctx)), nil
	})
	if err != nil {
		return nil, err
	}

	status, ok := checked.(*kernelports.UpdateStatus)
	if !ok {
		return nil, fmt.Errorf("unexpected update status type %T", checked)
	}

	return status, nil
}

func (s *Service) currentStatus() *kernelports.UpdateStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cachedStatus != nil && time.Now().Before(s.nextCheck) {
		return s.cachedStatus
	}

	return nil
}

func (s *Service) refreshStatus(ctx context.Context) *kernelports.UpdateStatus {
	release, err := s.fetchLatestRelease(ctx)
	switch {
	case errors.Is(err, ErrNoReleaseFound):
		return s.storeStatus(s.noUpdateStatus(), s.updateConfig.GetCheckInterval())
	case err != nil:
		s.l.Warn("failed to fetch latest release", zap.Error(err))
		return s.storeStatus(s.lastKnownStatus(), failedCheckCooldown)
	}

	return s.storeStatus(&kernelports.UpdateStatus{
		CurrentVersion:  s.appConfig.Version,
		LatestVersion:   release.Version,
		UpdateAvailable: s.isNewerVersion(release.Version, s.appConfig.Version),
		LatestRelease:   release,
		LastChecked:     timeutils.NowUnix(),
	}, s.updateConfig.GetCheckInterval())
}

func (s *Service) noUpdateStatus() *kernelports.UpdateStatus {
	return &kernelports.UpdateStatus{
		CurrentVersion:  s.appConfig.Version,
		UpdateAvailable: false,
		LastChecked:     timeutils.NowUnix(),
	}
}

func (s *Service) lastKnownStatus() *kernelports.UpdateStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cachedStatus != nil {
		return s.cachedStatus
	}

	return s.noUpdateStatus()
}

func (s *Service) storeStatus(
	status *kernelports.UpdateStatus,
	validFor time.Duration,
) *kernelports.UpdateStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cachedStatus = status
	s.nextCheck = time.Now().Add(validFor)

	return status
}

func (s *Service) fetchLatestRelease(ctx context.Context) (*kernelports.ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		s.updateConfig.GetGitHubOwner(),
		s.updateConfig.GetGitHubRepo(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("Trenova/%s", s.appConfig.Version))

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

	var ghRelease kernelports.GitHubRelease
	if err = sonic.Unmarshal(body, &ghRelease); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if ghRelease.Draft {
		return nil, ErrNoReleaseFound
	}

	if ghRelease.Prerelease && !s.updateConfig.AllowPrerelease {
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

	return &kernelports.ReleaseInfo{
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
