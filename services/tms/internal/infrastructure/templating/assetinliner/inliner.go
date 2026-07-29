// Package assetinliner rewrites the image references in a rendered document so
// nothing remote is ever fetched by the renderer.
//
// The renderer is a browser, and the markup it renders is authored by an
// organization administrator. If that markup could name a URL the browser would
// fetch, a template would become a way to probe the internal network and to signal
// an outside server that a particular invoice was printed. So every asset is
// resolved here, in a process that can reason about tenancy and can refuse, and
// handed to the renderer as a data: URI.
package assetinliner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/imageutils"
	"go.uber.org/zap"
	"golang.org/x/net/html"
)

// remoteFetchTimeout bounds a single outbound asset fetch. It is short on purpose:
// an organization logo served slowly must not hold an invoice render open.
const remoteFetchTimeout = 3 * time.Second

// Limits bound what one document may inline.
type Limits struct {
	// MaxAssetBytes is the largest single decoded image.
	MaxAssetBytes int64
	// MaxAssets is how many images one document may carry.
	MaxAssets int
	// MaxTotalBytes is the combined decoded size across all of them.
	MaxTotalBytes int64
}

func (l Limits) withDefaults() Limits {
	if l.MaxAssetBytes <= 0 {
		l.MaxAssetBytes = 1 << 20
	}
	if l.MaxAssets <= 0 {
		l.MaxAssets = 10
	}
	if l.MaxTotalBytes <= 0 {
		l.MaxTotalBytes = 4 << 20
	}
	return l
}

// ObjectReader is the slice of object storage the inliner needs.
//
// Narrow on purpose: reading one object by key is the entire dependency, and
// depending on the full storage client would mean every test double has to
// implement multipart uploads and presigning to inline a logo. storage.Client
// satisfies this, so wiring is unaffected.
type ObjectReader interface {
	Download(ctx context.Context, key string) (*storage.DownloadResult, error)
}

// Params are the inliner's dependencies.
type Params struct {
	Storage ObjectReader
	Logger  *zap.Logger
	Limits  Limits
}

// Inliner resolves document asset references to data: URIs.
type Inliner struct {
	storage    ObjectReader
	httpClient *http.Client
	transcode  imageutils.WebPTranscoder
	limits     Limits
	l          *zap.Logger
}

func New(p Params) *Inliner {
	return &Inliner{
		storage: p.Storage,
		// httpsafe refuses loopback, private, link-local, and CGNAT addresses and
		// re-checks every resolved IP at dial time, which is what makes a
		// DNS-rebinding attempt on an organization-supplied logo URL fail.
		httpClient: httpsafe.NewClient(remoteFetchTimeout),
		transcode:  NewWebPTranscoder(),
		limits:     p.Limits.withDefaults(),
		l:          p.Logger.Named("templating.assetinliner"),
	}
}

// Request is one document to rewrite.
type Request struct {
	// HTML is rendered output, not template source.
	HTML string
	// Field names the editor pane, so a diagnostic can be routed back to it.
	Field string
}

// Result is the rewritten document plus whatever had to be dropped.
type Result struct {
	HTML  string
	Diags templateengine.Diagnostics
	// Inlined counts the assets that made it in, for logging and metrics.
	Inlined int
	// Bytes is the total decoded size of the inlined assets.
	Bytes int64
}

// Inline rewrites every <img src> in a rendered document.
//
// An asset that cannot be resolved is dropped and reported rather than failing the
// document. A customer invoice must still go out when a decorative image 404s; the
// diagnostic surfaces in the preview and is recorded against the generated
// document so the omission is never silent.
func (i *Inliner) Inline(ctx context.Context, req *Request) (*Result, error) {
	if req == nil || strings.TrimSpace(req.HTML) == "" {
		return &Result{}, nil
	}

	result := &Result{}
	var out strings.Builder
	out.Grow(len(req.HTML))

	tokenizer := html.NewTokenizer(strings.NewReader(req.HTML))

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			if err := tokenizer.Err(); err != nil && !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("scan document for assets: %w", err)
			}
			break
		}

		if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
			out.Write(tokenizer.Raw())
			continue
		}

		name, _ := tokenizer.TagName()
		if !strings.EqualFold(string(name), "img") {
			out.Write(tokenizer.Raw())
			continue
		}

		rewritten, keep := i.rewriteImage(ctx, tokenizer, req.Field, result)
		if !keep {
			continue
		}
		out.WriteString(rewritten)
	}

	result.HTML = out.String()
	return result, nil
}

// rewriteImage resolves one img tag, returning the replacement markup and whether
// to keep the element at all.
func (i *Inliner) rewriteImage(
	ctx context.Context,
	tokenizer *html.Tokenizer,
	field string,
	result *Result,
) (markup string, keep bool) {
	attrs := collectAttributes(tokenizer)

	src, hasSrc := attrs["src"]
	if !hasSrc || strings.TrimSpace(src) == "" {
		// An img with no source is inert; keep it so a layout that reserves space
		// still reserves it.
		return renderImageTag(attrs), true
	}

	if result.Inlined >= i.limits.MaxAssets {
		result.Diags = append(result.Diags, templateengine.Diagnostic{
			Severity: templateengine.SeverityWarning,
			Code:     templateengine.CodeAssetBudgetExceeded,
			Field:    field,
			Message: fmt.Sprintf(
				"this document already inlines %d images, which is the limit; the rest were dropped",
				i.limits.MaxAssets,
			),
		})
		return "", false
	}

	img, err := i.resolve(ctx, src)
	if err != nil {
		result.Diags = append(result.Diags, templateengine.Diagnostic{
			Severity: templateengine.SeverityWarning,
			Code:     templateengine.CodeRemoteAssetBlocked,
			Field:    field,
			Message:  fmt.Sprintf("an image was dropped from this document: %s", err.Error()),
		})
		i.l.Debug("dropped document asset", zap.Error(err))
		return "", false
	}

	if result.Bytes+int64(len(img.Data)) > i.limits.MaxTotalBytes {
		result.Diags = append(result.Diags, templateengine.Diagnostic{
			Severity: templateengine.SeverityWarning,
			Code:     templateengine.CodeAssetBudgetExceeded,
			Field:    field,
			Message:  "this document exceeded its total image size budget; the rest were dropped",
		})
		return "", false
	}

	attrs["src"] = imageutils.ToDataURI(img)
	// Explicit dimensions are what stop a 4000-pixel logo from pushing the rest of
	// the document off the page — the protection the fixed-coordinate renderer used
	// to provide implicitly.
	applyIntrinsicDimensions(attrs, img)

	result.Inlined++
	result.Bytes += int64(len(img.Data))

	return renderImageTag(attrs), true
}

// resolve turns one src value into validated image bytes.
func (i *Inliner) resolve(ctx context.Context, src string) (*imageutils.Image, error) {
	trimmed := strings.TrimSpace(src)

	switch {
	case imageutils.IsDataURI(trimmed):
		// Already inline, but still checked: a data: URI claiming image/png while
		// carrying markup is how an inert image slot becomes a delivery channel.
		return imageutils.DecodeDataURIWith(trimmed, i.limits.MaxAssetBytes, i.transcode)

	case isHTTPURL(trimmed):
		return i.fetchRemote(ctx, trimmed)

	case isDisallowedScheme(trimmed):
		return nil, fmt.Errorf("%q cannot be loaded", schemeOf(trimmed))

	default:
		// Anything else is a storage key. Relative paths and bare filenames land
		// here too, and fail as missing objects rather than reaching the host
		// filesystem.
		return i.fetchStorage(ctx, trimmed)
	}
}

func (i *Inliner) fetchStorage(ctx context.Context, key string) (*imageutils.Image, error) {
	if i.storage == nil {
		return nil, errors.New("object storage is unavailable")
	}

	result, err := i.storage.Download(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read %q from storage: %w", key, err)
	}
	defer func() { _ = result.Body.Close() }()

	body, err := i.readCapped(result.Body)
	if err != nil {
		return nil, err
	}

	return imageutils.NormalizeWith(body, i.transcode)
}

func (i *Inliner) fetchRemote(ctx context.Context, rawURL string) (*imageutils.Image, error) {
	// Validate before dialing so an obviously internal target never opens a socket.
	if _, err := httpsafe.ValidateURL(rawURL); err != nil {
		return nil, fmt.Errorf("refused to fetch %q: %w", rawURL, err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, remoteFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build asset request: %w", err)
	}

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", rawURL, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %q returned status %d", rawURL, resp.StatusCode)
	}

	body, err := i.readCapped(resp.Body)
	if err != nil {
		return nil, err
	}

	return imageutils.NormalizeWith(body, i.transcode)
}

// readCapped reads at most MaxAssetBytes, refusing anything larger rather than
// truncating it into a corrupt image.
func (i *Inliner) readCapped(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, i.limits.MaxAssetBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read asset: %w", err)
	}
	if int64(len(body)) > i.limits.MaxAssetBytes {
		return nil, fmt.Errorf("asset exceeds the %d byte limit", i.limits.MaxAssetBytes)
	}
	return body, nil
}
