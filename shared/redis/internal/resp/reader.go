package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

type Kind uint8

const (
	KindStatus Kind = iota + 1
	KindError
	KindInt
	KindBulk
	KindArray
	KindNil
)

const (
	maxBulkLen    = 512 << 20
	maxArrayLen   = 1 << 24
	maxDepth      = 64
	maxPrealloc   = 1024
	minBufferSize = 4096
)

var ErrProtocol = errors.New("redis: protocol error")

type Value struct {
	Kind  Kind
	Str   string
	Int   int64
	Elems []Value
}

type Error string

func (e Error) Error() string { return string(e) }

func (Error) RedisError() {}

type Reader struct {
	br *bufio.Reader
}

func NewReader(r io.Reader, size int) *Reader {
	return &Reader{br: bufio.NewReaderSize(r, max(size, minBufferSize))}
}

func (r *Reader) Read() (Value, error) {
	return r.read(0)
}

func (r *Reader) read(depth int) (Value, error) {
	line, err := r.line()
	if err != nil {
		return Value{}, err
	}

	switch line[0] {
	case '+':
		return Value{Kind: KindStatus, Str: string(line[1:])}, nil
	case '-':
		return Value{Kind: KindError, Str: string(line[1:])}, nil
	case ':':
		n, intErr := parseInt(line[1:])
		if intErr != nil {
			return Value{}, intErr
		}
		return Value{Kind: KindInt, Int: n}, nil
	case '$':
		return r.bulk(line[1:])
	case '*':
		return r.array(line[1:], depth)
	default:
		return Value{}, fmt.Errorf("%w: unexpected reply type %q", ErrProtocol, line[0])
	}
}

func (r *Reader) line() ([]byte, error) {
	line, err := r.br.ReadSlice('\n')
	if err != nil {
		if errors.Is(err, bufio.ErrBufferFull) {
			return nil, fmt.Errorf("%w: reply line too long", ErrProtocol)
		}
		return nil, err
	}
	n := len(line)
	if n < 3 || line[n-2] != '\r' {
		return nil, fmt.Errorf("%w: malformed reply line %q", ErrProtocol, line)
	}

	return line[:n-2], nil
}

func (r *Reader) bulk(header []byte) (Value, error) {
	n, err := parseInt(header)
	if err != nil {
		return Value{}, err
	}
	if n == -1 {
		return Value{Kind: KindNil}, nil
	}
	if n < 0 || n > maxBulkLen {
		return Value{}, fmt.Errorf("%w: bulk length %d", ErrProtocol, n)
	}

	buf := make([]byte, n+2)
	if _, err = io.ReadFull(r.br, buf); err != nil {
		return Value{}, unexpectedEOF(err)
	}
	if buf[n] != '\r' || buf[n+1] != '\n' {
		return Value{}, fmt.Errorf("%w: bulk string not terminated", ErrProtocol)
	}

	return Value{Kind: KindBulk, Str: string(buf[:n])}, nil
}

func (r *Reader) array(header []byte, depth int) (Value, error) {
	n, err := parseInt(header)
	if err != nil {
		return Value{}, err
	}
	if n == -1 {
		return Value{Kind: KindNil}, nil
	}
	if n < 0 || n > maxArrayLen {
		return Value{}, fmt.Errorf("%w: array length %d", ErrProtocol, n)
	}
	if depth >= maxDepth {
		return Value{}, fmt.Errorf("%w: reply nested too deeply", ErrProtocol)
	}

	elems := make([]Value, 0, min(int(n), maxPrealloc))
	for range n {
		elem, elemErr := r.read(depth + 1)
		if elemErr != nil {
			return Value{}, unexpectedEOF(elemErr)
		}
		elems = append(elems, elem)
	}

	return Value{Kind: KindArray, Elems: elems}, nil
}

func unexpectedEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}

func parseInt(b []byte) (int64, error) {
	if len(b) == 0 {
		return 0, fmt.Errorf("%w: empty integer", ErrProtocol)
	}
	neg := b[0] == '-'
	if neg {
		b = b[1:]
		if len(b) == 0 {
			return 0, fmt.Errorf("%w: malformed integer", ErrProtocol)
		}
	}

	var n uint64
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("%w: malformed integer %q", ErrProtocol, b)
		}
		if n > (1<<63)/10 {
			return 0, fmt.Errorf("%w: integer overflows int64", ErrProtocol)
		}
		n = n*10 + uint64(c-'0')
		if n > 1<<63 {
			return 0, fmt.Errorf("%w: integer overflows int64", ErrProtocol)
		}
	}
	if neg {
		return -int64(n), nil
	}
	if n == 1<<63 {
		return 0, fmt.Errorf("%w: integer overflows int64", ErrProtocol)
	}

	return int64(n), nil
}
