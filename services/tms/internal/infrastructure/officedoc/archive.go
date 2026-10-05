package officedoc

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
)

var (
	errTooManyEntries = errors.New("archive has too many entries")
	errPartTooLarge   = errors.New("archive part is too large")
	errArchiveTooBig  = errors.New("archive expands past the size limit")
	errPartMissing    = errors.New("archive part is missing")
)

type relationship struct {
	target   string
	relType  string
	external bool
}

type archive struct {
	files    map[string]*zip.File
	folded   map[string]*zip.File
	consumed int64
	rels     map[string]map[string]relationship
}

func openArchive(data []byte) (*archive, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	if len(zr.File) > maxArchiveEntries {
		return nil, errTooManyEntries
	}

	a := &archive{
		files:  make(map[string]*zip.File, len(zr.File)),
		folded: make(map[string]*zip.File, len(zr.File)),
		rels:   map[string]map[string]relationship{},
	}
	for _, f := range zr.File {
		name := normalizePart(f.Name)
		a.files[name] = f
		a.folded[strings.ToLower(name)] = f
	}

	return a, nil
}

func normalizePart(name string) string {
	return strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(name, "\\", "/")), "/")
}

func (a *archive) lookup(name string) *zip.File {
	name = normalizePart(name)
	if f, ok := a.files[name]; ok {
		return f
	}
	return a.folded[strings.ToLower(name)]
}

func (a *archive) has(name string) bool { return a.lookup(name) != nil }

func (a *archive) open(name string, limit int64) (io.ReadCloser, error) {
	f := a.lookup(name)
	if f == nil {
		return nil, fmt.Errorf("%w: %s", errPartMissing, name)
	}
	if a.consumed >= maxTotalBytes {
		return nil, errArchiveTooBig
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}

	return &meteredReader{rc: rc, archive: a, remaining: limit, name: name}, nil
}

func (a *archive) readAll(name string, limit int64) ([]byte, error) {
	rc, err := a.open(name, limit)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

func (a *archive) decoder(name string) (*xml.Decoder, io.Closer, error) {
	rc, err := a.open(name, maxPartBytes)
	if err != nil {
		return nil, nil, err
	}

	dec := xml.NewDecoder(rc)
	dec.Strict = false
	dec.CharsetReader = passthroughCharset

	return dec, rc, nil
}

func passthroughCharset(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(charset) {
	case "", "utf-8", "utf8", "us-ascii", "ascii":
		return input, nil
	default:
		return nil, fmt.Errorf("unsupported charset %q", charset)
	}
}

func (a *archive) relationships(part string) (map[string]relationship, error) {
	part = normalizePart(part)
	if rels, ok := a.rels[part]; ok {
		return rels, nil
	}

	dir, base := path.Split(part)
	relsPath := path.Join(dir, "_rels", base+".rels")
	rels := map[string]relationship{}
	a.rels[part] = rels
	if !a.has(relsPath) {
		return rels, nil
	}

	raw, err := a.readAll(relsPath, maxPartBytes)
	if err != nil {
		return nil, err
	}

	var doc struct {
		Relationships []struct {
			ID         string `xml:"Id,attr"`
			Target     string `xml:"Target,attr"`
			Type       string `xml:"Type,attr"`
			TargetMode string `xml:"TargetMode,attr"`
		} `xml:"Relationship"`
	}
	if err = xml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", relsPath, err)
	}

	for _, rel := range doc.Relationships {
		external := strings.EqualFold(rel.TargetMode, "External")
		target := rel.Target
		if !external {
			target = resolvePart(dir, rel.Target)
		}
		rels[rel.ID] = relationship{target: target, relType: rel.Type, external: external}
	}

	return rels, nil
}

func (a *archive) mainPart(suffix, fallback string) string {
	rels, err := a.relationships("")
	if err == nil {
		for _, rel := range rels {
			if !rel.external && strings.HasSuffix(rel.relType, "/officeDocument") &&
				a.has(rel.target) {
				return rel.target
			}
		}
	}
	if a.has(fallback) {
		return fallback
	}
	for name := range a.files {
		if strings.HasSuffix(name, suffix) {
			return name
		}
	}
	return fallback
}

func resolvePart(baseDir, target string) string {
	target, _, _ = strings.Cut(target, "#")
	if unescaped, err := url.PathUnescape(target); err == nil {
		target = unescaped
	}
	if strings.HasPrefix(target, "/") {
		return normalizePart(target)
	}
	return normalizePart(path.Join(baseDir, target))
}

type meteredReader struct {
	rc        io.ReadCloser
	archive   *archive
	remaining int64
	name      string
}

func (m *meteredReader) Read(p []byte) (int, error) {
	n, err := m.rc.Read(p)
	m.remaining -= int64(n)
	m.archive.consumed += int64(n)
	if m.remaining < 0 {
		return n, fmt.Errorf("%w: %s", errPartTooLarge, m.name)
	}
	if m.archive.consumed > maxTotalBytes {
		return n, errArchiveTooBig
	}
	return n, err
}

func (m *meteredReader) Close() error { return m.rc.Close() }
