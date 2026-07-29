package assetinliner

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/chai2010/webp"
	"github.com/emoss08/trenova/shared/imageutils"
)

// NewWebPTranscoder returns the WebP-to-PNG converter imageutils asks for.
//
// It lives here rather than in shared/imageutils because the only usable Go WebP
// decoder requires CGO, and shared/ is imported by services that build without a C
// toolchain. Keeping the dependency on this side of the boundary means adding WebP
// support did not force CGO onto gtc or samsara-sim.
func NewWebPTranscoder() imageutils.WebPTranscoder {
	return func(body []byte) ([]byte, error) {
		img, err := webp.Decode(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("decode webp: %w", err)
		}

		var buf bytes.Buffer
		if err = png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("encode png: %w", err)
		}

		return buf.Bytes(), nil
	}
}
