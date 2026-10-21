// Package captureqr encodes the QR code printed on a capture cover sheet.
//
// Reading codes off scanned pages is the imaging worker's; issuing a sheet
// needs only the encoder.
package captureqr

import (
	"errors"
	"fmt"
	"image/color"
	"strings"

	"github.com/boombuler/barcode/qr"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
)

// maxPayloadLength bounds what a sheet may carry. A cover sheet payload is a
// prefix and a token, well under a hundred characters; anything longer is a
// caller's mistake, not a bigger code.
const maxPayloadLength = 256

var ErrPayloadInvalid = errors.New("a cover sheet payload must be 1 to 256 characters")

type Encoder struct{}

func New() captureports.CaptureQREncoder {
	return &Encoder{}
}

// Encode draws the payload at error correction level Q, which survives about
// a quarter of the code being lost: a staple, a coffee ring, a fax's streaks.
func (e *Encoder) Encode(payload string) (*captureports.CaptureQRCode, error) {
	if payload == "" || len(payload) > maxPayloadLength {
		return nil, ErrPayloadInvalid
	}

	code, err := qr.Encode(payload, qr.Q, qr.Unicode)
	if err != nil {
		return nil, fmt.Errorf("encode cover sheet code: %w", err)
	}

	size := code.Bounds().Dx()
	rows := make([]string, 0, size)

	var row strings.Builder
	row.Grow(size)
	for y := range size {
		row.Reset()
		for x := range size {
			if dark(code.At(x, y)) {
				row.WriteByte('1')
			} else {
				row.WriteByte('0')
			}
		}
		rows = append(rows, row.String())
	}

	return &captureports.CaptureQRCode{Size: size, Modules: rows}, nil
}

func dark(c color.Color) bool {
	luma, _, _, _ := color.GrayModel.Convert(c).RGBA()
	return luma < 0x8000
}
