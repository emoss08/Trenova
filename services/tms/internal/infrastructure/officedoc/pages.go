package officedoc

import (
	"bytes"
	"path"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

var ocrImageExts = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".bmp": {},
	".tif": {}, ".tiff": {}, ".webp": {},
}

type pageBuilder struct {
	pages      []services.OfficePage
	text       bytes.Buffer
	images     []services.OfficeImage
	seenImages map[string]struct{}
	textBytes  int
	imageBytes int
	full       bool
}

func newPageBuilder() *pageBuilder {
	return &pageBuilder{seenImages: map[string]struct{}{}}
}

func (b *pageBuilder) done() bool { return b.full }

func (b *pageBuilder) write(s string) {
	if b.full || s == "" {
		return
	}
	room := maxTextBytes - b.textBytes
	if room <= 0 {
		return
	}
	if len(s) > room {
		s = strings.ToValidUTF8(s[:room], "")
	}
	b.text.WriteString(s)
	b.textBytes += len(s)
}

func (b *pageBuilder) writeCollapsed(s string) {
	words := strings.Fields(s)
	if len(words) == 0 {
		if s != "" {
			b.space()
		}
		return
	}
	if startsWithSpace(s) {
		b.space()
	}
	b.write(strings.Join(words, " "))
	if endsWithSpace(s) {
		b.space()
	}
}

func (b *pageBuilder) space() {
	if n := b.text.Len(); n > 0 {
		switch b.text.Bytes()[n-1] {
		case ' ', '\n', '\t':
			return
		}
		b.write(" ")
	}
}

func startsWithSpace(s string) bool {
	return s != "" && strings.TrimLeftFunc(s[:1], unicode.IsSpace) == ""
}

func endsWithSpace(s string) bool {
	return s != "" && strings.TrimRightFunc(s[len(s)-1:], unicode.IsSpace) == ""
}

func (b *pageBuilder) newline() {
	if b.text.Len() > 0 && !bytes.HasSuffix(b.text.Bytes(), []byte("\n")) {
		b.write("\n")
	}
}

func (b *pageBuilder) endCell() {
	b.trimTrailing('\n')
	b.write("\t")
}

func (b *pageBuilder) endRow() {
	b.trimTrailing('\t')
	b.write("\n")
}

func (b *pageBuilder) trimTrailing(c byte) {
	for b.text.Len() > 0 && b.text.Bytes()[b.text.Len()-1] == c {
		b.text.Truncate(b.text.Len() - 1)
		b.textBytes--
	}
}

func (b *pageBuilder) hasContent() bool {
	return strings.TrimSpace(b.text.String()) != "" || len(b.images) > 0
}

func (b *pageBuilder) addImage(pkg *archive, part string) {
	if b.full || len(b.images) >= maxImagesPerPage {
		return
	}
	ext := strings.ToLower(path.Ext(part))
	if _, ok := ocrImageExts[ext]; !ok {
		return
	}
	if _, seen := b.seenImages[part]; seen {
		return
	}

	data, err := pkg.readAll(part, maxImageBytes)
	if err != nil || len(data) == 0 {
		return
	}
	b.attach(path.Base(part), ext, data)
	b.seenImages[part] = struct{}{}
}

func (b *pageBuilder) attach(name, ext string, data []byte) {
	if b.full || len(b.images) >= maxImagesPerPage || len(data) > maxImageBytes {
		return
	}
	if b.imageBytes+len(data) > maxTotalBytes {
		return
	}
	b.imageBytes += len(data)
	b.images = append(b.images, services.OfficeImage{Name: name, Ext: ext, Data: data})
}

func (b *pageBuilder) breakPage() {
	if b.hasContent() {
		b.finishPage()
	}
}

func (b *pageBuilder) finishPage() {
	if b.full {
		return
	}
	b.pages = append(b.pages, services.OfficePage{
		Text:   strings.TrimSpace(b.text.String()),
		Images: b.images,
	})
	b.text.Reset()
	b.images = nil
	b.seenImages = map[string]struct{}{}
	if len(b.pages) >= maxPages {
		b.full = true
	}
}

func (b *pageBuilder) result() []services.OfficePage {
	if b.hasContent() {
		b.finishPage()
	}
	return b.pages
}
