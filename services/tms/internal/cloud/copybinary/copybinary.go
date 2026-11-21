package copybinary

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	bufferSize      = 256 << 10
	MaxRow          = 64 << 20
	maxHeaderExtend = 1 << 20
	nullLength      = -1
)

var (
	Signature = []byte("PGCOPY\n\xff\r\n\x00")

	ErrRowTooLarge = fmt.Errorf("a row larger than %d bytes cannot be rewritten", MaxRow)
	ErrShape       = errors.New(
		"the rows are not PostgreSQL binary COPY with the manifest's columns",
	)
)

type Row struct {
	Fields [][]byte
	null   []bool
	size   int
}

func (r *Row) Null(i int) bool {
	return r.null[i]
}

func (r *Row) Bytes(i int) []byte {
	if r.null[i] {
		return nil
	}
	return r.Fields[i]
}

func (r *Row) Set(i int, value []byte) {
	r.Fields[i] = value
	r.null[i] = false
}

func (r *Row) SetNull(i int) {
	r.Fields[i] = nil
	r.null[i] = true
}

type stream struct {
	in  *bufio.Reader
	out *bufio.Writer
}

func newStream(dst io.Writer, src io.Reader) *stream {
	return &stream{
		in:  bufio.NewReaderSize(src, bufferSize),
		out: bufio.NewWriterSize(dst, bufferSize),
	}
}

func (s *stream) header() error {
	head := make([]byte, len(Signature)+8)
	if _, err := io.ReadFull(s.in, head); err != nil {
		return ErrShape
	}
	if !bytes.Equal(head[:len(Signature)], Signature) {
		return ErrShape
	}
	extension := binary.BigEndian.Uint32(head[len(Signature)+4:])
	if extension > maxHeaderExtend {
		return ErrShape
	}
	if _, err := s.out.Write(head); err != nil {
		return err
	}
	_, err := io.CopyN(s.out, s.in, int64(extension))
	if err != nil {
		return ErrShape
	}
	return nil
}

func (s *stream) fieldCount(columns int) (bool, error) {
	var raw [2]byte
	if _, err := io.ReadFull(s.in, raw[:]); err != nil {
		return false, ErrShape
	}
	unsigned := binary.BigEndian.Uint16(raw[:])
	count := int16(unsigned) //nolint:gosec // the wire format is a signed int16
	if count == -1 {
		if _, err := s.out.Write(raw[:]); err != nil {
			return false, err
		}
		if _, err := s.in.ReadByte(); !errors.Is(err, io.EOF) {
			return false, ErrShape
		}
		return false, nil
	}
	if int(count) != columns {
		return false, ErrShape
	}
	return true, nil
}

func fieldLength(raw []byte) int32 {
	return int32(binary.BigEndian.Uint32(raw)) //nolint:gosec // the wire format is a signed int32
}

func Count(dst io.Writer, src io.Reader, columns int) (int64, error) {
	s := newStream(dst, src)
	if err := s.header(); err != nil {
		return 0, err
	}
	var rows int64
	var raw [4]byte
	countRaw := make([]byte, 2)
	binary.BigEndian.PutUint16(countRaw, uint16(columns)) //nolint:gosec // bounded by the catalog
	for {
		more, err := s.fieldCount(columns)
		if err != nil {
			return rows, err
		}
		if !more {
			return rows, s.out.Flush()
		}
		if _, err = s.out.Write(countRaw); err != nil {
			return rows, err
		}
		for range columns {
			if _, err = io.ReadFull(s.in, raw[:]); err != nil {
				return rows, ErrShape
			}
			if _, err = s.out.Write(raw[:]); err != nil {
				return rows, err
			}
			length := fieldLength(raw[:])
			if length == nullLength {
				continue
			}
			if length < 0 {
				return rows, ErrShape
			}
			if _, err = io.CopyN(s.out, s.in, int64(length)); err != nil {
				return rows, ErrShape
			}
		}
		rows++
	}
}

func Rewrite(dst io.Writer, src io.Reader, columns int, fn func(*Row) error) (int64, error) {
	s := newStream(dst, src)
	if err := s.header(); err != nil {
		return 0, err
	}
	row := &Row{Fields: make([][]byte, columns), null: make([]bool, columns)}
	var rows int64
	for {
		more, err := s.fieldCount(columns)
		if err != nil {
			return rows, err
		}
		if !more {
			return rows, s.out.Flush()
		}
		if err = s.readRow(row); err != nil {
			return rows, err
		}
		if fn != nil {
			if err = fn(row); err != nil {
				return rows, err
			}
		}
		if err = s.writeRow(row); err != nil {
			return rows, err
		}
		rows++
	}
}

func (s *stream) readRow(row *Row) error {
	row.size = 0
	var raw [4]byte
	for i := range row.Fields {
		if _, err := io.ReadFull(s.in, raw[:]); err != nil {
			return ErrShape
		}
		length := fieldLength(raw[:])
		if length == nullLength {
			row.Fields[i], row.null[i] = nil, true
			continue
		}
		if length < 0 {
			return ErrShape
		}
		row.size += int(length)
		if row.size > MaxRow {
			return ErrRowTooLarge
		}
		field := make([]byte, length)
		if _, err := io.ReadFull(s.in, field); err != nil {
			return ErrShape
		}
		row.Fields[i], row.null[i] = field, false
	}
	return nil
}

func (s *stream) writeRow(row *Row) error {
	var raw [4]byte
	fields := uint16(len(row.Fields)) //nolint:gosec // bounded by the catalog
	binary.BigEndian.PutUint16(raw[:2], fields)
	if _, err := s.out.Write(raw[:2]); err != nil {
		return err
	}
	for i, field := range row.Fields {
		if row.null[i] {
			binary.BigEndian.PutUint32(raw[:], uint32(0xFFFFFFFF))
			if _, err := s.out.Write(raw[:]); err != nil {
				return err
			}
			continue
		}
		binary.BigEndian.PutUint32(raw[:], uint32(len(field))) //nolint:gosec // bounded by MaxRow
		if _, err := s.out.Write(raw[:]); err != nil {
			return err
		}
		if _, err := s.out.Write(field); err != nil {
			return err
		}
	}
	return nil
}
