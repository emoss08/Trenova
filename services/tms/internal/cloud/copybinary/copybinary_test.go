package copybinary_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/copybinary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encode(rows [][][]byte) []byte {
	var b bytes.Buffer
	b.Write(copybinary.Signature)
	_ = binary.Write(&b, binary.BigEndian, int32(0))
	_ = binary.Write(&b, binary.BigEndian, int32(0))
	for _, row := range rows {
		_ = binary.Write(&b, binary.BigEndian, int16(len(row)))
		for _, field := range row {
			if field == nil {
				_ = binary.Write(&b, binary.BigEndian, int32(-1))
				continue
			}
			_ = binary.Write(&b, binary.BigEndian, int32(len(field)))
			b.Write(field)
		}
	}
	_ = binary.Write(&b, binary.BigEndian, int16(-1))
	return b.Bytes()
}

func TestCountPassesTheStreamThroughUnchanged(t *testing.T) {
	t.Parallel()

	stream := encode([][][]byte{
		{[]byte("shp_1"), nil, {}},
		{[]byte("shp_2"), []byte("TX"), []byte{1, 2, 3}},
	})
	var out bytes.Buffer
	rows, err := copybinary.Count(&out, bytes.NewReader(stream), 3)
	require.NoError(t, err)
	assert.Equal(t, int64(2), rows)
	assert.Equal(t, stream, out.Bytes())
}

func TestRewriteChangesOnlyWhatItIsAsked(t *testing.T) {
	t.Parallel()

	stream := encode([][][]byte{
		{[]byte("shp_1"), []byte("st_a"), []byte("keep")},
		{[]byte("shp_2"), nil, []byte("drop")},
	})
	var out bytes.Buffer
	rows, err := copybinary.Rewrite(
		&out,
		bytes.NewReader(stream),
		3,
		func(row *copybinary.Row) error {
			if !row.Null(1) {
				row.Set(1, []byte("TX"))
			}
			if string(row.Bytes(2)) == "drop" {
				row.SetNull(2)
			}
			return nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(2), rows)
	assert.Equal(t, encode([][][]byte{
		{[]byte("shp_1"), []byte("TX"), []byte("keep")},
		{[]byte("shp_2"), nil, nil},
	}), out.Bytes())
}

func TestAStreamOfAnotherShapeIsRefused(t *testing.T) {
	t.Parallel()

	stream := encode([][][]byte{{[]byte("a"), []byte("b")}})
	_, err := copybinary.Count(io.Discard, bytes.NewReader(stream), 3)
	require.ErrorIs(t, err, copybinary.ErrShape)
	_, err = copybinary.Count(io.Discard, bytes.NewReader(stream[:len(stream)-5]), 2)
	require.ErrorIs(t, err, copybinary.ErrShape)
	_, err = copybinary.Rewrite(io.Discard, bytes.NewReader([]byte("PGCOPY nope")), 2, nil)
	require.ErrorIs(t, err, copybinary.ErrShape)
	_, err = copybinary.Count(io.Discard, bytes.NewReader(append(bytes.Clone(stream), 0)), 2)
	require.ErrorIs(t, err, copybinary.ErrShape, "bytes after the trailer are refused")
}
