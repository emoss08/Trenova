package report

import "github.com/emoss08/keel"

type ReportCatalogVersion struct {
	keel.Table `keel:"report_catalog_versions"`

	Version     string `json:"version"     keel:"version,pk"`
	FirstSeenAt int64  `json:"firstSeenAt" keel:"first_seen_at"`
	LastSeenAt  int64  `json:"lastSeenAt"  keel:"last_seen_at"`
}
