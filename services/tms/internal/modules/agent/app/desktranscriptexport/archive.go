package desktranscriptexport

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
)

const (
	recordsFileName = "conversations.jsonl"
	recordsTempName = "desk-transcripts-*.jsonl"
)

type countingWriter struct {
	inner io.Writer
	hash  hash.Hash
	bytes int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n, err := w.inner.Write(p)
	if n > 0 {
		w.hash.Write(p[:n])
		w.bytes += int64(n)
	}

	return n, err
}

type ArchiveStats struct {
	Conversations int
	Bytes         int64
	SHA256        string
}

type archive struct {
	out     *countingWriter
	zip     *zip.Writer
	records *os.File
	buffer  *bufio.Writer
	names   map[string]int
	written int
	now     time.Time
}

func openArchive(w io.Writer, now time.Time) (*archive, error) {
	records, err := os.CreateTemp("", recordsTempName)
	if err != nil {
		return nil, fmt.Errorf("open the conversations file: %w", err)
	}
	out := &countingWriter{inner: w, hash: sha256.New()}

	return &archive{
		out:     out,
		zip:     zip.NewWriter(out),
		records: records,
		buffer:  bufio.NewWriter(records),
		names:   make(map[string]int),
		now:     now,
	}, nil
}

func (a *archive) entryName(folder, name string) string {
	base := folder + "/" + name
	seen := a.names[strings.ToLower(base)]
	a.names[strings.ToLower(base)] = seen + 1
	if seen == 0 {
		return base
	}

	stem := strings.TrimSuffix(name, ".md")

	return folder + "/" + stem + " (" + strconv.Itoa(seen+1) + ").md"
}

func (a *archive) add(rendered *desktranscriptports.RenderedConversation) error {
	entry, err := a.zip.CreateHeader(&zip.FileHeader{
		Name:     a.entryName(rendered.ArchiveFolder, rendered.ArchiveName),
		Method:   zip.Deflate,
		Modified: a.now,
	})
	if err != nil {
		return fmt.Errorf("add a transcript to the archive: %w", err)
	}
	if _, err = io.WriteString(entry, rendered.Markdown); err != nil {
		return fmt.Errorf("write a transcript to the archive: %w", err)
	}

	line, err := sonic.Marshal(&rendered.Record)
	if err != nil {
		return fmt.Errorf("encode a conversation record: %w", err)
	}
	if _, err = a.buffer.Write(line); err != nil {
		return fmt.Errorf("keep a conversation record: %w", err)
	}
	if err = a.buffer.WriteByte('\n'); err != nil {
		return fmt.Errorf("keep a conversation record: %w", err)
	}
	a.written++

	return nil
}

func (a *archive) finish() (*ArchiveStats, error) {
	if err := a.buffer.Flush(); err != nil {
		return nil, fmt.Errorf("keep the conversation records: %w", err)
	}
	if _, err := a.records.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("read back the conversation records: %w", err)
	}
	entry, err := a.zip.CreateHeader(&zip.FileHeader{
		Name:     recordsFileName,
		Method:   zip.Deflate,
		Modified: a.now,
	})
	if err != nil {
		return nil, fmt.Errorf("add the conversation records to the archive: %w", err)
	}
	if _, err = io.Copy(entry, a.records); err != nil {
		return nil, fmt.Errorf("write the conversation records to the archive: %w", err)
	}
	if err = a.zip.Close(); err != nil {
		return nil, fmt.Errorf("close the archive: %w", err)
	}

	return &ArchiveStats{
		Conversations: a.written,
		Bytes:         a.out.bytes,
		SHA256:        hex.EncodeToString(a.out.hash.Sum(nil)),
	}, nil
}

func (a *archive) close() error {
	name := a.records.Name()

	return errors.Join(a.records.Close(), os.Remove(name))
}
