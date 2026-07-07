package as2

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/textproto"
	"slices"
	"strings"
)

const (
	Version = "1.2"

	HeaderAS2From                        = "As2-From"
	HeaderAS2To                          = "As2-To"
	HeaderAS2Version                     = "As2-Version"
	HeaderMessageID                      = "Message-Id"
	HeaderSubject                        = "Subject"
	HeaderDispositionNotificationTo      = "Disposition-Notification-To"
	HeaderDispositionNotificationOptions = "Disposition-Notification-Options"
	HeaderReceiptDeliveryOption          = "Receipt-Delivery-Option"

	contentTypePKCS7MIME       = "application/pkcs7-mime"
	contentTypePKCS7Signature  = "application/pkcs7-signature"
	contentTypeMultipartSigned = "multipart/signed"
	contentTypeEDIX12          = "application/edi-x12"

	smimeTypeEnvelopedData  = "enveloped-data"
	smimeTypeSignedData     = "signed-data"
	smimeTypeCompressedData = "compressed-data"
)

var (
	ErrMissingPayload     = errors.New("as2: payload is required")
	ErrMissingIdentifiers = errors.New("as2: AS2-From and AS2-To identifiers are required")
	ErrSignatureRequired  = errors.New("as2: message is not signed but a signature is required")
	ErrEncryptionRequired = errors.New(
		"as2: message is not encrypted but encryption is required",
	)
	ErrMalformedMultipart   = errors.New("as2: malformed multipart entity")
	ErrMissingSignaturePart = errors.New(
		"as2: multipart/signed entity is missing the signature part",
	)
	ErrUnsupportedContentType = errors.New("as2: unsupported content type")
)

type BuildMessageOptions struct {
	From                  string
	To                    string
	MessageID             string
	Subject               string
	FileName              string
	Payload               []byte
	SigningCertificate    *x509.Certificate
	SigningKey            crypto.PrivateKey
	EncryptionCertificate *x509.Certificate
	SigningAlgorithm      string
	EncryptionAlgorithm   string
	MICAlgorithm          string
	Compress              bool
	RequestMDN            bool
	RequestSignedMDN      bool
	AsyncMDNURL           string
}

type BuiltMessage struct {
	MessageID   string
	MIC         string
	ContentType string
	Body        []byte
	Headers     textproto.MIMEHeader
}

func BuildMessage(opts *BuildMessageOptions) (*BuiltMessage, error) {
	if opts == nil || len(opts.Payload) == 0 {
		return nil, ErrMissingPayload
	}
	if strings.TrimSpace(opts.From) == "" || strings.TrimSpace(opts.To) == "" {
		return nil, ErrMissingIdentifiers
	}
	messageID := opts.MessageID
	if messageID == "" {
		messageID = NewMessageID()
	}

	fileName := opts.FileName
	if fileName == "" {
		fileName = "payload.edi"
	}
	contentType := contentTypeEDIX12
	entity := buildEntity(textproto.MIMEHeader{
		"Content-Type": {contentType},
		"Content-Disposition": {
			mime.FormatMediaType("attachment", map[string]string{"filename": fileName}),
		},
		"Content-Transfer-Encoding": {"binary"},
	}, canonicalizeCRLF(opts.Payload))

	micAlgorithm := normalizeAlgorithm(opts.MICAlgorithm, MICAlgorithmSHA256)
	mic, err := ComputeMIC(entity, micAlgorithm)
	if err != nil {
		return nil, err
	}

	if opts.Compress {
		compressed, compressErr := Compress(entity)
		if compressErr != nil {
			return nil, compressErr
		}
		entity = buildEntity(
			pkcs7MIMEHeader(smimeTypeCompressedData, "smime.p7z"),
			encodeBase64(compressed),
		)
	}

	if opts.SigningCertificate != nil && opts.SigningKey != nil {
		entity, err = buildSignedEntity(entity, opts)
		if err != nil {
			return nil, err
		}
	}

	if opts.EncryptionCertificate != nil {
		ciphertext, encryptErr := Encrypt(
			entity,
			opts.EncryptionCertificate,
			opts.EncryptionAlgorithm,
		)
		if encryptErr != nil {
			return nil, encryptErr
		}
		entity = buildEntity(
			pkcs7MIMEHeader(smimeTypeEnvelopedData, "smime.p7m"),
			encodeBase64(ciphertext),
		)
	}

	headers, body, err := splitEntity(entity)
	if err != nil {
		return nil, err
	}
	built := &BuiltMessage{
		MessageID:   messageID,
		MIC:         mic,
		ContentType: headers.Get("Content-Type"),
		Body:        body,
		Headers:     textproto.MIMEHeader{},
	}
	built.Headers.Set(HeaderAS2Version, Version)
	built.Headers.Set(HeaderAS2From, opts.From)
	built.Headers.Set(HeaderAS2To, opts.To)
	built.Headers.Set(HeaderMessageID, messageID)
	if opts.Subject != "" {
		built.Headers.Set(HeaderSubject, opts.Subject)
	}
	if cte := headers.Get("Content-Transfer-Encoding"); cte != "" {
		built.Headers.Set("Content-Transfer-Encoding", cte)
	}
	if opts.RequestMDN {
		built.Headers.Set(HeaderDispositionNotificationTo, opts.From)
		if opts.RequestSignedMDN {
			built.Headers.Set(
				HeaderDispositionNotificationOptions,
				"signed-receipt-protocol=optional, pkcs7-signature; signed-receipt-micalg=optional, "+
					micAlgorithm,
			)
		}
		if opts.AsyncMDNURL != "" {
			built.Headers.Set(HeaderReceiptDeliveryOption, opts.AsyncMDNURL)
		}
	}
	return built, nil
}

type ParseMessageOptions struct {
	DecryptionCertificate *x509.Certificate
	DecryptionKey         crypto.PrivateKey
	PartnerCertificate    *x509.Certificate
	MICAlgorithm          string
	TransferEncoding      string
	RequireSignature      bool
	RequireEncryption     bool
}

type ParsedMessage struct {
	Payload    []byte
	MIC        string
	FileName   string
	Signed     bool
	Encrypted  bool
	Compressed bool
}

//nolint:funlen,gocognit // The parser walks every S/MIME layering combination in one loop.
func ParseMessage(
	contentType string,
	body []byte,
	opts *ParseMessageOptions,
) (*ParsedMessage, error) {
	if opts == nil {
		opts = &ParseMessageOptions{}
	}
	result := &ParsedMessage{}
	currentType := contentType
	currentBody := body
	currentHeaders := textproto.MIMEHeader{}
	if opts.TransferEncoding != "" {
		currentHeaders.Set("Content-Transfer-Encoding", opts.TransferEncoding)
	}

	for range 8 {
		mediaType, params, err := mime.ParseMediaType(currentType)
		if err != nil {
			return nil, fmt.Errorf("as2: parse content type %q: %w", currentType, err)
		}
		switch {
		case mediaType == contentTypePKCS7MIME &&
			smimeTypeOf(params) == smimeTypeEnvelopedData:
			if opts.DecryptionKey == nil || opts.DecryptionCertificate == nil {
				return nil, errors.New(
					"as2: message is encrypted but no decryption key is configured",
				)
			}
			decoded := decodeTransferEncoding(
				currentBody,
				currentHeaders.Get("Content-Transfer-Encoding"),
			)
			plaintext, decryptErr := Decrypt(
				decoded,
				opts.DecryptionCertificate,
				opts.DecryptionKey,
			)
			if decryptErr != nil {
				return nil, decryptErr
			}
			result.Encrypted = true
			currentHeaders, currentBody, err = splitEntity(plaintext)
			if err != nil {
				return nil, err
			}
			currentType = entityContentType(currentHeaders)
		case mediaType == contentTypeMultipartSigned:
			contentPart, verifyErr := verifySignedMultipart(
				currentBody,
				params["boundary"],
				opts.PartnerCertificate,
			)
			if verifyErr != nil {
				return nil, verifyErr
			}
			result.Signed = true
			result.MIC, err = ComputeMIC(
				contentPart,
				normalizeAlgorithm(opts.MICAlgorithm, micAlgorithmFromParams(params)),
			)
			if err != nil {
				return nil, err
			}
			currentHeaders, currentBody, err = splitEntity(contentPart)
			if err != nil {
				return nil, err
			}
			currentType = entityContentType(currentHeaders)
		case mediaType == contentTypePKCS7MIME &&
			smimeTypeOf(params) == smimeTypeSignedData:
			decoded := decodeTransferEncoding(
				currentBody,
				currentHeaders.Get("Content-Transfer-Encoding"),
			)
			content, verifyErr := verifyEmbeddedSignedData(decoded, opts.PartnerCertificate)
			if verifyErr != nil {
				return nil, verifyErr
			}
			result.Signed = true
			result.MIC, err = ComputeMIC(
				content,
				normalizeAlgorithm(opts.MICAlgorithm, MICAlgorithmSHA256),
			)
			if err != nil {
				return nil, err
			}
			currentHeaders, currentBody, err = splitEntity(content)
			if err != nil {
				return nil, err
			}
			currentType = entityContentType(currentHeaders)
		case mediaType == contentTypePKCS7MIME &&
			smimeTypeOf(params) == smimeTypeCompressedData:
			decoded := decodeTransferEncoding(
				currentBody,
				currentHeaders.Get("Content-Transfer-Encoding"),
			)
			decompressed, decompressErr := Decompress(decoded)
			if decompressErr != nil {
				return nil, decompressErr
			}
			result.Compressed = true
			currentHeaders, currentBody, err = splitEntity(decompressed)
			if err != nil {
				return nil, err
			}
			currentType = entityContentType(currentHeaders)
		default:
			if opts.RequireSignature && !result.Signed {
				return nil, ErrSignatureRequired
			}
			if opts.RequireEncryption && !result.Encrypted {
				return nil, ErrEncryptionRequired
			}
			result.Payload = decodeTransferEncoding(
				currentBody,
				currentHeaders.Get("Content-Transfer-Encoding"),
			)
			result.FileName = entityFileName(currentHeaders)
			if result.MIC == "" {
				mic, micErr := ComputeMIC(
					buildEntity(currentHeaders, currentBody),
					normalizeAlgorithm(opts.MICAlgorithm, MICAlgorithmSHA256),
				)
				if micErr != nil {
					return nil, micErr
				}
				result.MIC = mic
			}
			return result, nil
		}
	}
	return nil, errors.New("as2: message exceeds the maximum S/MIME nesting depth")
}

func NewMessageID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "<trenova-as2@localhost>"
	}
	return "<" + hex.EncodeToString(buffer) + "@trenova.as2>"
}

func buildSignedEntity(entity []byte, opts *BuildMessageOptions) ([]byte, error) {
	signature, err := Sign(entity, opts.SigningCertificate, opts.SigningKey, opts.SigningAlgorithm)
	if err != nil {
		return nil, err
	}
	boundary, err := newBoundary()
	if err != nil {
		return nil, err
	}
	signaturePart := buildEntity(textproto.MIMEHeader{
		"Content-Type": {
			mime.FormatMediaType(contentTypePKCS7Signature, map[string]string{"name": "smime.p7s"}),
		},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition": {
			mime.FormatMediaType("attachment", map[string]string{"filename": "smime.p7s"}),
		},
	}, encodeBase64(signature))

	micalg := "sha-" + strings.TrimPrefix(
		normalizeAlgorithm(opts.SigningAlgorithm, SigningAlgorithmSHA256),
		"sha",
	)
	header := textproto.MIMEHeader{
		"Content-Type": {mime.FormatMediaType(contentTypeMultipartSigned, map[string]string{
			"protocol": contentTypePKCS7Signature,
			"micalg":   micalg,
			"boundary": boundary,
		})},
	}
	var multipartBody bytes.Buffer
	multipartBody.WriteString("--" + boundary + "\r\n")
	multipartBody.Write(entity)
	multipartBody.WriteString("\r\n--" + boundary + "\r\n")
	multipartBody.Write(signaturePart)
	multipartBody.WriteString("\r\n--" + boundary + "--\r\n")
	return buildEntity(header, multipartBody.Bytes()), nil
}

func verifySignedMultipart(
	body []byte,
	boundary string,
	partnerCertificate *x509.Certificate,
) ([]byte, error) {
	if boundary == "" {
		return nil, ErrMalformedMultipart
	}
	parts, err := splitMultipart(body, boundary)
	if err != nil {
		return nil, err
	}
	if len(parts) < 2 {
		return nil, ErrMissingSignaturePart
	}
	contentPart := parts[0]
	signatureHeaders, signatureBody, err := splitEntity(parts[1])
	if err != nil {
		return nil, err
	}
	signature := decodeTransferEncoding(
		signatureBody,
		signatureHeaders.Get("Content-Transfer-Encoding"),
	)
	if err = Verify(contentPart, signature, partnerCertificate); err != nil {
		return nil, err
	}
	return contentPart, nil
}

func verifyEmbeddedSignedData(
	der []byte,
	partnerCertificate *x509.Certificate,
) ([]byte, error) {
	return VerifySignedData(der, partnerCertificate)
}

func smimeTypeOf(params map[string]string) string {
	return strings.ToLower(params["smime-type"])
}

func micAlgorithmFromParams(params map[string]string) string {
	micalg := strings.ToLower(params["micalg"])
	micalg = strings.ReplaceAll(micalg, "sha-", "sha")
	if micalg == "" {
		return MICAlgorithmSHA256
	}
	return micalg
}

func entityContentType(headers textproto.MIMEHeader) string {
	contentType := headers.Get("Content-Type")
	if contentType == "" {
		return "text/plain"
	}
	return contentType
}

func entityFileName(headers textproto.MIMEHeader) string {
	disposition := headers.Get("Content-Disposition")
	if disposition == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		return ""
	}
	return params["filename"]
}

func buildEntity(headers textproto.MIMEHeader, body []byte) []byte {
	var buffer bytes.Buffer
	for _, key := range sortedHeaderKeys(headers) {
		for _, value := range headers[key] {
			buffer.WriteString(key + ": " + value + "\r\n")
		}
	}
	buffer.WriteString("\r\n")
	buffer.Write(body)
	return buffer.Bytes()
}

func splitEntity(entity []byte) (textproto.MIMEHeader, []byte, error) {
	reader := textproto.NewReader(bufio.NewReader(bytes.NewReader(entity)))
	headers, err := reader.ReadMIMEHeader()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, fmt.Errorf("as2: parse MIME entity headers: %w", err)
	}
	body, err := io.ReadAll(reader.R)
	if err != nil {
		return nil, nil, fmt.Errorf("as2: read MIME entity body: %w", err)
	}
	return headers, body, nil
}

func splitMultipart(body []byte, boundary string) ([][]byte, error) {
	delimiter := []byte("--" + boundary)
	segments := bytes.Split(body, delimiter)
	if len(segments) < 3 {
		return nil, ErrMalformedMultipart
	}
	parts := make([][]byte, 0, len(segments)-2)
	for _, segment := range segments[1 : len(segments)-1] {
		parts = append(parts, trimOneLineBreak(segment))
	}
	return parts, nil
}

func trimOneLineBreak(part []byte) []byte {
	switch {
	case bytes.HasPrefix(part, []byte("\r\n")):
		part = part[2:]
	case bytes.HasPrefix(part, []byte("\n")):
		part = part[1:]
	}
	switch {
	case bytes.HasSuffix(part, []byte("\r\n")):
		part = part[:len(part)-2]
	case bytes.HasSuffix(part, []byte("\n")):
		part = part[:len(part)-1]
	}
	return part
}

func decodeTransferEncoding(body []byte, encoding string) []byte {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		cleaned := strings.Map(func(r rune) rune {
			switch r {
			case '\r', '\n', ' ', '\t':
				return -1
			}
			return r
		}, string(body))
		decoded, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return body
		}
		return decoded
	case "quoted-printable":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			return body
		}
		return decoded
	default:
		return body
	}
}

func encodeBase64(content []byte) []byte {
	encoded := base64.StdEncoding.EncodeToString(content)
	var buffer bytes.Buffer
	for len(encoded) > 76 {
		buffer.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	buffer.WriteString(encoded)
	return buffer.Bytes()
}

func canonicalizeCRLF(content []byte) []byte {
	normalized := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(normalized, []byte("\n"), []byte("\r\n"))
}

func pkcs7MIMEHeader(smimeType, fileName string) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Type": {mime.FormatMediaType(contentTypePKCS7MIME, map[string]string{
			"smime-type": smimeType,
			"name":       fileName,
		})},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition": {
			mime.FormatMediaType("attachment", map[string]string{"filename": fileName}),
		},
	}
}

func newBoundary() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("as2: generate multipart boundary: %w", err)
	}
	return "as2-" + hex.EncodeToString(buffer), nil
}

func sortedHeaderKeys(headers textproto.MIMEHeader) []string {
	order := []string{
		"Content-Type",
		"Content-Transfer-Encoding",
		"Content-Disposition",
	}
	keys := make([]string, 0, len(headers))
	for _, key := range order {
		if _, ok := headers[key]; ok {
			keys = append(keys, key)
		}
	}
	for key := range headers {
		if !slices.Contains(order, key) {
			keys = append(keys, key)
		}
	}
	return keys
}
