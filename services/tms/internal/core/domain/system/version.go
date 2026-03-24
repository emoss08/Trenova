package system

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

type UpdateHistoryEntry struct {
	ID           string `json:"id"`
	FromVersion  string `json:"fromVersion"`
	ToVersion    string `json:"toVersion"`
	Status       string `json:"status"`
	BackupPath   string `json:"backupPath,omitempty"`
	StartedAt    int64  `json:"startedAt"`
	CompletedAt  int64  `json:"completedAt,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
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
