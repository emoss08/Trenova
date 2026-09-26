package aitrainingservice

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/infrastructure/storage/uploadpipe"
	"go.uber.org/zap"
)

const (
	jsonLinesContentType = "application/x-ndjson"
	partBufferSize       = 64 * 1024
	metadataExport       = "trenova-training-export"
	metadataSplit        = "trenova-training-split"
)

type partWriter struct {
	part   aitraining.Part
	pipe   *uploadpipe.Pipe
	digest hash.Hash
	buffer *bufio.Writer
}

func (w *partWriter) write(line []byte) error {
	if _, err := w.buffer.Write(line); err != nil {
		return fmt.Errorf("write training example: %w", err)
	}
	if err := w.buffer.WriteByte('\n'); err != nil {
		return fmt.Errorf("write training example: %w", err)
	}
	w.part.Examples++

	return nil
}

func (w *partWriter) close() (aitraining.Part, error) {
	if err := w.buffer.Flush(); err != nil {
		w.pipe.Abort(err)
		return aitraining.Part{}, fmt.Errorf("flush training part: %w", err)
	}
	size, err := w.pipe.Close()
	if err != nil {
		return aitraining.Part{}, fmt.Errorf("upload training part: %w", err)
	}
	w.part.Bytes = size
	w.part.SHA256 = hex.EncodeToString(w.digest.Sum(nil))

	return w.part, nil
}

type partSet struct {
	ctx     context.Context
	storage storage.Client
	export  *aitraining.TrainingExport
	ordinal int
	writers map[aitraining.Split]*partWriter
	written []string
	l       *zap.Logger
}

func newPartSet(
	ctx context.Context,
	client storage.Client,
	export *aitraining.TrainingExport,
	ordinal int,
	l *zap.Logger,
) *partSet {
	return &partSet{
		ctx:     ctx,
		storage: client,
		export:  export,
		ordinal: ordinal,
		writers: map[aitraining.Split]*partWriter{},
		l:       l,
	}
}

func (s *partSet) write(split aitraining.Split, line []byte) error {
	writer, ok := s.writers[split]
	if !ok {
		key := s.export.PartKey(s.ordinal, split)
		pipe := uploadpipe.Open(s.ctx, s.storage, uploadpipe.Params{
			Key:         key,
			ContentType: jsonLinesContentType,
			Metadata: map[string]string{
				metadataExport: s.export.ID.String(),
				metadataSplit:  split.String(),
			},
		})
		digest := sha256.New()
		writer = &partWriter{
			part:   aitraining.Part{Ordinal: s.ordinal, Split: split, Key: key},
			pipe:   pipe,
			digest: digest,
			buffer: bufio.NewWriterSize(io.MultiWriter(pipe.Writer(), digest), partBufferSize),
		}
		s.writers[split] = writer
		s.written = append(s.written, key)
	}

	return writer.write(line)
}

func (s *partSet) close() ([]aitraining.Part, error) {
	parts := make([]aitraining.Part, 0, len(s.writers))
	var firstErr error
	for _, split := range []aitraining.Split{aitraining.SplitTrain, aitraining.SplitValidation} {
		writer, ok := s.writers[split]
		if !ok {
			continue
		}
		delete(s.writers, split)
		part, err := writer.close()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		parts = append(parts, part)
	}

	return parts, firstErr
}

func (s *partSet) discard(ctx context.Context, cause error) {
	for split, writer := range s.writers {
		writer.pipe.Abort(cause)
		delete(s.writers, split)
	}
	for _, key := range s.written {
		if err := s.storage.Delete(ctx, key); err != nil {
			s.l.Warn("failed to delete discarded training part", zap.String("key", key), zap.Error(err))
		}
	}
	s.written = nil
}
