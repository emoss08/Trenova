package logs

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
)

const (
	CutComponent = "component cap"
	CutBundle    = "bundle cap"

	manifestName = "manifest.json"
	lineFormat   = "timestamp pod container message, oldest first; a line break inside a message is written as \\n"
	redaction    = "every value of every Secret in the instance's namespace (as stored, as base64 and a line " +
		"at a time), credentials in URLs, bearer and basic tokens, and values assigned to names that say " +
		"they are secrets are replaced with [redacted]"

	minShare  = 4 << 10
	spillSize = 64 << 10
	zipBuffer = 256 << 10
)

type Caps struct {
	Period    time.Duration
	Component int64
	Total     int64
	Sources   int
}

var DefaultCaps = Caps{
	Period:    7 * 24 * time.Hour,
	Component: 128 << 20,
	Total:     256 << 20,
	Sources:   50,
}

type ComponentResult struct {
	Component string     `json:"component"`
	File      string     `json:"file"`
	Lines     int64      `json:"lines"`
	Bytes     int64      `json:"bytes"`
	Truncated bool       `json:"truncated"`
	Cut       string     `json:"cut,omitempty"`
	Oldest    *time.Time `json:"oldest,omitempty"`
	Newest    *time.Time `json:"newest,omitempty"`
}

type ManifestCaps struct {
	PeriodHours    int   `json:"periodHours"`
	ComponentBytes int64 `json:"componentBytes"`
	TotalBytes     int64 `json:"totalBytes"`
	Sources        int   `json:"sources"`
}

type Manifest struct {
	InstanceID      uuid.UUID         `json:"instanceId"`
	BundleID        uuid.UUID         `json:"bundleId"`
	Since           time.Time         `json:"since"`
	Until           time.Time         `json:"until"`
	CreatedAt       time.Time         `json:"createdAt"`
	Format          string            `json:"format"`
	Redaction       string            `json:"redaction"`
	Caps            ManifestCaps      `json:"caps"`
	TotalCapReached bool              `json:"totalCapReached"`
	SkippedSources  []string          `json:"skippedSources,omitempty"`
	Components      []ComponentResult `json:"components"`
}

type Bundle struct {
	Path     string
	Size     int64
	SHA256   string
	Manifest Manifest
}

type bundleSpec struct {
	bundleID   uuid.UUID
	instanceID uuid.UUID
	since      time.Time
	until      time.Time
	dir        string
	redact     func(string) string
	created    time.Time
	pageLimit  int
}

type held struct {
	line string
	at   time.Time
}

type source struct {
	pager     *pager
	path      string
	file      *os.File
	w         *bufio.Writer
	held      *held
	res       ComponentResult
	exhausted bool
	finished  bool
}

func writeBundle(ctx context.Context, src Source, spec *bundleSpec, caps Caps) (*Bundle, error) {
	if !spec.until.After(spec.since) {
		return nil, errors.New("the period ends before it starts")
	}
	if spec.until.Sub(spec.since) > caps.Period {
		return nil, fmt.Errorf("the period is longer than %d days", int(caps.Period.Hours()/24))
	}
	names, err := src.Components(ctx, spec.since, spec.until)
	if err != nil {
		return nil, err
	}
	manifest := Manifest{
		InstanceID: spec.instanceID,
		BundleID:   spec.bundleID,
		Since:      spec.since.UTC(),
		Until:      spec.until.UTC(),
		CreatedAt:  spec.created.UTC(),
		Format:     lineFormat,
		Redaction:  redaction,
		Caps: ManifestCaps{
			PeriodHours:    int(caps.Period.Hours()),
			ComponentBytes: caps.Component,
			TotalBytes:     caps.Total,
			Sources:        caps.Sources,
		},
		Components: []ComponentResult{},
	}
	var sources []*source
	defer func() {
		for _, s := range sources {
			if s.file != nil {
				_ = s.file.Close()
			}
			_ = os.Remove(s.path)
		}
	}()
	limit := spec.pageLimit
	if limit <= 0 {
		limit = pageLimit
	}
	for _, name := range names {
		if !ValidComponent(name) || len(sources) >= caps.Sources {
			manifest.SkippedSources = append(manifest.SkippedSources, name)
			continue
		}
		s := &source{
			pager: newPager(src, name, spec.since, spec.until, limit),
			path:  filepath.Join(spec.dir, spec.bundleID.String()+"-"+name+".spill"),
			res:   ComponentResult{Component: name, File: name + ".log"},
		}
		sources = append(sources, s)
		if s.file, err = os.OpenFile(s.path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600); err != nil {
			return nil, fmt.Errorf("make scratch space for the %s logs: %w", name, err)
		}
		s.w = bufio.NewWriterSize(s.file, spillSize)
	}
	if manifest.TotalCapReached, err = share(ctx, sources, caps, spec.redact); err != nil {
		return nil, err
	}
	for _, s := range sources {
		if err = s.w.Flush(); err != nil {
			return nil, fmt.Errorf("write the %s logs to scratch space: %w", s.res.Component, err)
		}
		manifest.Components = append(manifest.Components, s.res)
	}
	return zipBundle(spec, sources, &manifest)
}

func share(ctx context.Context, sources []*source, caps Caps, redact func(string) string) (bool, error) {
	remaining := caps.Total
	for {
		var active []*source
		for _, s := range sources {
			if !s.finished {
				active = append(active, s)
			}
		}
		if len(active) == 0 {
			return false, nil
		}
		each := remaining / int64(len(active))
		if each < minShare {
			cutAll(active)
			return true, nil
		}
		progressed := false
		for _, s := range active {
			capLeft := caps.Component - s.res.Bytes
			budget := min(each, capLeft)
			used, err := s.fill(ctx, budget, redact)
			if err != nil {
				return false, err
			}
			remaining -= used
			progressed = progressed || used > 0
			switch {
			case s.exhausted:
				s.finished = true
			case budget == capLeft:
				s.finished, s.res.Truncated, s.res.Cut = true, true, CutComponent
			}
		}
		if !progressed {
			var stuck []*source
			for _, s := range active {
				if !s.finished {
					stuck = append(stuck, s)
				}
			}
			if len(stuck) > 0 {
				cutAll(stuck)
				return true, nil
			}
		}
	}
}

func cutAll(sources []*source) {
	for _, s := range sources {
		s.finished, s.res.Truncated, s.res.Cut = true, true, CutBundle
	}
}

func (s *source) fill(ctx context.Context, budget int64, redact func(string) string) (int64, error) {
	var used int64
	for {
		h := s.held
		if h == nil {
			e, ok, err := s.pager.next(ctx)
			if err != nil {
				return used, err
			}
			if !ok {
				s.exhausted = true
				return used, nil
			}
			h = &held{line: bundleLine(&e, redact), at: e.Time}
		}
		size := int64(len(h.line))
		if used+size > budget {
			s.held = h
			return used, nil
		}
		if _, err := s.w.WriteString(h.line); err != nil {
			return used, fmt.Errorf("write the %s logs to scratch space: %w", s.res.Component, err)
		}
		s.held = nil
		used += size
		s.res.Bytes += size
		s.res.Lines++
		at := h.at
		if s.res.Newest == nil {
			s.res.Newest = &at
		}
		s.res.Oldest = &at
	}
}

type counter struct{ n int64 }

func (c *counter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}

func zipBundle(spec *bundleSpec, sources []*source, manifest *Manifest) (*Bundle, error) {
	path := filepath.Join(spec.dir, spec.bundleID.String()+".zip.part")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("make scratch space for the bundle: %w", err)
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	hash := sha256.New()
	size := &counter{}
	buf := bufio.NewWriterSize(io.MultiWriter(f, hash, size), zipBuffer)
	zw := zip.NewWriter(buf)
	for _, s := range sources {
		if err = addSource(zw, s, spec.until); err != nil {
			return nil, err
		}
	}
	body, err := sonic.ConfigStd.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("write the bundle's manifest: %w", err)
	}
	w, err := zw.CreateHeader(&zip.FileHeader{Name: manifestName, Method: zip.Deflate, Modified: spec.until})
	if err != nil {
		return nil, fmt.Errorf("write the bundle's manifest: %w", err)
	}
	if _, err = w.Write(body); err != nil {
		return nil, fmt.Errorf("write the bundle's manifest: %w", err)
	}
	if err = zw.Close(); err != nil {
		return nil, fmt.Errorf("finish the bundle: %w", err)
	}
	if err = buf.Flush(); err != nil {
		return nil, fmt.Errorf("finish the bundle: %w", err)
	}
	if err = f.Close(); err != nil {
		return nil, fmt.Errorf("finish the bundle: %w", err)
	}
	ok = true
	return &Bundle{
		Path:     path,
		Size:     size.n,
		SHA256:   hex.EncodeToString(hash.Sum(nil)),
		Manifest: *manifest,
	}, nil
}

func addSource(zw *zip.Writer, s *source, modified time.Time) error {
	w, err := zw.CreateHeader(&zip.FileHeader{Name: s.res.File, Method: zip.Deflate, Modified: modified})
	if err != nil {
		return fmt.Errorf("add the %s logs to the bundle: %w", s.res.Component, err)
	}
	info, err := s.file.Stat()
	if err != nil {
		return fmt.Errorf("add the %s logs to the bundle: %w", s.res.Component, err)
	}
	newline := []byte{'\n'}
	err = reverseLines(s.file, info.Size(), func(line []byte) error {
		if _, werr := w.Write(line); werr != nil {
			return werr
		}
		_, werr := w.Write(newline)
		return werr
	})
	if err != nil {
		return fmt.Errorf("add the %s logs to the bundle: %w", s.res.Component, err)
	}
	_ = s.file.Close()
	s.file = nil
	return os.Remove(s.path)
}
