package archivecrypt

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	KeySize         = 32
	SegmentSize     = 1 << 20
	NoncePrefixSize = 7
	TagSize         = 16
	wrapNonceSize   = 12
	lastSegment     = byte(1)
)

var (
	ErrWrappingKey = errors.New("an archive key is 32 bytes")
	ErrUnwrap      = errors.New(
		"the archive's data key does not open with this archive key: the key is wrong or the manifest was altered",
	)
	ErrDataKey  = errors.New("an archive data key is 32 bytes")
	ErrTampered = errors.New(
		"an archive part failed authentication: it was changed, cut short or sealed with another key",
	)
	ErrTooLong   = errors.New("an archive part has more segments than its nonce can count")
	ErrTruncated = fmt.Errorf("%w: it ends before its last segment", ErrTampered)
)

type DataKey struct {
	aead cipher.AEAD
	raw  []byte
}

func NewDataKey() (*DataKey, error) {
	raw := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return nil, fmt.Errorf("make an archive data key: %w", err)
	}
	return DataKeyOf(raw)
}

func DataKeyOf(raw []byte) (*DataKey, error) {
	if len(raw) != KeySize {
		return nil, ErrDataKey
	}
	aead, err := gcm(raw)
	if err != nil {
		return nil, err
	}
	return &DataKey{aead: aead, raw: raw}, nil
}

func gcm(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("prepare an AES-256 key: %w", err)
	}
	return cipher.NewGCM(block)
}

func Wrap(key *DataKey, archiveKey []byte, instanceID string) (string, error) {
	if len(archiveKey) != KeySize {
		return "", ErrWrappingKey
	}
	aead, err := gcm(archiveKey)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, wrapNonceSize)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("make the wrap nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(
		aead.Seal(nonce, nonce, key.raw, []byte(instanceID)),
	), nil
}

func Unwrap(wrapped string, archiveKey []byte, instanceID string) (*DataKey, error) {
	if len(archiveKey) != KeySize {
		return nil, ErrWrappingKey
	}
	aead, err := gcm(archiveKey)
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(wrapped)
	if err != nil || len(raw) <= wrapNonceSize {
		return nil, ErrUnwrap
	}
	plain, err := aead.Open(nil, raw[:wrapNonceSize], raw[wrapNonceSize:], []byte(instanceID))
	if err != nil {
		return nil, ErrUnwrap
	}
	return DataKeyOf(plain)
}

func segmentNonce(prefix []byte, index uint64, last bool) []byte {
	nonce := make([]byte, 0, NoncePrefixSize+5)
	nonce = append(nonce, prefix...)
	counter := uint32(index) //nolint:gosec // bounded by ErrTooLong
	nonce = binary.BigEndian.AppendUint32(nonce, counter)
	if last {
		return append(nonce, lastSegment)
	}
	return append(nonce, 0)
}

type Writer struct {
	dst    io.Writer
	aead   cipher.AEAD
	aad    []byte
	prefix []byte
	plain  []byte
	sealed []byte
	index  uint64
	closed bool
}

func (k *DataKey) Encrypt(dst io.Writer, part string) (*Writer, error) {
	prefix := make([]byte, NoncePrefixSize)
	if _, err := io.ReadFull(rand.Reader, prefix); err != nil {
		return nil, fmt.Errorf("make the nonce prefix of %s: %w", part, err)
	}
	return k.EncryptWithPrefix(dst, part, prefix)
}

func (k *DataKey) EncryptWithPrefix(dst io.Writer, part string, prefix []byte) (*Writer, error) {
	if len(prefix) != NoncePrefixSize {
		return nil, fmt.Errorf("a nonce prefix is %d bytes", NoncePrefixSize)
	}
	if _, err := dst.Write(prefix); err != nil {
		return nil, err
	}
	return &Writer{
		dst:    dst,
		aead:   k.aead,
		aad:    []byte(part),
		prefix: append([]byte(nil), prefix...),
		plain:  make([]byte, 0, SegmentSize),
		sealed: make([]byte, 0, SegmentSize+TagSize),
	}, nil
}

func (w *Writer) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		if len(w.plain) == SegmentSize {
			if err := w.flush(false); err != nil {
				return written, err
			}
		}
		n := min(len(p), SegmentSize-len(w.plain))
		w.plain = append(w.plain, p[:n]...)
		p = p[n:]
		written += n
	}
	return written, nil
}

func (w *Writer) flush(last bool) error {
	if w.index > math.MaxUint32 {
		return ErrTooLong
	}
	w.sealed = w.aead.Seal(w.sealed[:0], segmentNonce(w.prefix, w.index, last), w.plain, w.aad)
	w.index++
	w.plain = w.plain[:0]
	_, err := w.dst.Write(w.sealed)
	return err
}

func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	return w.flush(true)
}

type Reader struct {
	src    *bufio.Reader
	aead   cipher.AEAD
	aad    []byte
	prefix []byte
	sealed []byte
	out    []byte
	offset int
	index  uint64
	done   bool
	err    error
}

func (k *DataKey) Decrypt(src io.Reader, part string) (*Reader, error) {
	r := &Reader{
		src:    bufio.NewReaderSize(src, SegmentSize+TagSize+1),
		aead:   k.aead,
		aad:    []byte(part),
		prefix: make([]byte, NoncePrefixSize),
		sealed: make([]byte, SegmentSize+TagSize),
	}
	if _, err := io.ReadFull(r.src, r.prefix); err != nil {
		return nil, ErrTruncated
	}
	return r, nil
}

func (r *Reader) Read(p []byte) (int, error) {
	for r.offset >= len(r.out) {
		if r.err != nil {
			return 0, r.err
		}
		if r.done {
			return 0, io.EOF
		}
		if err := r.next(); err != nil {
			r.err = err
			return 0, err
		}
	}
	n := copy(p, r.out[r.offset:])
	r.offset += n
	return n, nil
}

func (r *Reader) next() error {
	n, err := io.ReadFull(r.src, r.sealed)
	switch {
	case errors.Is(err, io.EOF):
		return ErrTruncated
	case errors.Is(err, io.ErrUnexpectedEOF):
	case err != nil:
		return err
	}
	last := n < len(r.sealed)
	if !last {
		if _, peekErr := r.src.Peek(1); errors.Is(peekErr, io.EOF) {
			last = true
		} else if peekErr != nil {
			return peekErr
		}
	}
	if n < TagSize || r.index > math.MaxUint32 {
		return ErrTruncated
	}
	out, openErr := r.aead.Open(
		r.out[:0],
		segmentNonce(r.prefix, r.index, last),
		r.sealed[:n],
		r.aad,
	)
	if openErr != nil {
		return ErrTampered
	}
	r.index++
	r.out, r.offset = out, 0
	r.done = last
	return nil
}

func PlainSize(stored int64) (int64, bool) {
	body := stored - NoncePrefixSize
	if body < TagSize {
		return 0, false
	}
	segments := (body + SegmentSize + TagSize - 1) / (SegmentSize + TagSize)
	return body - segments*TagSize, true
}
