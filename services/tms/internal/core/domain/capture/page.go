package capture

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	// MaxPageBytes bounds one uploaded page. A letter page at 600 DPI in colour
	// is a few megabytes as JPEG; twenty leaves room for legal and tabloid
	// without letting one request tie up a worker.
	MaxPageBytes = 20 << 20
	// PageContentType is the only thing a scanned page may be. The companion
	// wraps every page as a one-page PDF so the rest of the pipeline reads one
	// format.
	PageContentType = "application/pdf"
	// BlankThreshold is the ink coverage under which a page counts as blank. It
	// sits above scanner noise and a stray staple shadow, and well below the
	// lightest real page (a mostly empty POD with one signature).
	BlankThreshold = 0.004
)

// CapturePage is one scanned side, or one page of a print job.
type CapturePage struct {
	bun.BaseModel `bun:"table:capture_pages,alias:cpg" json:"-"`

	ID             pulid.ID    `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BatchID        pulid.ID    `json:"batchId"        bun:"batch_id,type:VARCHAR(100),notnull"`
	Sequence       int         `json:"sequence"       bun:"sequence,type:INTEGER,notnull"`
	Status         PageStatus  `json:"status"         bun:"status,type:VARCHAR(20),notnull"`
	StoragePath    string      `json:"-"              bun:"storage_path,type:VARCHAR(512),notnull"`
	ChecksumSHA256 string      `json:"checksumSha256" bun:"checksum_sha256,type:VARCHAR(64),notnull"`
	ByteSize       int64       `json:"byteSize"       bun:"byte_size,type:BIGINT,notnull"`
	ContentType    string      `json:"contentType"    bun:"content_type,type:VARCHAR(100),notnull"`
	WidthPx        int         `json:"widthPx"        bun:"width_px,type:INTEGER,notnull,default:0"`
	HeightPx       int         `json:"heightPx"       bun:"height_px,type:INTEGER,notnull,default:0"`
	DPI            int         `json:"dpi"            bun:"dpi,type:INTEGER,notnull,default:0"`
	Rotation       int         `json:"rotation"       bun:"rotation,type:INTEGER,notnull,default:0"`
	ThumbnailPath  string      `json:"-"              bun:"thumbnail_path,type:VARCHAR(512),nullzero"`
	BlankScore     *float64    `json:"blankScore"     bun:"blank_score,type:NUMERIC(6,5),nullzero"`
	IsSeparator    bool        `json:"isSeparator"    bun:"is_separator,type:BOOLEAN,notnull,default:false"`
	Markers        PageMarkers `json:"markers"        bun:"markers,type:JSONB,notnull,default:'{}'"`
	FailureMessage string      `json:"failureMessage" bun:"failure_message,type:VARCHAR(500),nullzero"`
	CreatedAt      int64       `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64       `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// PageMarkers are what a page says about where the stack divides.
type PageMarkers struct {
	// PatchCode is the patch sheet the scanner reported (1, 2, 3, 4, 6 or T),
	// empty when there was none. Only the device can see these: the scanner
	// reads them from the paper as it passes.
	PatchCode string `json:"patchCode,omitempty"`
	// DeviceBarcodes are the barcodes the scanner's own driver decoded.
	DeviceBarcodes []string `json:"deviceBarcodes,omitempty"`
	// CoverSheetID is set when the server read a cover sheet on this page and
	// found it genuine for this organization.
	CoverSheetID *pulid.ID `json:"coverSheetId,omitempty"`
	// UnrecognizedCoverSheet is a page that looked like a cover sheet but did
	// not verify. It still divides the stack; it routes nothing.
	UnrecognizedCoverSheet bool `json:"unrecognizedCoverSheet,omitempty"`
}

// IsBlank reports whether the page carries nothing worth keeping.
func (p *CapturePage) IsBlank() bool {
	return p.BlankScore != nil && *p.BlankScore < BlankThreshold
}

// NormalizeRotation keeps a rotation to the four a page can have.
func NormalizeRotation(degrees int) int {
	r := degrees % 360
	if r < 0 {
		r += 360
	}

	return (r / 90) * 90
}

func (p *CapturePage) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("cpg_")
		}
		if p.Status == "" {
			p.Status = PageReceived
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

func (p *CapturePage) GetID() pulid.ID      { return p.ID }
func (p *CapturePage) GetCreatedAt() int64  { return p.CreatedAt }
func (p *CapturePage) GetTableName() string { return "capture_pages" }
