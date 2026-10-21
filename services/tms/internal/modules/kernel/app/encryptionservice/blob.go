package encryptionservice

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	CryptoModeBlobV1 = "blob_v1"
	BlobChunkSize    = 64 << 10

	blobMagic           = "TRNBLOB"
	blobVersion         = byte(1)
	blobNoncePrefixSize = 7
	blobTagSize         = 16
	blobFixedSize       = len(blobMagic) + 1 + 4 + blobNoncePrefixSize + sha256.Size
	blobKeyFieldMax     = 2048
	blobKeySectionMax   = 5 * (2 + blobKeyFieldMax)
	blobLastChunk       = byte(1)
)

var (
	ErrBlobRejected = errors.New("the blob could not be decrypted or authenticated")
	ErrBlobFormat   = fmt.Errorf("%w: the object is not a sealed blob", ErrBlobRejected)
	ErrBlobAAD      = fmt.Errorf("%w: it is sealed for other associated data", ErrBlobRejected)
	ErrBlobTampered = fmt.Errorf("%w: it failed authentication", ErrBlobRejected)
	ErrBlobSize     = errors.New("the blob's plaintext is not the declared size")
)

type blobHeader struct {
	fixed       []byte
	noncePrefix []byte
	aadHash     []byte
	wrapped     WrappedKey
	length      int64
}

func IsBlob(prefix []byte) bool {
	return len(prefix) > len(blobMagic) &&
		string(prefix[:len(blobMagic)]) == blobMagic &&
		prefix[len(blobMagic)] == blobVersion
}

func BlobPrefixSize() int {
	return len(blobMagic) + 1
}

func SealedBlobSize(headerSize, plaintextSize int64) int64 {
	return headerSize + plaintextSize + blobChunkCount(plaintextSize)*blobTagSize
}

func blobChunkCount(plaintextSize int64) int64 {
	if plaintextSize <= 0 {
		return 1
	}
	return (plaintextSize + BlobChunkSize - 1) / BlobChunkSize
}

func blobPlaintextSize(headerSize, sealedSize int64) int64 {
	body := sealedSize - headerSize
	if body < blobTagSize {
		return -1
	}
	chunks := (body + BlobChunkSize + blobTagSize - 1) / (BlobChunkSize + blobTagSize)
	return body - chunks*blobTagSize
}

func blobAADHash(aad AAD) []byte {
	sum := sha256.Sum256(aad.Bytes())
	return sum[:]
}

func newBlobAEAD(dek []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func chunkNonce(prefix []byte, counter uint32, last bool) []byte {
	nonce := make([]byte, 0, blobNoncePrefixSize+5)
	nonce = append(nonce, prefix...)
	nonce = binary.BigEndian.AppendUint32(nonce, counter)
	if last {
		return append(nonce, blobLastChunk)
	}
	return append(nonce, 0)
}

func encodeBlobHeader(fixed []byte, wrapped *WrappedKey) ([]byte, error) {
	fields := [][]byte{
		[]byte(wrapped.Provider),
		[]byte(wrapped.KeyID),
		[]byte(wrapped.Algorithm),
		wrapped.Nonce,
		wrapped.Bytes,
	}
	section := make([]byte, 0, blobKeySectionMax)
	for _, field := range fields {
		if len(field) > blobKeyFieldMax {
			return nil, fmt.Errorf("%w: a wrapped key field is %d bytes", ErrBlobFormat, len(field))
		}
		fieldSize := uint16(len(field)) //nolint:gosec // bounded above
		section = binary.BigEndian.AppendUint16(section, fieldSize)
		section = append(section, field...)
	}

	header := make([]byte, 0, len(fixed)+2+len(section))
	header = append(header, fixed...)
	sectionSize := uint16(len(section)) //nolint:gosec // bounded by the fields
	header = binary.BigEndian.AppendUint16(header, sectionSize)
	return append(header, section...), nil
}

func headerReadError(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("%w: %w", ErrBlobFormat, err)
	}
	return err
}

func readBlobHeader(src io.Reader) (*blobHeader, error) {
	fixed := make([]byte, blobFixedSize)
	if _, err := io.ReadFull(src, fixed); err != nil {
		return nil, headerReadError(err)
	}
	if !IsBlob(fixed) {
		return nil, ErrBlobFormat
	}
	offset := len(blobMagic) + 1
	if binary.BigEndian.Uint32(fixed[offset:]) != BlobChunkSize {
		return nil, fmt.Errorf("%w: unsupported chunk size", ErrBlobFormat)
	}
	offset += 4
	header := &blobHeader{
		fixed:       fixed,
		noncePrefix: fixed[offset : offset+blobNoncePrefixSize],
		aadHash:     fixed[offset+blobNoncePrefixSize:],
	}

	var sizeField [2]byte
	if _, err := io.ReadFull(src, sizeField[:]); err != nil {
		return nil, headerReadError(err)
	}
	sectionSize := int(binary.BigEndian.Uint16(sizeField[:]))
	if sectionSize > blobKeySectionMax {
		return nil, fmt.Errorf("%w: the key section is %d bytes", ErrBlobFormat, sectionSize)
	}
	section := make([]byte, sectionSize)
	if _, err := io.ReadFull(src, section); err != nil {
		return nil, headerReadError(err)
	}

	fields := make([][]byte, 0, 5)
	for range 5 {
		if len(section) < 2 {
			return nil, ErrBlobFormat
		}
		size := int(binary.BigEndian.Uint16(section))
		section = section[2:]
		if size > len(section) {
			return nil, ErrBlobFormat
		}
		fields = append(fields, section[:size])
		section = section[size:]
	}
	if len(section) != 0 || len(fields[0]) == 0 || len(fields[1]) == 0 || len(fields[4]) == 0 {
		return nil, ErrBlobFormat
	}
	header.wrapped = WrappedKey{
		Provider:  string(fields[0]),
		KeyID:     string(fields[1]),
		Algorithm: string(fields[2]),
		Nonce:     fields[3],
		Bytes:     fields[4],
	}
	header.length = int64(blobFixedSize + 2 + sectionSize)
	return header, nil
}

type SealedBlob struct {
	src       io.Reader
	aead      cipher.AEAD
	chunkAAD  []byte
	prefix    []byte
	plain     []byte
	out       []byte
	offset    int
	counter   uint32
	remaining int64
	size      int64
	done      bool
	err       error
}

func (s *Service) SealBlob(
	ctx context.Context,
	src io.Reader,
	plaintextSize int64,
	aad AAD,
) (*SealedBlob, error) {
	if s.keyManager == nil {
		return nil, ErrKeyManagerDisabled
	}
	if plaintextSize < 0 || blobChunkCount(plaintextSize) > math.MaxUint32 {
		return nil, fmt.Errorf("%w: %d bytes", ErrBlobSize, plaintextSize)
	}

	dek := make([]byte, dekSize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, err
	}
	aead, err := newBlobAEAD(dek)
	if err != nil {
		return nil, err
	}
	wrapped, err := s.keyManager.WrapKey(ctx, dek, aad)
	if err != nil {
		return nil, err
	}

	prefix := make([]byte, blobNoncePrefixSize)
	if _, err = io.ReadFull(rand.Reader, prefix); err != nil {
		return nil, err
	}
	fixed := make([]byte, 0, blobFixedSize)
	fixed = append(fixed, blobMagic...)
	fixed = append(fixed, blobVersion)
	fixed = binary.BigEndian.AppendUint32(fixed, BlobChunkSize)
	fixed = append(fixed, prefix...)
	fixed = append(fixed, blobAADHash(aad)...)

	header, err := encodeBlobHeader(fixed, wrapped)
	if err != nil {
		return nil, err
	}

	return &SealedBlob{
		src:       src,
		aead:      aead,
		chunkAAD:  append(bytes.Clone(fixed), aad.Bytes()...),
		prefix:    prefix,
		plain:     make([]byte, BlobChunkSize),
		out:       header,
		remaining: plaintextSize,
		size:      SealedBlobSize(int64(len(header)), plaintextSize),
	}, nil
}

func (b *SealedBlob) Size() int64 {
	return b.size
}

func (b *SealedBlob) Read(p []byte) (int, error) {
	for b.offset >= len(b.out) {
		if b.err != nil {
			return 0, b.err
		}
		if b.done {
			return 0, io.EOF
		}
		if err := b.sealNext(); err != nil {
			b.err = err
			return 0, err
		}
	}
	n := copy(p, b.out[b.offset:])
	b.offset += n
	return n, nil
}

func (b *SealedBlob) sealNext() error {
	size := int(min(b.remaining, BlobChunkSize))
	if _, err := io.ReadFull(b.src, b.plain[:size]); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: the source ended early", ErrBlobSize)
		}
		return err
	}
	b.remaining -= int64(size)
	last := b.remaining == 0
	if last {
		var extra [1]byte
		n, err := b.src.Read(extra[:])
		for n == 0 && err == nil {
			n, err = b.src.Read(extra[:])
		}
		if n > 0 {
			return fmt.Errorf("%w: the source is longer than declared", ErrBlobSize)
		}
		if !errors.Is(err, io.EOF) {
			return err
		}
	}

	nonce := chunkNonce(b.prefix, b.counter, last)
	b.out = b.aead.Seal(b.out[:0], nonce, b.plain[:size], b.chunkAAD)
	b.offset = 0
	b.counter++
	b.done = last
	return nil
}

type OpenedBlob struct {
	src      io.Reader
	aead     cipher.AEAD
	chunkAAD []byte
	prefix   []byte
	sealed   []byte
	have     int
	out      []byte
	offset   int
	counter  uint32
	size     int64
	done     bool
	err      error
}

func (s *Service) OpenBlob(
	ctx context.Context,
	src io.Reader,
	sealedSize int64,
	aad AAD,
) (*OpenedBlob, error) {
	if s.keyManager == nil {
		return nil, ErrKeyManagerDisabled
	}
	header, err := readBlobHeader(src)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare(header.aadHash, blobAADHash(aad)) != 1 {
		return nil, ErrBlobAAD
	}
	dek, err := s.keyManager.UnwrapKey(ctx, header.wrapped, aad)
	if err != nil {
		return nil, fmt.Errorf("%w: its key did not unwrap: %w", ErrBlobRejected, err)
	}
	aead, err := newBlobAEAD(dek)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBlobRejected, err)
	}

	size := int64(-1)
	if sealedSize >= 0 {
		size = blobPlaintextSize(header.length, sealedSize)
		if size < 0 {
			return nil, fmt.Errorf("%w: the object is too short", ErrBlobTampered)
		}
	}
	return &OpenedBlob{
		src:      src,
		aead:     aead,
		chunkAAD: append(bytes.Clone(header.fixed), aad.Bytes()...),
		prefix:   bytes.Clone(header.noncePrefix),
		sealed:   make([]byte, BlobChunkSize+blobTagSize+1),
		size:     size,
	}, nil
}

func (b *OpenedBlob) Size() int64 {
	return b.size
}

func (b *OpenedBlob) Read(p []byte) (int, error) {
	for b.offset >= len(b.out) {
		if b.err != nil {
			return 0, b.err
		}
		if b.done {
			return 0, io.EOF
		}
		if err := b.openNext(); err != nil {
			b.err = err
			return 0, err
		}
	}
	n := copy(p, b.out[b.offset:])
	b.offset += n
	return n, nil
}

func (b *OpenedBlob) openNext() error {
	n, err := io.ReadFull(b.src, b.sealed[b.have:])
	b.have += n
	last := false
	switch {
	case err == nil:
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		last = true
	default:
		return err
	}

	chunk := b.sealed[:min(b.have, BlobChunkSize+blobTagSize)]
	if len(chunk) < blobTagSize {
		return fmt.Errorf("%w: the object ends inside a chunk", ErrBlobTampered)
	}
	if b.counter == math.MaxUint32 && !last {
		return fmt.Errorf("%w: too many chunks", ErrBlobTampered)
	}
	opened, openErr := b.aead.Open(
		b.out[:0],
		chunkNonce(b.prefix, b.counter, last),
		chunk,
		b.chunkAAD,
	)
	if openErr != nil {
		return ErrBlobTampered
	}

	b.out = opened
	b.offset = 0
	b.counter++
	b.done = last
	if !last {
		b.sealed[0] = b.sealed[len(b.sealed)-1]
		b.have = 1
	}
	return nil
}

type RewrappedBlob struct {
	Header   []byte
	Consumed int64
}

func (s *Service) RewrapBlob(
	ctx context.Context,
	src io.Reader,
	aads []AAD,
) (rewrapped *RewrappedBlob, changed bool, err error) {
	if s.keyManager == nil {
		return nil, false, ErrKeyManagerDisabled
	}
	header, err := readBlobHeader(src)
	if err != nil {
		return nil, false, err
	}

	var aad *AAD
	for i := range aads {
		if subtle.ConstantTimeCompare(header.aadHash, blobAADHash(aads[i])) == 1 {
			aad = &aads[i]
			break
		}
	}
	if aad == nil {
		return nil, false, ErrBlobAAD
	}
	if header.wrapped.Provider == s.keyManager.Provider() &&
		header.wrapped.KeyID == s.keyManager.ActiveKeyID() {
		return &RewrappedBlob{Consumed: header.length}, false, nil
	}

	dek, err := s.keyManager.UnwrapKey(ctx, header.wrapped, *aad)
	if err != nil {
		return nil, false, err
	}
	next, err := s.keyManager.WrapKey(ctx, dek, *aad)
	if err != nil {
		return nil, false, err
	}
	encoded, err := encodeBlobHeader(header.fixed, next)
	if err != nil {
		return nil, false, err
	}
	return &RewrappedBlob{Header: encoded, Consumed: header.length}, true, nil
}
