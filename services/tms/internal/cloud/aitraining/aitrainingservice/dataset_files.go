package aitrainingservice

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

const datasetBufferSize = 256 * 1024

type countingWriter struct {
	bytes int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.bytes += int64(len(p))
	return len(p), nil
}

type datasetFile struct {
	name    string
	closer  io.Closer
	digest  hash.Hash
	counter *countingWriter
	buffer  *bufio.Writer
	records int
}

func createDatasetFile(sink services.TrainingDatasetSink, name string) (*datasetFile, error) {
	target, err := sink.Create(name)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", name, err)
	}
	digest := sha256.New()
	counter := &countingWriter{}

	return &datasetFile{
		name:    name,
		closer:  target,
		digest:  digest,
		counter: counter,
		buffer:  bufio.NewWriterSize(io.MultiWriter(target, digest, counter), datasetBufferSize),
	}, nil
}

func (f *datasetFile) writeRecord(record any) error {
	encoded, err := sonic.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode %s record: %w", f.name, err)
	}
	if _, err = f.buffer.Write(encoded); err != nil {
		return fmt.Errorf("write %s: %w", f.name, err)
	}
	if err = f.buffer.WriteByte('\n'); err != nil {
		return fmt.Errorf("write %s: %w", f.name, err)
	}
	f.records++

	return nil
}

func (f *datasetFile) writeDocument(document any) error {
	encoded, err := sonic.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", f.name, err)
	}
	if _, err = f.buffer.Write(encoded); err != nil {
		return fmt.Errorf("write %s: %w", f.name, err)
	}

	return f.buffer.WriteByte('\n')
}

func (f *datasetFile) close() (aitraining.DatasetFile, error) {
	flushErr := f.buffer.Flush()
	closeErr := f.closer.Close()
	if flushErr != nil {
		return aitraining.DatasetFile{}, fmt.Errorf("flush %s: %w", f.name, flushErr)
	}
	if closeErr != nil {
		return aitraining.DatasetFile{}, fmt.Errorf("close %s: %w", f.name, closeErr)
	}

	return aitraining.DatasetFile{
		Name:    f.name,
		Records: f.records,
		Bytes:   f.counter.bytes,
		SHA256:  hex.EncodeToString(f.digest.Sum(nil)),
	}, nil
}

type datasetFiles struct {
	order []*datasetFile
	byKey map[string]*datasetFile
}

func openDatasetFiles(sink services.TrainingDatasetSink, names ...string) (*datasetFiles, error) {
	files := &datasetFiles{
		order: make([]*datasetFile, 0, len(names)),
		byKey: make(map[string]*datasetFile, len(names)),
	}
	for _, name := range names {
		file, err := createDatasetFile(sink, name)
		if err != nil {
			files.abandon()
			return nil, err
		}
		files.order = append(files.order, file)
		files.byKey[name] = file
	}

	return files, nil
}

func (d *datasetFiles) write(name string, record any) error {
	return d.byKey[name].writeRecord(record)
}

func (d *datasetFiles) close() ([]aitraining.DatasetFile, error) {
	out := make([]aitraining.DatasetFile, 0, len(d.order))
	var firstErr error
	for _, file := range d.order {
		closed, err := file.close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		out = append(out, closed)
	}

	return out, firstErr
}

func (d *datasetFiles) abandon() {
	for _, file := range d.order {
		_ = file.closer.Close()
	}
}
