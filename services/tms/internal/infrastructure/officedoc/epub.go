package officedoc

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"golang.org/x/net/html"
)

const epubContainer = "META-INF/container.xml"

var errNoRootFile = errors.New("epub has no package document")

var htmlBlocks = map[string]struct{}{
	"p": {}, "div": {}, "br": {}, "li": {}, "ul": {}, "ol": {},
	"h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {}, "h6": {},
	"section": {}, "article": {}, "blockquote": {}, "pre": {},
	"table": {}, "hr": {}, "dd": {}, "dt": {}, "header": {},
	"footer": {}, "figure": {}, "figcaption": {}, "aside": {}, "nav": {},
}

var htmlSkipped = map[string]struct{}{
	"head": {}, "script": {}, "style": {}, "noscript": {}, "template": {},
}

func readEPUB(ctx context.Context, pkg *archive) ([]services.OfficePage, error) {
	opf, err := epubRootFile(pkg)
	if err != nil {
		return nil, err
	}

	raw, err := pkg.readAll(opf, maxPartBytes)
	if err != nil {
		return nil, err
	}

	var manifest struct {
		Items []struct {
			ID        string `xml:"id,attr"`
			Href      string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	if err = xml.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("parse %s: %w", opf, err)
	}

	opfDir := path.Dir(opf)
	hrefs := make(map[string]string, len(manifest.Items))
	for _, item := range manifest.Items {
		hrefs[item.ID] = resolvePart(opfDir, item.Href)
	}

	pages := newPageBuilder()
	for _, ref := range manifest.Spine {
		if pages.done() {
			break
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		part, ok := hrefs[ref.IDRef]
		if !ok || !pkg.has(part) {
			continue
		}
		if err = readXHTML(ctx, pkg, part, pages); err != nil {
			return nil, err
		}
		pages.finishPage()
	}

	return pages.pages, nil
}

func epubRootFile(pkg *archive) (string, error) {
	if pkg.has(epubContainer) {
		raw, err := pkg.readAll(epubContainer, maxPartBytes)
		if err != nil {
			return "", err
		}
		var container struct {
			RootFiles []struct {
				FullPath string `xml:"full-path,attr"`
			} `xml:"rootfiles>rootfile"`
		}
		if err = xml.Unmarshal(raw, &container); err != nil {
			return "", fmt.Errorf("parse %s: %w", epubContainer, err)
		}
		for _, root := range container.RootFiles {
			if part := resolvePart("", root.FullPath); pkg.has(part) {
				return part, nil
			}
		}
	}

	for name := range pkg.files {
		if strings.HasSuffix(strings.ToLower(name), ".opf") {
			return name, nil
		}
	}

	return "", errNoRootFile
}

type xhtmlWalker struct {
	pkg       *archive
	pages     *pageBuilder
	dir       string
	skipDepth int
}

func readXHTML(ctx context.Context, pkg *archive, part string, pages *pageBuilder) error {
	rc, err := pkg.open(part, maxPartBytes)
	if err != nil {
		return err
	}
	defer rc.Close()

	walker := &xhtmlWalker{pkg: pkg, pages: pages, dir: path.Dir(part)}
	tokenizer := html.NewTokenizer(rc)

	for tokens := 1; !pages.done(); tokens++ {
		if tokens%ctxCheckEvery == 0 {
			if err = ctx.Err(); err != nil {
				return err
			}
		}

		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			if errors.Is(tokenizer.Err(), io.EOF) {
				return nil
			}
			return fmt.Errorf("parse %s: %w", part, tokenizer.Err())
		}
		walker.token(tt, tokenizer)
	}

	return nil
}

func (w *xhtmlWalker) token(tt html.TokenType, tokenizer *html.Tokenizer) {
	switch tt {
	case html.StartTagToken, html.SelfClosingTagToken:
		tok := tokenizer.Token()
		if _, skip := htmlSkipped[tok.Data]; skip {
			if tt == html.StartTagToken {
				w.skipDepth++
			}
			return
		}
		if w.skipDepth == 0 {
			w.start(tok)
		}
	case html.EndTagToken:
		tok := tokenizer.Token()
		if _, skip := htmlSkipped[tok.Data]; skip {
			w.skipDepth = max(0, w.skipDepth-1)
			return
		}
		if w.skipDepth == 0 {
			w.end(tok)
		}
	case html.TextToken:
		if w.skipDepth == 0 {
			w.pages.writeCollapsed(string(tokenizer.Text()))
		}
	case html.ErrorToken, html.CommentToken, html.DoctypeToken:
	}
}

func (w *xhtmlWalker) start(tok html.Token) {
	if _, block := htmlBlocks[tok.Data]; block {
		w.pages.newline()
	}

	var src string
	switch tok.Data {
	case "img":
		src = tokenAttr(tok, "src")
	case "image":
		src = tokenAttr(tok, "href")
	}
	if src != "" && !strings.Contains(src, ":") {
		w.pages.addImage(w.pkg, resolvePart(w.dir, src))
	}
}

func (w *xhtmlWalker) end(tok html.Token) {
	switch tok.Data {
	case "td", "th":
		w.pages.endCell()
	case "tr":
		w.pages.endRow()
	default:
		if _, block := htmlBlocks[tok.Data]; block {
			w.pages.newline()
		}
	}
}

func tokenAttr(tok html.Token, name string) string {
	for _, a := range tok.Attr {
		if strings.EqualFold(a.Key, name) || strings.EqualFold(a.Key, "xlink:"+name) {
			return a.Val
		}
	}
	return ""
}
