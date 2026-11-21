package archivestream

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"

	"github.com/emoss08/trenova/internal/cloud/archivecrypt"
	"github.com/klauspost/compress/zstd"
)

const (
	EncoderWindow = 4 << 20
	DecoderWindow = 8 << 20
)

var ErrStoredBytesChanged = errors.New(
	"the part's stored bytes do not match the manifest's SHA-256: the archive was changed or cut short",
)

type Options struct {
	Part     string
	Key      *archivecrypt.DataKey
	Compress bool
}

type Sum struct {
	Bytes  int64
	SHA256 string
}

type hashingWriter struct {
	dst   io.Writer
	hash  hash.Hash
	bytes int64
}

func (w *hashingWriter) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)
	w.hash.Write(p[:n])
	w.bytes += int64(n)
	return n, err
}

type Writer struct {
	head    io.Writer
	encoder *zstd.Encoder
	cipher  io.WriteCloser
	stored  *hashingWriter
	plain   int64
	closed  bool
}

func NewWriter(dst io.Writer, opts *Options) (*Writer, error) {
	w := &Writer{stored: &hashingWriter{dst: dst, hash: sha256.New()}}
	var next io.Writer = w.stored
	if opts.Key != nil {
		enc, err := opts.Key.Encrypt(w.stored, opts.Part)
		if err != nil {
			return nil, err
		}
		w.cipher = enc
		next = enc
	}
	if opts.Compress {
		encoder, err := zstd.NewWriter(next,
			zstd.WithEncoderLevel(zstd.SpeedDefault),
			zstd.WithEncoderConcurrency(1),
			zstd.WithWindowSize(EncoderWindow),
			zstd.WithLowerEncoderMem(true),
		)
		if err != nil {
			return nil, fmt.Errorf("start compressing %s: %w", opts.Part, err)
		}
		w.encoder = encoder
		next = encoder
	}
	w.head = next
	return w, nil
}

func (w *Writer) Write(p []byte) (int, error) {
	n, err := w.head.Write(p)
	w.plain += int64(n)
	return n, err
}

func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if w.encoder != nil {
		if err := w.encoder.Close(); err != nil {
			return fmt.Errorf("finish compressing: %w", err)
		}
	}
	if w.cipher != nil {
		if err := w.cipher.Close(); err != nil {
			return fmt.Errorf("finish encrypting: %w", err)
		}
	}
	return nil
}

func (w *Writer) Plain() int64 {
	return w.plain
}

func (w *Writer) Sum() Sum {
	return Sum{Bytes: w.stored.bytes, SHA256: hex.EncodeToString(w.stored.hash.Sum(nil))}
}

type hashingReader struct {
	src    io.Reader
	hash   hash.Hash
	bytes  int64
	expect Sum
}

func (r *hashingReader) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)
	r.hash.Write(p[:n])
	r.bytes += int64(n)
	if errors.Is(err, io.EOF) && !r.matches() {
		return n, ErrStoredBytesChanged
	}
	return n, err
}

func (r *hashingReader) matches() bool {
	return r.bytes == r.expect.Bytes && hex.EncodeToString(r.hash.Sum(nil)) == r.expect.SHA256
}

type Reader struct {
	head    io.Reader
	decoder *zstd.Decoder
	stored  *hashingReader
}

func NewReader(src io.Reader, expect Sum, opts *Options) (*Reader, error) {
	r := &Reader{stored: &hashingReader{src: src, hash: sha256.New(), expect: expect}}
	var next io.Reader = r.stored
	if opts.Key != nil {
		dec, err := opts.Key.Decrypt(r.stored, opts.Part)
		if err != nil {
			return nil, err
		}
		next = dec
	}
	if opts.Compress {
		decoder, err := zstd.NewReader(next,
			zstd.WithDecoderConcurrency(1),
			zstd.WithDecoderLowmem(true),
			zstd.WithDecoderMaxWindow(DecoderWindow),
		)
		if err != nil {
			return nil, fmt.Errorf("start decompressing %s: %w", opts.Part, err)
		}
		r.decoder = decoder
		next = decoder
	}
	r.head = next
	return r, nil
}

func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.head.Read(p)
	if errors.Is(err, io.EOF) {
		if drainErr := r.drain(); drainErr != nil {
			return n, drainErr
		}
	}
	return n, err
}

func (r *Reader) drain() error {
	if _, err := io.Copy(io.Discard, r.stored); err != nil {
		return err
	}
	if !r.stored.matches() {
		return ErrStoredBytesChanged
	}
	return nil
}

func (r *Reader) Close() {
	if r.decoder != nil {
		r.decoder.Close()
	}
}

func Verify(src io.Reader, expect Sum) error {
	r := &hashingReader{src: src, hash: sha256.New(), expect: expect}
	if _, err := io.Copy(io.Discard, r); err != nil {
		return err
	}
	if !r.matches() {
		return ErrStoredBytesChanged
	}
	return nil
}
