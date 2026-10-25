package capture

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
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
	keel.Table `keel:"capture_pages"`

	ID             pulid.ID    `json:"id"             keel:"id,pk,autoid:cpg_"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID    `json:"organizationId" keel:"organization_id,pk,tenant"`
	BatchID        pulid.ID    `json:"batchId"        keel:"batch_id"`
	Sequence       int         `json:"sequence"       keel:"sequence"`
	Status         PageStatus  `json:"status"         keel:"status"`
	StoragePath    string      `json:"-"              keel:"storage_path"`
	ChecksumSHA256 string      `json:"checksumSha256" keel:"checksum_sha256"`
	ByteSize       int64       `json:"byteSize"       keel:"byte_size"`
	ContentType    string      `json:"contentType"    keel:"content_type"`
	WidthPx        int         `json:"widthPx"        keel:"width_px"`
	HeightPx       int         `json:"heightPx"       keel:"height_px"`
	DPI            int         `json:"dpi"            keel:"dpi"`
	Rotation       int         `json:"rotation"       keel:"rotation"`
	ThumbnailPath  string      `json:"-"              keel:"thumbnail_path,nullzero"`
	BlankScore     *float64    `json:"blankScore"     keel:"blank_score"`
	IsSeparator    bool        `json:"isSeparator"    keel:"is_separator"`
	Markers        PageMarkers `json:"markers"        keel:"markers,jsonb"`
	FailureMessage string      `json:"failureMessage" keel:"failure_message,nullzero"`
	CreatedAt      int64       `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64       `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

// PageMarkers are what a page says about where the stack divides.
type PageMarkers struct {
	// PatchCode is the patch sheet the scanner reported (1, 2, 3, 4, 6 or T),
	// empty when there was none. Only the device can see these: the scanner
	// reads them from the paper as it passes.
	PatchCode string `json:"patchCode,omitempty"`
	// DeviceBarcodes are the barcodes the scanner's own driver decoded.
	DeviceBarcodes []string `json:"deviceBarcodes,omitempty"`
	// ReadCodes are the codes the server read off the page that are not
	// cover sheets: a QR code printed on a bill of lading, say.
	ReadCodes []string `json:"readCodes,omitempty"`
	// CoverSheetID is set when the server read a cover sheet on this page and
	// found it genuine for this organization.
	CoverSheetID *pulid.ID `json:"coverSheetId,omitempty"`
	// UnrecognizedCoverSheet is a page that looked like a cover sheet but did
	// not verify. It still divides the stack; it routes nothing.
	UnrecognizedCoverSheet bool `json:"unrecognizedCoverSheet,omitempty"`
}

// Codes are every code found on the page, the scanner's first.
func (m PageMarkers) Codes() []string {
	return append(append(make([]string, 0, len(m.DeviceBarcodes)+len(m.ReadCodes)),
		m.DeviceBarcodes...), m.ReadCodes...)
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

func (p *CapturePage) GetID() pulid.ID      { return p.ID }
func (p *CapturePage) GetCreatedAt() int64  { return p.CreatedAt }
func (p *CapturePage) GetTableName() string { return "capture_pages" }

func (p *CapturePage) BeforeCreate(ctx context.Context) error {
	if p.Status == "" {
		p.Status = PageReceived
	}

	return nil
}
