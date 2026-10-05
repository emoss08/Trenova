package officedoc

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	nsMarkupCompat = "http://schemas.openxmlformats.org/markup-compatibility/2006"
	ctxCheckEvery  = 512
)

func isRelationshipNS(space string) bool {
	return strings.HasSuffix(space, "/officeDocument/2006/relationships") ||
		strings.HasSuffix(space, "/package/2006/relationships")
}

func relAttr(el xml.StartElement, local string) string {
	for _, attr := range el.Attr {
		if attr.Name.Local == local && isRelationshipNS(attr.Name.Space) {
			return attr.Value
		}
	}
	return ""
}

func attr(el xml.StartElement, local string) string {
	for _, a := range el.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func isFalse(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "off":
		return true
	default:
		return false
	}
}

type ooxmlWalker struct {
	ctx     context.Context
	pkg     *archive
	pages   *pageBuilder
	part    string
	rels    map[string]relationship
	tokens  int
	runs    int
	inText  bool
	docxRun bool
}

func (w *ooxmlWalker) walk(dec *xml.Decoder) error {
	for {
		if w.pages.done() {
			return nil
		}
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("parse %s: %w", w.part, err)
		}

		w.tokens++
		if w.tokens%ctxCheckEvery == 0 {
			if err = w.ctx.Err(); err != nil {
				return err
			}
		}

		switch el := tok.(type) {
		case xml.StartElement:
			if err = w.start(dec, el); err != nil {
				return err
			}
		case xml.EndElement:
			w.end(el)
		case xml.CharData:
			if w.inText {
				w.pages.write(string(el))
			}
		}
	}
}

func (w *ooxmlWalker) start(dec *xml.Decoder, el xml.StartElement) error {
	if el.Name.Space == nsMarkupCompat && el.Name.Local == "Fallback" {
		return dec.Skip()
	}

	switch el.Name.Local {
	case "instrText", "delText", "delInstrText":
		return dec.Skip()
	case "r":
		w.runs++
	case "t":
		w.inText = true
	case "tab":
		if w.runs > 0 {
			w.pages.write("\t")
		}
	case "br":
		w.lineBreak(el)
	case "cr":
		if w.runs > 0 {
			w.pages.write("\n")
		}
	case "lastRenderedPageBreak":
		if w.docxRun {
			w.pages.breakPage()
		}
	case "pageBreakBefore":
		if w.docxRun && !isFalse(attr(el, "val")) {
			w.pages.breakPage()
		}
	case "blip":
		w.image(relAttr(el, "embed"))
		w.image(relAttr(el, "link"))
	case "imagedata":
		w.image(relAttr(el, "id"))
	}

	return nil
}

func (w *ooxmlWalker) lineBreak(el xml.StartElement) {
	if w.docxRun {
		if w.runs == 0 {
			return
		}
		if attr(el, "type") == "page" {
			w.pages.breakPage()
			return
		}
	}
	w.pages.write("\n")
}

func (w *ooxmlWalker) end(el xml.EndElement) {
	switch el.Name.Local {
	case "r":
		if w.runs > 0 {
			w.runs--
		}
	case "t":
		w.inText = false
	case "p":
		w.pages.newline()
	case "tc":
		w.pages.endCell()
	case "tr":
		w.pages.endRow()
	}
}

func (w *ooxmlWalker) image(id string) {
	if id == "" {
		return
	}
	rel, ok := w.rels[id]
	if !ok || rel.external {
		return
	}
	w.pages.addImage(w.pkg, rel.target)
}

func walkPart(ctx context.Context, pkg *archive, pages *pageBuilder, part string, docx bool) error {
	rels, err := pkg.relationships(part)
	if err != nil {
		return err
	}

	dec, closer, err := pkg.decoder(part)
	if err != nil {
		return err
	}
	defer closer.Close()

	walker := &ooxmlWalker{ctx: ctx, pkg: pkg, pages: pages, part: part, rels: rels, docxRun: docx}
	return walker.walk(dec)
}

func readDOCX(ctx context.Context, pkg *archive) ([]services.OfficePage, error) {
	part := pkg.mainPart("/document.xml", "word/document.xml")
	pages := newPageBuilder()

	if err := walkPart(ctx, pkg, pages, part, true); err != nil {
		return nil, err
	}

	return pages.result(), nil
}

func readPPTX(ctx context.Context, pkg *archive) ([]services.OfficePage, error) {
	presentation := pkg.mainPart("/presentation.xml", "ppt/presentation.xml")
	rels, err := pkg.relationships(presentation)
	if err != nil {
		return nil, err
	}

	slideIDs, err := slideOrder(pkg, presentation)
	if err != nil {
		return nil, err
	}

	pages := newPageBuilder()
	for _, id := range slideIDs {
		if pages.done() {
			break
		}
		rel, ok := rels[id]
		if !ok || rel.external || !pkg.has(rel.target) {
			continue
		}
		if err = walkPart(ctx, pkg, pages, rel.target, false); err != nil {
			return nil, err
		}
		pages.finishPage()
	}

	return pages.pages, nil
}

func slideOrder(pkg *archive, presentation string) ([]string, error) {
	dec, closer, err := pkg.decoder(presentation)
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	var ids []string
	for {
		tok, tokErr := dec.Token()
		if errors.Is(tokErr, io.EOF) {
			return ids, nil
		}
		if tokErr != nil {
			return nil, fmt.Errorf("parse %s: %w", presentation, tokErr)
		}
		if el, ok := tok.(xml.StartElement); ok && el.Name.Local == "sldId" {
			if id := relAttr(el, "id"); id != "" {
				ids = append(ids, id)
			}
		}
	}
}
