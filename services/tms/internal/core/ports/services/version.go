package services

import (
	"context"
)

type ReleaseInfo struct {
	Version      string `json:"version"`
	TagName      string `json:"tagName"`
	PublishedAt  int64  `json:"publishedAt"`
	ReleaseNotes string `json:"releaseNotes"`
	DownloadURL  string `json:"downloadUrl"`
	HTMLURL      string `json:"htmlUrl"`
	IsPrerelease bool   `json:"isPrerelease"`
}

type UpdateStatus struct {
	CurrentVersion  string       `json:"currentVersion"`
	LatestVersion   string       `json:"latestVersion"`
	UpdateAvailable bool         `json:"updateAvailable"`
	LatestRelease   *ReleaseInfo `json:"latestRelease,omitempty"`
	LastChecked     int64        `json:"lastChecked"`
}

type VersionInfo struct {
	Version     string `json:"version"`
	Environment string `json:"environment"`
	BuildDate   string `json:"buildDate,omitempty"`
	GitCommit   string `json:"gitCommit,omitempty"`
}

type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type VersionService interface {
	GetVersionInfo(ctx context.Context) (*VersionInfo, error)
	GetUpdateStatus(ctx context.Context) (*UpdateStatus, error)
	CheckForUpdates(ctx context.Context) (*UpdateStatus, error)
}
