// Package inboundmail reads received messages back from the provider that
// received them.
//
// Resend's inbound webhook carries a message's metadata and nothing else: the
// body, the headers and the attachments are read from its API afterwards, with
// the mailbox's own key. This is that read.
package inboundmail

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/httpsafe"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultResendBaseURL = "https://api.resend.com"
	// MaxAttachmentBytes is the largest file accepted from a provider. It is
	// the inbound webhook's own body limit, so a file Postmark could deliver
	// inline is one Resend's mail can carry too.
	MaxAttachmentBytes = 32 << 20
	// maxEmailBytes bounds the message itself. Resend inlines images into the
	// HTML as data URIs, so a message with a signature logo is far larger than
	// its text.
	maxEmailBytes     = 40 << 20
	maxListBytes      = 1 << 20
	maxErrorBytes     = 64 << 10
	attachmentPage    = 100
	maxAttachmentPage = 20
	maxRedirects      = 3
	apiTimeout        = 30 * time.Second
	downloadTimeout   = 2 * time.Minute
)

type Params struct {
	fx.In

	Logger *zap.Logger
}

// Options are the fetcher's seams: the base URL and clients a test points at
// its own server.
type Options struct {
	ResendBaseURL  string
	APIClient      *http.Client
	DownloadClient *http.Client
	Logger         *zap.Logger
}

type Fetcher struct {
	resendBaseURL string
	api           *http.Client
	downloads     *http.Client
	l             *zap.Logger
}

func New(p Params) services.InboundContentFetcher {
	return NewFetcher(&Options{Logger: p.Logger})
}

// NewFetcher builds a fetcher. Attachment downloads go through the egress
// guard, because their URLs arrive in a response body rather than from
// configuration; a few redirects are allowed, each re-checked by the same
// guarded dialer.
func NewFetcher(opts *Options) *Fetcher {
	baseURL := strings.TrimRight(strings.TrimSpace(opts.ResendBaseURL), "/")
	if baseURL == "" {
		baseURL = defaultResendBaseURL
	}

	api := opts.APIClient
	if api == nil {
		api = &http.Client{Timeout: apiTimeout}
	}

	downloads := opts.DownloadClient
	if downloads == nil {
		downloads = httpsafe.NewClient(downloadTimeout)
		downloads.CheckRedirect = limitRedirects
	}

	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Fetcher{
		resendBaseURL: baseURL,
		api:           api,
		downloads:     downloads,
		l:             logger.Named("inbound-content-fetcher"),
	}
}

func limitRedirects(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("too many redirects")
	}
	if req.URL.Scheme != "https" {
		return errors.New("refusing a redirect away from https")
	}

	return nil
}

func (f *Fetcher) Fetch(
	ctx context.Context,
	req *services.InboundContentRequest,
) (*services.InboundContent, error) {
	switch req.Provider {
	case inboundmessage.ProviderResend:
		return f.fetchResend(ctx, req)
	case inboundmessage.ProviderPostmark:
		return nil, fmt.Errorf("%w: %s messages arrive whole and are never fetched",
			services.ErrInboundContentRejected, req.Provider)
	default:
		return nil, fmt.Errorf("%w: %s is not a provider messages are fetched from",
			services.ErrInboundContentRejected, req.Provider)
	}
}

type resendEmail struct {
	MessageID   string                     `json:"message_id"`
	Text        *string                    `json:"text"`
	HTML        *string                    `json:"html"`
	Headers     map[string]any             `json:"headers"`
	Attachments []resendAttachmentMetadata `json:"attachments"`
}

type resendAttachmentMetadata struct {
	ID string `json:"id"`
}

type resendAttachmentList struct {
	HasMore bool                   `json:"has_more"`
	Data    []resendAttachmentFile `json:"data"`
}

type resendAttachmentFile struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
}

type resendError struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

func (f *Fetcher) fetchResend(
	ctx context.Context,
	req *services.InboundContentRequest,
) (*services.InboundContent, error) {
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%w: the mailbox has no Resend API key",
			services.ErrInboundContentRejected)
	}
	emailID := strings.TrimSpace(req.ProviderMessageID)
	if emailID == "" {
		return nil, fmt.Errorf("%w: the message carries no Resend email id",
			services.ErrInboundContentRejected)
	}

	email := new(resendEmail)
	if err := f.getResend(
		ctx,
		apiKey,
		"/emails/receiving/"+url.PathEscape(emailID),
		nil,
		maxEmailBytes,
		email,
	); err != nil {
		return nil, err
	}

	content := &services.InboundContent{
		MessageID: strings.TrimSpace(email.MessageID),
	}
	if email.Text != nil {
		content.Text = *email.Text
	}
	if email.HTML != nil {
		content.HTML = *email.HTML
	}
	applyHeaders(content, email.Headers)

	if len(email.Attachments) == 0 {
		return content, nil
	}

	files, err := f.listResendAttachments(ctx, apiKey, emailID)
	if err != nil {
		return nil, err
	}

	content.Attachments = make([]services.InboundContentAttachment, 0, len(files))
	for i := range files {
		attachment, dErr := f.downloadAttachment(ctx, &files[i])
		if dErr != nil {
			return nil, dErr
		}
		content.Attachments = append(content.Attachments, attachment)
	}

	return content, nil
}

func (f *Fetcher) listResendAttachments(
	ctx context.Context,
	apiKey, emailID string,
) ([]resendAttachmentFile, error) {
	path := "/emails/receiving/" + url.PathEscape(emailID) + "/attachments"

	var files []resendAttachmentFile
	after := ""
	for range maxAttachmentPage {
		query := url.Values{"limit": {strconv.Itoa(attachmentPage)}}
		if after != "" {
			query.Set("after", after)
		}

		page := new(resendAttachmentList)
		if err := f.getResend(ctx, apiKey, path, query, maxListBytes, page); err != nil {
			return nil, err
		}
		files = append(files, page.Data...)

		if !page.HasMore || len(page.Data) == 0 {
			return files, nil
		}
		after = page.Data[len(page.Data)-1].ID
	}

	return nil, fmt.Errorf("%w: Resend listed more than %d attachments",
		services.ErrInboundContentRejected, attachmentPage*maxAttachmentPage)
}

func (f *Fetcher) downloadAttachment(
	ctx context.Context,
	file *resendAttachmentFile,
) (services.InboundContentAttachment, error) {
	attachment := services.InboundContentAttachment{
		ProviderID:  file.ID,
		FileName:    file.Filename,
		ContentType: file.ContentType,
	}

	if file.Size > MaxAttachmentBytes {
		attachment.FailureText = tooLargeText()

		return attachment, nil
	}

	target, err := url.Parse(file.DownloadURL)
	if err != nil || target.Scheme != "https" || target.Host == "" {
		attachment.FailureText = "Resend did not give a download address for the file."

		return attachment, nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), http.NoBody)
	if err != nil {
		return attachment, fmt.Errorf("%w: build the attachment download: %w",
			services.ErrInboundContentUnavailable, err)
	}

	resp, err := f.downloads.Do(httpReq)
	if err != nil {
		return attachment, fmt.Errorf("%w: download %q: %w",
			services.ErrInboundContentUnavailable, file.Filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBytes))

		if downloadRefused(resp.StatusCode) {
			attachment.FailureText = fmt.Sprintf(
				"Resend refused to hand over the file (%d).", resp.StatusCode)

			return attachment, nil
		}

		return attachment, fmt.Errorf("%w: downloading %q answered %d",
			services.ErrInboundContentUnavailable, file.Filename, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxAttachmentBytes+1))
	if err != nil {
		return attachment, fmt.Errorf("%w: read %q: %w",
			services.ErrInboundContentUnavailable, file.Filename, err)
	}
	if len(body) > MaxAttachmentBytes {
		attachment.FailureText = tooLargeText()

		return attachment, nil
	}

	attachment.Content = body

	return attachment, nil
}

// downloadRefused is a download answer another attempt will not change. The
// address was listed moments before, so a client error on it is the file being
// gone or withheld, not an expired link; recording it on the file keeps the
// message's body and its other files rather than failing them all. A timeout or
// a rate limit is still retried.
func downloadRefused(status int) bool {
	return status >= 400 && status < 500 &&
		status != http.StatusRequestTimeout && status != http.StatusTooManyRequests
}

func tooLargeText() string {
	return fmt.Sprintf("The file is larger than the %d MB an inbound attachment may be.",
		MaxAttachmentBytes>>20)
}

func (f *Fetcher) getResend(
	ctx context.Context,
	apiKey, path string,
	query url.Values,
	limit int64,
	out any,
) error {
	endpoint := f.resendBaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("%w: build the Resend request: %w",
			services.ErrInboundContentRejected, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := f.api.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: Resend could not be reached: %w",
			services.ErrInboundContentUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return fmt.Errorf("%w: read the Resend response: %w",
			services.ErrInboundContentUnavailable, err)
	}
	if int64(len(body)) > limit {
		return fmt.Errorf("%w: the Resend response is larger than %d MB",
			services.ErrInboundContentRejected, limit>>20)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err = sonic.Unmarshal(body, out); err != nil {
			return fmt.Errorf("%w: the Resend response could not be read: %w",
				services.ErrInboundContentUnavailable, err)
		}

		return nil
	}

	return statusError(resp.StatusCode, body)
}

func statusError(status int, body []byte) error {
	detail := ""
	var parsed resendError
	if sonic.Unmarshal(body, &parsed) == nil {
		detail = strings.TrimSpace(parsed.Message)
	}

	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return fmt.Errorf(
			"%w: Resend refused the mailbox's API key (%d%s); reading received mail needs a full access key",
			services.ErrInboundContentRejected,
			status,
			detailSuffix(detail),
		)
	case status == http.StatusNotFound:
		return fmt.Errorf(
			"%w: Resend has no received email with this id under the mailbox's API key (%d%s)",
			services.ErrInboundContentRejected, status, detailSuffix(detail),
		)
	case status == http.StatusTooManyRequests || status >= 500:
		return fmt.Errorf("%w: Resend answered %d%s",
			services.ErrInboundContentUnavailable, status, detailSuffix(detail))
	default:
		return fmt.Errorf("%w: Resend answered %d%s",
			services.ErrInboundContentRejected, status, detailSuffix(detail))
	}
}

func detailSuffix(detail string) string {
	if detail == "" {
		return ""
	}

	return ": " + detail
}

// applyHeaders reads the threading headers. Resend returns headers as an
// object keyed by lower-case name, with a repeated header as a list.
func applyHeaders(content *services.InboundContent, headers map[string]any) {
	for name, raw := range headers {
		value := headerValue(raw)
		if value == "" {
			continue
		}

		switch strings.ToLower(name) {
		case "in-reply-to":
			content.InReplyTo = value
		case "references":
			content.References = strings.Fields(value)
		case "message-id":
			if content.MessageID == "" {
				content.MessageID = value
			}
		}
	}
}

func headerValue(raw any) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				parts = append(parts, strings.TrimSpace(s))
			}
		}

		return strings.Join(parts, " ")
	default:
		return ""
	}
}
