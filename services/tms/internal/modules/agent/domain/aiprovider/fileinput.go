package aiprovider

import "strings"

// FileKind is how a model is sent a file: as a document, as an image, or not
// at all, in which case it reads the text read from the file.
type FileKind string

const (
	FileKindNone     = FileKind("")
	FileKindDocument = FileKind("document")
	FileKindImage    = FileKind("image")
)

const (
	// claudeImageBytes is the most an image may be, decoded. The API takes 5
	// MB of base64 per image, which is 3.75 MB of image.
	claudeImageBytes = 3_750_000
	// claudeFileBytes is the most every file of one request may come to,
	// decoded. A request may be 32 MB; base64 adds a third, and the rest of
	// the conversation needs room beside the files.
	claudeFileBytes = 20_000_000
	// claudePDFPages and claudeLongPDFPages are the most pages a PDF may
	// have on a model with a 200,000-token window and on a million-token one.
	claudePDFPages     = 100
	claudeLongPDFPages = 600
)

// FileInput is what a provider's model takes as files itself.
type FileInput struct {
	MaxPDFPages   int
	MaxImageBytes int64
	// MaxTotalBytes bounds every file of one request together.
	MaxTotalBytes int64
}

// FileInput reports whether the provider's model reads files itself, and the
// bounds it reads them within. Only an Anthropic Messages provider serving a
// model this can read as Claude does: the same protocol also fronts models
// from other vendors, behind gateways and compatible endpoints, that refuse a
// document block, and a model that refuses one fails the whole request.
func (p *Provider) FileInput() (FileInput, bool) {
	if p.Kind != KindAnthropicMessages {
		return FileInput{}, false
	}
	claude, ok := ParseClaudeModel(p.Model)
	if !ok {
		return FileInput{}, false
	}

	pages := claudePDFPages
	if claude.ContextWindow() >= claudeMillionWindow {
		pages = claudeLongPDFPages
	}

	return FileInput{
		MaxPDFPages:   pages,
		MaxImageBytes: claudeImageBytes,
		MaxTotalBytes: claudeFileBytes,
	}, true
}

// Kind is how a file of the content type is sent: PDFs as documents, and the
// four image formats Claude reads as images.
func (FileInput) Kind(contentType string) FileKind {
	switch MediaType(contentType) {
	case "application/pdf":
		return FileKindDocument
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return FileKindImage
	default:
		return FileKindNone
	}
}

// MediaType is the content type as the API names it, without parameters.
func MediaType(contentType string) string {
	mediaType, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(contentType)), ";")

	return strings.TrimSpace(mediaType)
}
