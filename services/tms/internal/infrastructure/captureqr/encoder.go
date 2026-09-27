// Package captureqr encodes the QR code printed on a capture cover sheet.
//
// It is apart from captureimaging, which reads codes off rendered pages and
// needs MuPDF, because issuing a sheet needs neither.
package captureqr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode/decoder"
	"github.com/makiuchi-d/gozxing/qrcode/encoder"
)

// maxPayloadLength bounds what a sheet may carry. A cover sheet payload is a
// prefix and a token, well under a hundred characters; anything longer is a
// caller's mistake, not a bigger code.
const maxPayloadLength = 256

var ErrPayloadInvalid = errors.New("a cover sheet payload must be 1 to 256 characters")

type Encoder struct{}

func New() services.CaptureQREncoder {
	return &Encoder{}
}

// Encode draws the payload at error correction level Q, which survives about
// a quarter of the code being lost: a staple, a coffee ring, a fax's streaks.
func (e *Encoder) Encode(payload string) (*services.CaptureQRCode, error) {
	if payload == "" || len(payload) > maxPayloadLength {
		return nil, ErrPayloadInvalid
	}

	hints := map[gozxing.EncodeHintType]any{
		gozxing.EncodeHintType_CHARACTER_SET: "UTF-8",
	}
	code, err := encoder.Encoder_encode(payload, decoder.ErrorCorrectionLevel_Q, hints)
	if err != nil {
		return nil, fmt.Errorf("encode cover sheet code: %w", err)
	}

	matrix := code.GetMatrix()
	size := matrix.GetWidth()
	rows := make([]string, 0, size)

	var row strings.Builder
	row.Grow(size)
	for y := range size {
		row.Reset()
		for x := range size {
			if matrix.Get(x, y) == 1 {
				row.WriteByte('1')
			} else {
				row.WriteByte('0')
			}
		}
		rows = append(rows, row.String())
	}

	return &services.CaptureQRCode{Size: size, Modules: rows}, nil
}
