//go:build fuzz

package copybinary_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/copybinary"
)

func FuzzRowsFromAnArchiveAreReadWithinTheirBoundsOrRefused(f *testing.F) {
	f.Add(encode([][][]byte{{[]byte("shp_1"), nil}}), uint8(2))
	f.Add(encode([][][]byte{{[]byte("a")}, {[]byte("b")}}), uint8(1))
	f.Add(encode(nil), uint8(3))
	huge := encode(nil)
	huge = huge[:len(huge)-2]
	huge = binary.BigEndian.AppendUint16(huge, 1)
	huge = binary.BigEndian.AppendUint32(huge, 0x7fffffff)
	f.Add(huge, uint8(1))
	f.Add([]byte("PGCOPY\n\xff\r\n\x00\x00\x00\x00\x00\xff\xff\xff\xff"), uint8(1))
	f.Add([]byte{}, uint8(0))

	f.Fuzz(func(t *testing.T, stream []byte, columns uint8) {
		width := int(columns%8) + 1
		var counted bytes.Buffer
		rows, countErr := copybinary.Count(&counted, bytes.NewReader(stream), width)
		if rows < 0 {
			t.Fatalf("counted %d rows", rows)
		}
		if countErr == nil && !bytes.Equal(counted.Bytes(), stream) {
			t.Fatalf("a stream that counted was not passed through unchanged")
		}

		var rewritten bytes.Buffer
		again, rewriteErr := copybinary.Rewrite(&rewritten, bytes.NewReader(stream), width, nil)
		if (countErr == nil) != (rewriteErr == nil) {
			t.Fatalf("count said %v and rewrite said %v of one stream", countErr, rewriteErr)
		}
		if countErr == nil && (again != rows || !bytes.Equal(rewritten.Bytes(), stream)) {
			t.Fatalf("an unchanged rewrite gave %d rows and different bytes", again)
		}
		if _, err := copybinary.Count(io.Discard, bytes.NewReader(stream), width); (err == nil) != (countErr == nil) {
			t.Fatalf("counting twice disagreed")
		}
	})
}
