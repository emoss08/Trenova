// Package templateengine parses, validates, and renders the HTML and text
// templates that organizations author for their documents and messages.
//
// It is a pure package: no database, no storage, no network. That is what lets
// the domain validator and the rendering service share one implementation, and it
// is what makes the security tests here fast enough to run on every commit.
//
// The contract is that a template is validated at save time and re-checked at
// publish time against sample data, so a template that reaches a customer has
// already been executed once by us.
package templateengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	texttemplate "text/template"
	"text/template/parse"

	lru "github.com/hashicorp/golang-lru/v2"
)

// Defaults applied when a Config field is zero.
const (
	defaultMaxSourceBytes = 256 << 10
	defaultMaxHTMLBytes   = 2 << 20
	defaultMaxTextBytes   = 64 << 10
	defaultCacheSize      = 512
)

// Config bounds what the engine will accept and produce.
type Config struct {
	MaxSourceBytes int64
	MaxHTMLBytes   int64
	MaxTextBytes   int64
	CacheSize      int
}

func (c Config) withDefaults() Config {
	if c.MaxSourceBytes <= 0 {
		c.MaxSourceBytes = defaultMaxSourceBytes
	}
	if c.MaxHTMLBytes <= 0 {
		c.MaxHTMLBytes = defaultMaxHTMLBytes
	}
	if c.MaxTextBytes <= 0 {
		c.MaxTextBytes = defaultMaxTextBytes
	}
	if c.CacheSize <= 0 {
		c.CacheSize = defaultCacheSize
	}
	return c
}

// Engine compiles and renders templates.
type Engine struct {
	cfg   Config
	cache *lru.Cache[string, *Compiled]
}

// New builds an engine. It only fails if the cache size is unusable, which the
// defaults prevent.
func New(cfg Config) (*Engine, error) {
	cfg = cfg.withDefaults()

	cache, err := lru.New[string, *Compiled](cfg.CacheSize)
	if err != nil {
		return nil, fmt.Errorf("create template cache: %w", err)
	}

	return &Engine{cfg: cfg, cache: cache}, nil
}

// MustNew is New for wiring code that cannot handle a failure.
func MustNew(cfg Config) *Engine {
	engine, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return engine
}

// MaxOutputBytes reports the output budget for a channel.
func (e *Engine) MaxOutputBytes(channel Channel) int64 {
	if channel.IsHTML() {
		return e.cfg.MaxHTMLBytes
	}
	return e.cfg.MaxTextBytes
}

// ParseRequest describes one template body to validate.
type ParseRequest struct {
	// Field names the editor pane this source came from, so diagnostics point at
	// the right box in the UI.
	Field string

	// Kind is the template kind, used only to name the compiled template.
	Kind string

	Channel Channel
	Source  string

	// AllowedPaths is the kind's variable catalog, flattened.
	AllowedPaths []string

	// Strict turns advisory findings into blocking ones. False while drafting,
	// true when publishing.
	Strict bool

	// CacheKey enables reuse across renders. Published version ids are stable and
	// immutable, so they make good keys; leave empty for drafts and previews.
	CacheKey string
}

// Parse validates a template body and compiles it.
//
// It returns a nil *Compiled whenever the diagnostics contain an error, so a
// caller cannot accidentally render something that failed validation.
func (e *Engine) Parse(req *ParseRequest) (*Compiled, Diagnostics) {
	if req.CacheKey != "" {
		if cached, ok := e.cache.Get(req.CacheKey); ok {
			return cached, nil
		}
	}

	field := req.Field
	if field == "" {
		field = string(req.Channel)
	}

	if int64(len(req.Source)) > e.cfg.MaxSourceBytes {
		return nil, Diagnostics{errorDiag(CodeTemplateTooLarge, field, fmt.Sprintf(
			"template is %d bytes; the limit is %d",
			len(req.Source), e.cfg.MaxSourceBytes,
		))}
	}

	var diags Diagnostics

	// Sanitize before parsing. A refusal here names a line the author can see,
	// where a parse failure on the same markup would not.
	if req.Channel.IsHTML() {
		opts := DocumentSanitizeOptions()
		if req.Channel == ChannelEmailHTML {
			opts = EmailSanitizeOptions()
		}
		diags = append(diags, SanitizeHTML(field, req.Source, opts)...)
	}

	compiled, parseDiags := e.compile(req, field)
	diags = append(diags, parseDiags...)
	if diags.HasErrors() {
		return nil, diags
	}

	if req.CacheKey != "" && compiled != nil {
		e.cache.Add(req.CacheKey, compiled)
	}

	return compiled, diags
}

// compile parses the source, then analyzes the resulting tree.
func (e *Engine) compile(req *ParseRequest, field string) (*Compiled, Diagnostics) {
	name := req.Kind + ":" + string(req.Channel)

	compiled := &Compiled{
		Channel:     req.Channel,
		ContentHash: ContentHash(req.Source),
	}

	var diags Diagnostics
	var rootTree *parse.Tree
	var associatedCount int

	if req.Channel.IsHTML() {
		tmpl, err := template.New(name).
			Funcs(FuncMap()).
			Option("missingkey=error").
			Parse(req.Source)
		if err != nil {
			return nil, Diagnostics{parseErrorDiag(field, err)}
		}
		compiled.htmlTmpl = tmpl
		// Read the tree before the first Execute: html/template rewrites it in
		// place while inserting escapers, and the author's structure is what we
		// want to reason about.
		rootTree = tmpl.Tree
		associatedCount = len(tmpl.Templates())
	} else {
		tmpl, err := texttemplate.New(name).
			Funcs(TextFuncMap()).
			Option("missingkey=error").
			Parse(req.Source)
		if err != nil {
			return nil, Diagnostics{parseErrorDiag(field, err)}
		}
		compiled.textTmpl = tmpl
		rootTree = tmpl.Tree
		associatedCount = len(tmpl.Templates())
	}

	result := analyzeTree(rootTree)

	// A version is exactly one template. define/template/block would let one
	// version reach into another, which breaks the promise that the bytes a
	// version renders are reproducible from that version alone.
	for _, nested := range result.nestedTemplates {
		diags = append(diags, errorDiag(CodeNestedTemplateForbidden, field, fmt.Sprintf(
			"{{template %q}} is not allowed; a template version must be self-contained", nested,
		)))
	}
	if associatedCount > 1 {
		diags = append(diags, errorDiag(CodeNestedTemplateForbidden, field,
			"{{define}} and {{block}} are not allowed; a template version must be self-contained"))
	}

	compiled.Paths = result.Paths()

	diags = append(diags, checkFunctions(field, result.functions)...)
	diags = append(diags, checkPaths(field, compiled.Paths, req.AllowedPaths, req.Strict)...)

	return compiled, diags
}

// VerifyRequest is a publish-time check: render against sample data and refuse
// anything that fails.
type VerifyRequest struct {
	Field    string
	Channel  Channel
	Compiled *Compiled
	Sample   any
}

// zgotmplZ is the placeholder html/template substitutes when a value cannot be
// safely escaped for the position it landed in — a dangerous URL scheme in an
// href or src, or a CSS value carrying punctuation that would end the
// declaration.
//
// It is data-driven, not template-shaped: the same template renders cleanly with
// one value and yields ZgotmplZ with another. So finding it means the template
// puts a field somewhere that field cannot survive, and a customer will one day
// receive a document with a broken link or a missing style. Callers decide how
// adversarial the data they verify against is; the registry's hostile probe
// context exists for exactly this.
const zgotmplZ = "ZgotmplZ"

// Verify executes a compiled template against supplied data.
//
// This is the last gate before a version goes live, and the reason an author
// cannot publish a template that only fails on real data: the sample context
// exercises every declared variable, including the ones a hand-test would miss.
func (e *Engine) Verify(ctx context.Context, req VerifyRequest) Diagnostics {
	if req.Compiled == nil {
		return Diagnostics{errorDiag(CodeSampleRenderFailed, req.Field,
			"template did not compile, so it cannot be verified")}
	}

	output, err := req.Compiled.Render(ctx, req.Sample, e.MaxOutputBytes(req.Channel))
	if err != nil {
		code := CodeSampleRenderFailed
		if errors.Is(err, ErrOutputTooLarge) {
			code = CodeOutputTooLarge
		}
		return Diagnostics{errorDiag(code, req.Field, fmt.Sprintf(
			"rendering against sample data failed: %s", err.Error(),
		))}
	}

	if strings.Contains(output, zgotmplZ) {
		return Diagnostics{errorDiag(CodeUnsafeInterpolationContext, req.Field,
			"a variable is placed where its value cannot be safely escaped, so it was "+
				"replaced with a placeholder. This happens when a text field is used as a "+
				"URL or as a CSS value. Move the value into the document body, or use a "+
				"field that holds a data: URI")}
	}

	return nil
}

// CheckRequiredPaths enforces the paths a kind declares mandatory.
//
// It takes the union of the paths referenced across every channel of a version,
// because a version is published as a whole. A field that only belongs in a subject
// line cannot be demanded of the body, and a charge table cannot be demanded of a
// subject; asking per channel would make one or the other impossible to satisfy.
//
// This is what stops an organization from publishing an invoice that no longer
// prints its total or its remit-to address.
func CheckRequiredPaths(field string, referencedAcrossChannels, required []string) Diagnostics {
	return checkRequiredPaths(field, referencedAcrossChannels, required)
}

// Render executes a compiled template under the channel's output budget.
func (e *Engine) Render(ctx context.Context, compiled *Compiled, data any) (string, error) {
	if compiled == nil {
		return "", errors.New("cannot render a nil template")
	}
	return compiled.Render(ctx, data, e.MaxOutputBytes(compiled.Channel))
}

// Invalidate drops a cached compilation, for when a version is edited in place.
func (e *Engine) Invalidate(cacheKey string) {
	if cacheKey != "" {
		e.cache.Remove(cacheKey)
	}
}

// ContentHash is the stable identity of a template body, used for cache keys and
// for recording which bytes produced a given document.
func ContentHash(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

// undefinedFunctionPattern matches how text/template reports a call to something
// outside the FuncMap. It is a parse-time failure, which is why there is no
// separate function-allowlist pass: an unknown call never reaches analysis.
var undefinedFunctionPattern = regexp.MustCompile(`function "([^"]+)" not defined`)

// parseErrorDiag converts a template parse failure into a positioned diagnostic.
//
// text/template reports position as "name:line: message" or
// "name:line:col: message", which is the only place the line number exists.
//
// Undefined-function failures get their own code rather than being reported as
// generic syntax errors, so the editor can answer the author's actual question —
// which functions can I call — instead of leaving them to guess.
func parseErrorDiag(field string, err error) Diagnostic {
	message := err.Error()
	line, column, cleaned := extractTemplatePosition(message)

	if match := undefinedFunctionPattern.FindStringSubmatch(message); match != nil {
		return positionedError(
			CodeUnknownFunction, field,
			fmt.Sprintf(
				"%q is not a function you can call in a template. Available functions: %s",
				match[1], strings.Join(AllowedFunctionNames(), ", "),
			),
			line, column,
		)
	}

	return positionedError(CodeSyntaxError, field, cleaned, line, column)
}
