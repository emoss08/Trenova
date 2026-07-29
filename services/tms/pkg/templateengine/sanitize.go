package templateengine

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// The sanitizer rejects rather than rewrites.
//
// A silent strip is worse than a refusal here: the author saves, sees their
// markup accepted, and finds out later that a customer received a document with
// a hole in it. Refusing at save time puts the problem in front of the one person
// who can fix it, while they still have the editor open.

// forbiddenElements can execute, navigate, submit, or load external content.
// Nothing in this set has a legitimate use in a document or an email body.
const (
	reasonFraming      = "embedded frames cannot load in a document or an email"
	reasonFormControls = "form controls have no meaning in a document or an email"
)

var forbiddenElements = map[string]string{
	"script":   "scripts cannot run in a document or an email",
	"iframe":   reasonFraming,
	"frame":    reasonFraming,
	"frameset": reasonFraming,
	"object":   "embedded objects cannot load in a document or an email",
	"embed":    "embedded plugins cannot load in a document or an email",
	"applet":   "embedded applets cannot load in a document or an email",
	"link":     "external stylesheets cannot load; put styles in the CSS field",
	"base":     "a base element would rewrite every relative URL in the document",
	"form":     "forms cannot be submitted from a document or an email",
	"input":    reasonFormControls,
	"button":   reasonFormControls,
	"textarea": reasonFormControls,
	"select":   reasonFormControls,
	// SVG is a script vector as much as an image, and it buys nothing in print.
	"svg":      "SVG is not supported; use a PNG or JPEG image",
	"math":     "MathML is not supported",
	"noscript": "noscript has no meaning where scripts never run",
}

// URL attributes fall into two groups, and conflating them breaks one or the other.
//
// A fetching attribute is resolved by the renderer while it builds the page, so it
// is the one that can reach the network on our behalf. Every asset is inlined as a
// data: URI before rendering, so nothing else is permitted here.
//
// A navigation attribute is resolved by the reader, later, in their own client. An
// https link in an email is the entire point of a portal invitation, and a clickable
// link in a PDF fetches nothing at render time. Refusing those would make it
// impossible to author a message that links anywhere.
var fetchingURLAttributes = map[string]struct{}{
	"src":        {},
	"srcset":     {},
	"poster":     {},
	"background": {},
	"data":       {},
	"usemap":     {},
	"profile":    {},
	"longdesc":   {},
}

var navigationURLAttributes = map[string]struct{}{
	"href":       {},
	"action":     {},
	"formaction": {},
	"cite":       {},
}

// fetchingSchemes is what a render-time fetch may use: nothing but an inline image.
var fetchingSchemes = []string{"data:"}

// navigationSchemes is what a reader may be sent to.
var navigationSchemes = []string{"https:", "http:", "mailto:", "tel:", "data:"}

// dangerousURLSchemes execute or read local state.
var dangerousURLSchemes = []string{
	"javascript:",
	"vbscript:",
	"livescript:",
	"file:",
	"about:",
	"chrome:",
	"chrome-extension:",
	"resource:",
	"jar:",
	"view-source:",
	"blob:",
}

// dataURIAllowedTypes are the media types a data: URI may declare. Notably
// absent: text/html and image/svg+xml, either of which turns an image slot into
// a script delivery channel.
var dataURIAllowedTypes = []string{
	"image/png",
	"image/jpeg",
	"image/jpg",
	"image/gif",
	"image/webp",
}

// cssImportPattern and friends catch stylesheet constructs that fetch or execute.
var (
	cssImportPattern     = regexp.MustCompile(`(?i)@import`)
	cssURLPattern        = regexp.MustCompile(`(?i)url\s*\(\s*['"]?([^'")]*)['"]?\s*\)`)
	cssExpressionPattern = regexp.MustCompile(`(?i)expression\s*\(`)
	cssBehaviorPattern   = regexp.MustCompile(`(?i)\bbehaviou?r\s*:`)
	cssBindingPattern    = regexp.MustCompile(`(?i)-moz-binding\s*:`)
)

// SanitizeHTML walks markup and reports every construct we refuse to render.
//
// It runs on the template source at save time, and again on rendered output
// before that output leaves the process. The second pass is not redundant: the
// data flowing through a template — a customer name, an invoice memo, a shipment
// comment — is attacker-influenced, and a template that is clean on its own can
// still interpolate something hostile into a position that matters.
func SanitizeHTML(field, source string, opts SanitizeOptions) Diagnostics {
	if strings.TrimSpace(source) == "" {
		return nil
	}

	diags := make(Diagnostics, 0, 4)
	tokenizer := html.NewTokenizer(strings.NewReader(source))

	// The tokenizer does not expose positions, so track them by counting the raw
	// bytes it consumes. An author needs to know which line to look at.
	line, column := 1, 1
	inStyle := false
	var styleBody strings.Builder

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			// io.EOF is the normal terminator; the tokenizer is lenient by design
			// and does not report malformed markup as an error.
			break
		}

		raw := string(tokenizer.Raw())
		tokenLine, tokenColumn := line, column
		line, column = advancePosition(raw, line, column)

		switch tokenType {
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := tokenizer.TagName()
			tag := strings.ToLower(string(name))

			if reason, forbidden := forbiddenElements[tag]; forbidden {
				diags = append(diags, positionedError(
					CodeForbiddenElement, field,
					fmt.Sprintf("<%s> is not allowed: %s", tag, reason),
					tokenLine, tokenColumn,
				))
				continue
			}

			if tag == "style" && tokenType == html.StartTagToken {
				if !opts.AllowStyleElement {
					diags = append(diags, positionedError(
						CodeForbiddenElement, field,
						"<style> blocks are stripped by mail clients; use inline style attributes",
						tokenLine, tokenColumn,
					))
					continue
				}
				inStyle = true
				styleBody.Reset()
			}

			diags = append(diags, checkAttributes(
				tokenizer, tag, field, hasAttr, tokenLine, tokenColumn,
			)...)

		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			if strings.EqualFold(string(name), "style") && inStyle {
				inStyle = false
				diags = append(diags, SanitizeCSS(field, styleBody.String())...)
			}

		case html.TextToken:
			if inStyle {
				styleBody.Write(tokenizer.Text())
			}

		case html.CommentToken:
			// Conditional comments are the one comment form that executes, in
			// legacy mail clients.
			if text := string(tokenizer.Text()); strings.Contains(text, "[if") {
				diags = append(diags, positionedError(
					CodeForbiddenElement, field,
					"conditional comments are not allowed",
					tokenLine, tokenColumn,
				))
			}

		case html.DoctypeToken, html.ErrorToken:
		}
	}

	return diags
}

// checkAttributes reports forbidden attributes and URL schemes on one tag.
func checkAttributes(
	tokenizer *html.Tokenizer,
	tag, field string,
	hasAttr bool,
	line, column int,
) Diagnostics {
	var diags Diagnostics

	for hasAttr {
		var key, val []byte
		key, val, hasAttr = tokenizer.TagAttr()
		name := strings.ToLower(strings.TrimSpace(string(key)))
		value := string(val)

		// Any on* attribute is an inline event handler.
		if strings.HasPrefix(name, "on") {
			diags = append(diags, positionedError(
				CodeForbiddenAttribute, field,
				fmt.Sprintf("event handler attribute %q is not allowed on <%s>", name, tag),
				line, column,
			))
			continue
		}

		// srcdoc injects a whole nested document.
		if name == "srcdoc" {
			diags = append(diags, positionedError(
				CodeForbiddenAttribute, field,
				"srcdoc is not allowed",
				line, column,
			))
			continue
		}

		if name == "http-equiv" {
			diags = append(diags, positionedError(
				CodeForbiddenAttribute, field,
				"http-equiv can redirect the document and is not allowed",
				line, column,
			))
			continue
		}

		if name == "style" {
			diags = append(diags, SanitizeCSS(field, value)...)
			continue
		}

		if _, fetches := fetchingURLAttributes[name]; fetches {
			if diag, bad := checkURLScheme(field, tag, name, value, true, line, column); bad {
				diags = append(diags, diag)
			}
			continue
		}

		if _, navigates := navigationURLAttributes[name]; navigates {
			if diag, bad := checkURLScheme(field, tag, name, value, false, line, column); bad {
				diags = append(diags, diag)
			}
		}
	}

	return diags
}

// checkURLScheme validates one URL attribute value.
//
// fetched selects the stricter rule set: a render-time fetch may only be an inline
// image, while a link the reader clicks may point at the web.
func checkURLScheme(
	field, tag, attr, value string,
	fetched bool,
	line, column int,
) (Diagnostic, bool) {
	// Entity-encoded and whitespace-padded schemes are the standard evasions;
	// browsers decode both before dispatching, so we normalize the same way.
	normalized := normalizeURLForSchemeCheck(value)
	if normalized == "" {
		return Diagnostic{}, false
	}

	for _, scheme := range dangerousURLSchemes {
		if strings.HasPrefix(normalized, scheme) {
			return positionedError(
				CodeForbiddenURLScheme, field,
				fmt.Sprintf("%s on <%s> uses the %s scheme, which is not allowed",
					attr, tag, strings.TrimSuffix(scheme, ":")),
				line, column,
			), true
		}
	}

	if strings.HasPrefix(normalized, "data:") {
		if diag, bad := checkDataURIType(field, tag, attr, normalized, line, column); bad {
			return diag, true
		}
		return Diagnostic{}, false
	}

	allowed := navigationSchemes
	if fetched {
		allowed = fetchingSchemes
	}
	for _, scheme := range allowed {
		if strings.HasPrefix(normalized, scheme) {
			return Diagnostic{}, false
		}
	}

	// A template variable in the URL position resolves at render time; the
	// render-time pass checks the result.
	if strings.Contains(value, "{{") {
		return Diagnostic{}, false
	}

	if fetched {
		// The renderer has no network route, so a remote or relative asset would
		// silently fail to load even if we allowed it through.
		return positionedError(
			CodeForbiddenURLScheme, field,
			fmt.Sprintf(
				"%s on <%s> must be a data: URI; remote and relative URLs cannot be "+
					"loaded while rendering. Upload the image and reference it by key, "+
					"or inline it as a data: URI",
				attr, tag,
			),
			line, column,
		), true
	}

	// A relative link has no base to resolve against once a document has left the
	// system, so it would arrive at the reader broken.
	return positionedError(
		CodeForbiddenURLScheme, field,
		fmt.Sprintf(
			"%s on <%s> must be an absolute https, http, mailto, or tel URL; a relative "+
				"link has nothing to resolve against once this has been sent",
			attr, tag,
		),
		line, column,
	), true
}

func checkDataURIType(
	field, tag, attr, normalized string,
	line, column int,
) (Diagnostic, bool) {
	meta, _, _ := strings.Cut(strings.TrimPrefix(normalized, "data:"), ",")
	mediaType, _, _ := strings.Cut(meta, ";")
	mediaType = strings.TrimSpace(mediaType)

	// A bare "data:," defaults to text/plain, which is inert.
	if mediaType == "" {
		return Diagnostic{}, false
	}

	for _, allowed := range dataURIAllowedTypes {
		if mediaType == allowed {
			return Diagnostic{}, false
		}
	}

	return positionedError(
		CodeForbiddenURLScheme, field,
		fmt.Sprintf(
			"%s on <%s> declares data: type %q; only PNG, JPEG, GIF, and WebP images are allowed",
			attr, tag, mediaType,
		),
		line, column,
	), true
}

// normalizeURLForSchemeCheck strips the tricks used to hide a scheme: HTML
// entities, percent-encoding of the colon, embedded control characters, and
// leading whitespace. Browsers ignore all of them when resolving a URL, so a
// check that does not is a check that can be walked around.
func normalizeURLForSchemeCheck(value string) string {
	unescaped := html.UnescapeString(value)

	var b strings.Builder
	b.Grow(len(unescaped))
	for _, r := range unescaped {
		// Tab, newline, and NUL are ignored inside a scheme by every browser.
		if r == '\t' || r == '\n' || r == '\r' || r == 0 {
			continue
		}
		b.WriteRune(r)
	}

	normalized := strings.ToLower(strings.TrimSpace(b.String()))
	// %3a is a colon; decoding only that one sequence is enough to expose a
	// hidden scheme without having to fully parse the URL.
	normalized = strings.ReplaceAll(normalized, "%3a", ":")

	return normalized
}

// SanitizeCSS reports stylesheet constructs that fetch or execute.
func SanitizeCSS(field, source string) Diagnostics {
	if strings.TrimSpace(source) == "" {
		return nil
	}

	var diags Diagnostics

	if cssImportPattern.MatchString(source) {
		diags = append(diags, errorDiag(CodeForbiddenCSS, field,
			"@import is not allowed: it would fetch a remote stylesheet"))
	}
	if cssExpressionPattern.MatchString(source) {
		diags = append(diags, errorDiag(CodeForbiddenCSS, field,
			"expression() is not allowed: it executes script"))
	}
	if cssBehaviorPattern.MatchString(source) {
		diags = append(diags, errorDiag(CodeForbiddenCSS, field,
			"behavior is not allowed: it executes script"))
	}
	if cssBindingPattern.MatchString(source) {
		diags = append(diags, errorDiag(CodeForbiddenCSS, field,
			"-moz-binding is not allowed: it executes script"))
	}

	for _, match := range cssURLPattern.FindAllStringSubmatch(source, -1) {
		target := normalizeURLForSchemeCheck(match[1])
		if target == "" || strings.HasPrefix(target, "data:") {
			continue
		}
		if strings.Contains(match[1], "{{") {
			continue
		}
		diags = append(diags, errorDiag(CodeForbiddenCSS, field, fmt.Sprintf(
			"url(%s) is not allowed: only data: URIs can be loaded",
			strings.TrimSpace(match[1]),
		)))
	}

	return diags
}

// SanitizeOptions tunes the rules per output channel.
type SanitizeOptions struct {
	// AllowStyleElement permits a <style> block. True for PDF documents, where a
	// real stylesheet is the point. False for email, where clients strip <style>
	// and an author who relies on it ships a document that looks broken in half
	// the inboxes that receive it.
	AllowStyleElement bool
}

// DocumentSanitizeOptions is the rule set for PDF output.
func DocumentSanitizeOptions() SanitizeOptions {
	return SanitizeOptions{AllowStyleElement: true}
}

// EmailSanitizeOptions is the rule set for HTML email bodies.
func EmailSanitizeOptions() SanitizeOptions {
	return SanitizeOptions{AllowStyleElement: false}
}

// advancePosition tracks line and column across consumed source bytes.
func advancePosition(raw string, line, column int) (nextLine, nextColumn int) {
	for _, r := range raw {
		if r == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}
