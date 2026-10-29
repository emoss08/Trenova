package anthropicadapter

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmessage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aimodel"
)

// Why a file went as its text rather than as itself.
const (
	FileFallbackType        = "type"
	FileFallbackTooLarge    = "too_large"
	FileFallbackTooManyPage = "too_many_pages"
	FileFallbackUnreadable  = "unreadable"
	FileFallbackUnavailable = "unavailable"
	FileFallbackRefused     = "refused"
)

// anthropicFiles is the conversation with its files attached: the messages
// with each sent file's entry swapped for the one that says the file itself
// is attached, and the blocks that carry the files, by message.
type anthropicFiles struct {
	messages []modeladapter.Message
	blocks   map[int][]anthropic.BetaContentBlockParamUnion
	report   modeladapter.FileReport
}

// attachFiles reads the files the conversation's messages carry and attaches
// each one the model can take as itself. A file it cannot take, or that cannot
// be read, is left as its text, which is how every model read it before.
func attachFiles(ctx context.Context, call *modeladapter.Call) anthropicFiles {
	attached := anthropicFiles{messages: call.Request.Messages}
	input, reads := call.Provider.FileInput()
	if !reads || call.Files == nil || !carriesFiles(call.Request.Messages) {
		return attached
	}

	attached.messages = slices.Clone(call.Request.Messages)
	attached.blocks = make(map[int][]anthropic.BetaContentBlockParamUnion, 1)
	budget := input.MaxTotalBytes
	for idx := range attached.messages {
		msg := &attached.messages[idx]
		if msg.Role != modeladapter.RoleUser || len(msg.Files) == 0 {
			continue
		}
		sent := make([]*modeladapter.MessageFile, 0, len(msg.Files))
		for fileIdx := range msg.Files {
			file := &msg.Files[fileIdx]
			blocks, reason := fileBlocks(ctx, call, &input, file, budget)
			if reason != "" {
				attached.report.Fallbacks = append(attached.report.Fallbacks,
					modeladapter.FileFallback{DocumentID: file.DocumentID, Reason: reason})
				continue
			}
			budget -= blocks.size
			attached.blocks[idx] = append(attached.blocks[idx], blocks.blocks...)
			attached.report.Sent = append(attached.report.Sent, file.DocumentID)
			sent = append(sent, file)
		}
		msg.Content = swapEntries(msg.Content, sent)
	}

	return attached
}

// swapEntries replaces each sent file's entry in the content with the one that
// says the file itself is attached. The last entry is swapped first, so the
// offsets of those before it still hold. An entry whose span no longer names
// its file, because the content changed after the turn opened, is left: the
// model is then sent the file and its text, which is only longer.
func swapEntries(content string, sent []*modeladapter.MessageFile) string {
	ordered := slices.Clone(sent)
	slices.SortFunc(
		ordered,
		func(a, b *modeladapter.MessageFile) int { return b.EntryStart - a.EntryStart },
	)
	for _, file := range ordered {
		start, end := file.EntryStart, file.EntryEnd
		lead := agentmessage.AttachmentEntryLead(file.DocumentID)
		if start < 0 || end <= start || end > len(content) ||
			!strings.HasPrefix(content[start:end], lead) {
			continue
		}
		content = content[:start] + file.NativeEntry + content[end:]
	}

	return content
}

func carriesFiles(messages []modeladapter.Message) bool {
	return slices.ContainsFunc(
		messages,
		func(msg modeladapter.Message) bool { return len(msg.Files) > 0 },
	)
}

type fileBlockSet struct {
	blocks []anthropic.BetaContentBlockParamUnion
	size   int64
}

// fileBlocks reads one file and builds the blocks that carry it, or says why
// it goes as its text. What the turn knew of the file is checked first, so a
// file plainly too large is never read; what was read is checked again.
func fileBlocks(
	ctx context.Context,
	call *modeladapter.Call,
	input *aimodel.FileInput,
	file *modeladapter.MessageFile,
	budget int64,
) (set fileBlockSet, fallback string) {
	declared := input.Kind(file.ContentType)
	if declared == aimodel.FileKindNone {
		return fileBlockSet{}, FileFallbackType
	}
	if reason := fileOverLimit(input, declared, file.Size, file.PageCount, budget); reason != "" {
		return fileBlockSet{}, reason
	}

	content, err := call.Files(ctx, file.DocumentID)
	if err != nil || content == nil || len(content.Data) == 0 {
		return fileBlockSet{}, FileFallbackUnavailable
	}
	contentType := content.ContentType
	if contentType == "" {
		contentType = file.ContentType
	}
	kind := input.Kind(contentType)
	if kind == aimodel.FileKindNone {
		return fileBlockSet{}, FileFallbackType
	}
	if content.Unreadable {
		return fileBlockSet{}, FileFallbackUnreadable
	}
	size := int64(len(content.Data))
	if reason := fileOverLimit(input, kind, size, content.PageCount, budget); reason != "" {
		return fileBlockSet{}, reason
	}

	data := base64.StdEncoding.EncodeToString(content.Data)
	if kind == aimodel.FileKindDocument {
		return fileBlockSet{size: size, blocks: []anthropic.BetaContentBlockParamUnion{{
			OfDocument: &anthropic.BetaRequestDocumentBlockParam{
				Source: anthropic.BetaRequestDocumentBlockSourceUnionParam{
					OfBase64: &anthropic.BetaBase64PDFSourceParam{Data: data},
				},
				Title:   param.NewOpt(fileTitle(file)),
				Context: param.NewOpt(fileContext(file)),
			},
		}}}, ""
	}

	// An image carries no title, so a line naming it goes first: with
	// several attached, the model can tell which is which.
	label := fileContext(file) + ", " + fileTitle(file) + ":"
	mediaType := anthropic.BetaBase64ImageSourceMediaType(aimodel.MediaType(contentType))

	return fileBlockSet{size: size, blocks: []anthropic.BetaContentBlockParamUnion{
		{OfText: &anthropic.BetaTextBlockParam{Text: label}},
		{OfImage: &anthropic.BetaImageBlockParam{
			Source: anthropic.BetaImageBlockParamSourceUnion{
				OfBase64: &anthropic.BetaBase64ImageSourceParam{Data: data, MediaType: mediaType},
			},
		}},
	}}, ""
}

// fileOverLimit says why a file of this size and length cannot go as itself,
// or nothing when it can. A size or page count of zero is not known yet.
func fileOverLimit(
	input *aimodel.FileInput,
	kind aimodel.FileKind,
	size int64,
	pages int,
	budget int64,
) string {
	switch {
	case size > budget:
		return FileFallbackTooLarge
	case kind == aimodel.FileKindImage && size > input.MaxImageBytes:
		return FileFallbackTooLarge
	case kind == aimodel.FileKindDocument && pages > input.MaxPDFPages:
		return FileFallbackTooManyPage
	default:
		return ""
	}
}

func fileTitle(file *modeladapter.MessageFile) string {
	if name := strings.TrimSpace(file.FileName); name != "" {
		return name
	}

	return file.DocumentID
}

func fileContext(file *modeladapter.MessageFile) string {
	return "Attached file " + file.DocumentID
}

// refusedFiles is the report of a request sent again without its files after
// the provider refused it with them: every file went as its text.
func refusedFiles(report modeladapter.FileReport) modeladapter.FileReport {
	fallbacks := slices.Clone(report.Fallbacks)
	for _, id := range report.Sent {
		fallbacks = append(
			fallbacks,
			modeladapter.FileFallback{DocumentID: id, Reason: FileFallbackRefused},
		)
	}

	return modeladapter.FileReport{Fallbacks: fallbacks}
}

// refusedOverFiles reports a request with files the provider refused as
// malformed or too large. A damaged file, or more than the model takes, is
// refused this way, and the conversation still goes through as text.
func refusedOverFiles(err error, attached *anthropicFiles) bool {
	if err == nil || len(attached.report.Sent) == 0 {
		return false
	}
	var transport *modeladapter.TransportError
	if !errors.As(err, &transport) {
		return false
	}

	return transport.StatusCode == http.StatusBadRequest ||
		transport.StatusCode == http.StatusRequestEntityTooLarge
}

// plainConversation is the call's conversation with no file attached, for a
// request sent again after the provider refused its files.
func plainConversation(call *modeladapter.Call, attached *anthropicFiles) anthropicFiles {
	return anthropicFiles{messages: call.Request.Messages, report: refusedFiles(attached.report)}
}
