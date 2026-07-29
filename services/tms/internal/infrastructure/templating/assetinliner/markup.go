package assetinliner

import (
	"html"
	"sort"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/imageutils"
)

// maxLogoWidthPx and maxLogoHeightPx bound how much of a page an image may claim
// when the template did not size it. The values approximate the fixed logo box the
// coordinate-based renderer used, so an organization that never sized its logo gets
// the same restraint it always had.
const (
	maxLogoWidthPx  = 320
	maxLogoHeightPx = 96
)

// collectAttributes reads every attribute off the current tag.
//
// Later duplicates win, matching how a browser resolves a repeated attribute, so
// an attempt to smuggle a second src past the rewrite lands on the value we
// actually process.
func collectAttributes(tokenizer interface {
	TagAttr() (key, val []byte, moreAttr bool)
}) map[string]string {
	attrs := make(map[string]string, 6)

	for {
		key, val, more := tokenizer.TagAttr()
		attrs[strings.ToLower(strings.TrimSpace(string(key)))] = string(val)
		if !more {
			break
		}
	}

	return attrs
}

// renderImageTag rebuilds an img element from attributes.
//
// It re-emits rather than patching the raw source because the raw form may contain
// the original src, and a rewrite that leaves the old value anywhere in the document
// has not rewritten anything. Every value is escaped on the way out.
func renderImageTag(attrs map[string]string) string {
	var b strings.Builder
	b.WriteString("<img")

	// Emit src first so the markup reads the way an author wrote it.
	if src, ok := attrs["src"]; ok {
		writeAttribute(&b, "src", src)
	}

	for _, name := range sortedAttributeNames(attrs) {
		if name == "src" || !isSafeImageAttribute(name) {
			continue
		}
		writeAttribute(&b, name, attrs[name])
	}

	b.WriteString(">")
	return b.String()
}

func writeAttribute(b *strings.Builder, name, value string) {
	b.WriteString(" ")
	b.WriteString(name)
	b.WriteString(`="`)
	b.WriteString(html.EscapeString(value))
	b.WriteString(`"`)
}

// safeImageAttributes are the attributes an img may keep through a rewrite.
//
// An allowlist rather than a deny list: the sanitizer already refused event
// handlers at save time, but rendered output is assembled from template plus data,
// and re-emitting only what we recognize means a surprise cannot survive the trip.
var safeImageAttributes = map[string]struct{}{
	"alt":    {},
	"width":  {},
	"height": {},
	"class":  {},
	"id":     {},
	"style":  {},
	"title":  {},
	"align":  {},
	"border": {},
}

func isSafeImageAttribute(name string) bool {
	_, ok := safeImageAttributes[name]
	return ok
}

func sortedAttributeNames(attrs map[string]string) []string {
	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// applyIntrinsicDimensions sizes an image the template left unsized.
//
// A template that states its own width and height is left alone: the author knows
// what the layout needs. One that does not gets the image fitted into a bounded
// box, which is what keeps an oversized upload from displacing the document.
func applyIntrinsicDimensions(attrs map[string]string, img *imageutils.Image) {
	if img == nil || img.Width <= 0 || img.Height <= 0 {
		return
	}
	if hasDimension(attrs, "width") || hasDimension(attrs, "height") {
		return
	}

	width, height := imageutils.FittedSize(
		float64(img.Width), float64(img.Height),
		maxLogoWidthPx, maxLogoHeightPx,
	)

	attrs["width"] = strconv.Itoa(int(width))
	attrs["height"] = strconv.Itoa(int(height))
}

func hasDimension(attrs map[string]string, name string) bool {
	value, ok := attrs[name]
	return ok && strings.TrimSpace(value) != ""
}

// isHTTPURL reports whether a src is an absolute http(s) URL.
func isHTTPURL(src string) bool {
	lower := strings.ToLower(src)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// disallowedSchemes are the schemes that must never be resolved, even though the
// save-time sanitizer already rejects them. Rendered output is template plus data,
// and a scheme can arrive through the data.
var disallowedSchemes = []string{
	"file:", "javascript:", "vbscript:", "about:", "chrome:", "blob:",
	"resource:", "jar:", "view-source:", "ftp:", "gopher:", "ws:", "wss:",
}

func isDisallowedScheme(src string) bool {
	lower := strings.ToLower(strings.TrimSpace(src))
	for _, scheme := range disallowedSchemes {
		if strings.HasPrefix(lower, scheme) {
			return true
		}
	}
	// A protocol-relative URL inherits the renderer's scheme and would be fetched.
	return strings.HasPrefix(lower, "//")
}

func schemeOf(src string) string {
	scheme, _, found := strings.Cut(strings.TrimSpace(src), ":")
	if !found {
		return src
	}
	return scheme + ":"
}
