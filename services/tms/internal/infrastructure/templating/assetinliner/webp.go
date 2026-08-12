package assetinliner

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/emoss08/trenova/shared/imageutils"
	"github.com/gen2brain/webp"
)

// NewWebPTranscoder returns the WebP-to-PNG converter imageutils asks for.
//
// It lives here rather than in shared/imageutils to keep the decoder dependency out
// of shared/, which is imported by gtc and samsara-sim. Neither service inlines
// template assets, so neither needs to carry a WebP codec.
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
