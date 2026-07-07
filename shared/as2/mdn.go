package as2

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"
	"mime"
	"net/textproto"
	"strings"
)

const (
	contentTypeMultipartReport         = "multipart/report"
	contentTypeDispositionNotification = "message/disposition-notification"
	DispositionProcessed               = "automatic-action/MDN-sent-automatically; processed"
	dispositionProcessedErrorPrefix    = "automatic-action/MDN-sent-automatically; processed/error: "
	mdnFieldOriginalMessageID          = "Original-Message-Id"
	mdnFieldDisposition                = "Disposition"
	mdnFieldReceivedContentMIC         = "Received-Content-Mic"
)

var (
	ErrNotAnMDN            = errors.New("as2: entity is not a disposition notification")
	ErrMissingMDNPart      = errors.New("as2: multipart/report is missing the disposition part")
	errMDNDispositionEmpty = errors.New("as2: disposition notification has no disposition field")
)

type BuildMDNOptions struct {
	From               string
	To                 string
	OriginalMessageID  string
	ReceivedContentMIC string
	Text               string
	ErrorText          string
	SigningCertificate *x509.Certificate
	SigningKey         crypto.PrivateKey
	SigningAlgorithm   string
}

type BuiltMDN struct {
	MessageID   string
	ContentType string
	Body        []byte
	Headers     textproto.MIMEHeader
}

func BuildMDN(opts *BuildMDNOptions) (*BuiltMDN, error) {
	if opts == nil || strings.TrimSpace(opts.From) == "" || strings.TrimSpace(opts.To) == "" {
		return nil, ErrMissingIdentifiers
	}
	disposition := DispositionProcessed
	text := opts.Text
	if opts.ErrorText != "" {
		disposition = dispositionProcessedErrorPrefix + sanitizeHeaderValue(opts.ErrorText)
		if text == "" {
			text = "The AS2 message could not be processed: " + opts.ErrorText
		}
	} else if text == "" {
		text = "The AS2 message was received and processed successfully."
	}

	var notification bytes.Buffer
	notification.WriteString("Reporting-UA: Trenova AS2\r\n")
	notification.WriteString("Original-Recipient: rfc822; " + opts.From + "\r\n")
	notification.WriteString("Final-Recipient: rfc822; " + opts.From + "\r\n")
	if opts.OriginalMessageID != "" {
		notification.WriteString("Original-Message-ID: " + opts.OriginalMessageID + "\r\n")
	}
	notification.WriteString("Disposition: " + disposition + "\r\n")
	if opts.ReceivedContentMIC != "" {
		notification.WriteString("Received-Content-MIC: " + opts.ReceivedContentMIC + "\r\n")
	}

	boundary, err := newBoundary()
	if err != nil {
		return nil, err
	}
	textPart := buildEntity(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=us-ascii"},
	}, []byte(text+"\r\n"))
	notificationPart := buildEntity(textproto.MIMEHeader{
		"Content-Type": {contentTypeDispositionNotification},
	}, notification.Bytes())

	var reportBody bytes.Buffer
	reportBody.WriteString("--" + boundary + "\r\n")
	reportBody.Write(textPart)
	reportBody.WriteString("\r\n--" + boundary + "\r\n")
	reportBody.Write(notificationPart)
	reportBody.WriteString("\r\n--" + boundary + "--\r\n")

	entity := buildEntity(textproto.MIMEHeader{
		"Content-Type": {mime.FormatMediaType(contentTypeMultipartReport, map[string]string{
			"report-type": "disposition-notification",
			"boundary":    boundary,
		})},
	}, reportBody.Bytes())

	if opts.SigningCertificate != nil && opts.SigningKey != nil {
		entity, err = buildSignedEntity(entity, &BuildMessageOptions{
			SigningCertificate: opts.SigningCertificate,
			SigningKey:         opts.SigningKey,
			SigningAlgorithm:   opts.SigningAlgorithm,
		})
		if err != nil {
			return nil, err
		}
	}

	headers, body, err := splitEntity(entity)
	if err != nil {
		return nil, err
	}
	built := &BuiltMDN{
		MessageID:   NewMessageID(),
		ContentType: headers.Get("Content-Type"),
		Body:        body,
		Headers:     textproto.MIMEHeader{},
	}
	built.Headers.Set(HeaderAS2Version, Version)
	built.Headers.Set(HeaderAS2From, opts.From)
	built.Headers.Set(HeaderAS2To, opts.To)
	built.Headers.Set(HeaderMessageID, built.MessageID)
	return built, nil
}

type ParsedMDN struct {
	OriginalMessageID  string
	Disposition        string
	ReceivedContentMIC string
	Signed             bool
	Text               string
}

func (m *ParsedMDN) Processed() bool {
	disposition := strings.ToLower(m.Disposition)
	return strings.Contains(disposition, "processed") &&
		!strings.Contains(disposition, "error") &&
		!strings.Contains(disposition, "failed") &&
		!strings.Contains(disposition, "failure")
}

func (m *ParsedMDN) FailureText() string {
	_, detail, found := strings.Cut(m.Disposition, ":")
	if !found {
		return ""
	}
	return strings.TrimSpace(detail)
}

func ParseMDN(
	contentType string,
	body []byte,
	partnerCertificate *x509.Certificate,
) (*ParsedMDN, error) {
	result := &ParsedMDN{}
	currentType := contentType
	currentBody := body

	mediaType, params, err := mime.ParseMediaType(currentType)
	if err != nil {
		return nil, fmt.Errorf("as2: parse MDN content type: %w", err)
	}
	if mediaType == contentTypeMultipartSigned {
		contentPart, verifyErr := verifySignedMultipart(
			currentBody,
			params["boundary"],
			partnerCertificate,
		)
		if verifyErr != nil {
			return nil, verifyErr
		}
		result.Signed = true
		headers, innerBody, splitErr := splitEntity(contentPart)
		if splitErr != nil {
			return nil, splitErr
		}
		currentType = entityContentType(headers)
		currentBody = innerBody
		mediaType, params, err = mime.ParseMediaType(currentType)
		if err != nil {
			return nil, fmt.Errorf("as2: parse signed MDN content type: %w", err)
		}
	}

	switch mediaType {
	case contentTypeMultipartReport:
		if err = parseMDNReport(result, currentBody, params["boundary"]); err != nil {
			return nil, err
		}
	case contentTypeDispositionNotification:
		parseMDNFields(result, currentBody)
	default:
		return nil, ErrNotAnMDN
	}
	if result.Disposition == "" {
		return nil, errMDNDispositionEmpty
	}
	return result, nil
}

func IsMDNContentType(contentType string) bool {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	switch mediaType {
	case contentTypeMultipartReport:
		return strings.EqualFold(params["report-type"], "disposition-notification")
	case contentTypeDispositionNotification:
		return true
	default:
		return false
	}
}

func parseMDNReport(result *ParsedMDN, body []byte, boundary string) error {
	if boundary == "" {
		return ErrMalformedMultipart
	}
	parts, err := splitMultipart(body, boundary)
	if err != nil {
		return err
	}
	found := false
	for _, part := range parts {
		headers, partBody, splitErr := splitEntity(part)
		if splitErr != nil {
			continue
		}
		mediaType, _, typeErr := mime.ParseMediaType(entityContentType(headers))
		if typeErr != nil {
			continue
		}
		switch mediaType {
		case contentTypeDispositionNotification:
			parseMDNFields(result, decodeTransferEncoding(
				partBody,
				headers.Get("Content-Transfer-Encoding"),
			))
			found = true
		case "text/plain":
			result.Text = strings.TrimSpace(string(partBody))
		}
	}
	if !found {
		return ErrMissingMDNPart
	}
	return nil
}

func parseMDNFields(result *ParsedMDN, body []byte) {
	headers, _, err := splitEntity(append(bytes.TrimLeft(body, "\r\n"), "\r\n\r\n"...))
	if err != nil {
		return
	}
	result.OriginalMessageID = headers.Get(mdnFieldOriginalMessageID)
	result.Disposition = headers.Get(mdnFieldDisposition)
	result.ReceivedContentMIC = headers.Get(mdnFieldReceivedContentMIC)
}

func sanitizeHeaderValue(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}
