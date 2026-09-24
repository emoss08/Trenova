package retrievalservice

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxStructuredFieldDepth = 4
	maxStructuredListItems  = 20
)

func ChunkMemory(memory *agent.Memory) []Chunk {
	content := strings.TrimSpace(memory.Content)
	if content == "" {
		return nil
	}

	var b strings.Builder
	b.Grow(len(content) + 64)
	b.WriteByte('[')
	b.WriteString(string(memory.Kind))
	b.WriteString("] ")
	if about := memory.About(); about != "" {
		b.WriteString(about)
		b.WriteString(": ")
	}
	b.WriteString(content)

	return sealChunks(MemoryChunkerVersion, []Chunk{{Text: b.String(), Body: content}})
}

type documentPage struct {
	number int
	text   string
}

func documentPages(source *repositories.RetrievalDocumentSource) []documentPage {
	pages := make([]documentPage, 0, len(source.Pages))
	for _, page := range source.Pages {
		if text := strings.TrimSpace(page.ExtractedText); text != "" {
			pages = append(pages, documentPage{number: page.PageNumber, text: text})
		}
	}
	slices.SortStableFunc(pages, func(a, b documentPage) int { return a.number - b.number })

	if len(pages) == 0 && source.Content != nil {
		if text := strings.TrimSpace(source.Content.ContentText); text != "" {
			pages = append(pages, documentPage{number: 1, text: text})
		}
	}

	return pages
}

func documentHeader(source *repositories.RetrievalDocumentSource) string {
	doc := source.Document

	var b strings.Builder
	b.WriteString("Document: ")
	b.WriteString(stringutils.FirstNonEmpty(doc.OriginalName, doc.FileName))
	if kind := documentKind(source); kind != "" {
		b.WriteString("\nKind: ")
		b.WriteString(kind)
	}
	if doc.ResourceType != "" {
		b.WriteString("\nAttached to: ")
		b.WriteString(stringutils.HumanizeSnakeCase(
			stringutils.ConvertCamelToSnake(doc.ResourceType),
		))
	}

	return b.String()
}

func documentKind(source *repositories.RetrievalDocumentSource) string {
	if source.Content != nil && source.Content.DetectedDocumentKind != "" {
		return source.Content.DetectedDocumentKind
	}

	return source.Document.DetectedKind
}

func documentPageCount(source *repositories.RetrievalDocumentSource, pages []documentPage) int {
	count := 0
	if source.Content != nil {
		count = source.Content.PageCount
	}
	if len(pages) > 0 {
		count = max(count, pages[len(pages)-1].number)
	}

	return count
}

func ChunkDocument(source *repositories.RetrievalDocumentSource) []Chunk {
	if source == nil || source.Document == nil {
		return nil
	}

	header := documentHeader(source)
	pages := documentPages(source)
	total := documentPageCount(source, pages)
	chunks := make([]Chunk, 0, min(MaxDocumentChunks, len(pages)*2+1))

	if fields := structuredFields(source.Content); fields != "" {
		body := stringutils.TruncateRunes(fields, windowRunes()*maxWindowMultiplier)
		chunks = append(chunks, Chunk{
			Text: header + "\nExtracted fields:\n" + body,
			Body: body,
		})
	}

	for _, page := range pages {
		prefix := header + "\nPage " + strconv.Itoa(page.number)
		if total > 0 {
			prefix += " of " + strconv.Itoa(total)
		}
		for _, window := range Windows(page.text) {
			if len(chunks) == MaxDocumentChunks {
				return sealChunks(DocumentChunkerVersion, chunks)
			}
			chunks = append(chunks, Chunk{
				Page: page.number,
				Text: prefix + "\n\n" + window,
				Body: window,
			})
		}
	}

	return sealChunks(DocumentChunkerVersion, chunks)
}

func structuredFields(content *documentcontent.Content) string {
	if content == nil || len(content.StructuredData) == 0 {
		return ""
	}

	lines := make([]string, 0, len(content.StructuredData))
	lines = appendFields(lines, "", content.StructuredData, 0)

	return strings.Join(lines, "\n")
}

func appendFields(lines []string, prefix string, value any, depth int) []string {
	switch typed := value.(type) {
	case map[string]any:
		if depth >= maxStructuredFieldDepth {
			return lines
		}
		for _, key := range slices.Sorted(maps.Keys(typed)) {
			lines = appendFields(lines, joinFieldPath(prefix, key), typed[key], depth+1)
		}
	case []any:
		if depth >= maxStructuredFieldDepth {
			return lines
		}
		scalars := make([]string, 0, len(typed))
		for idx, item := range typed {
			if idx == maxStructuredListItems {
				break
			}
			if scalar, ok := scalarText(item); ok {
				if scalar != "" {
					scalars = append(scalars, scalar)
				}
				continue
			}
			lines = appendFields(lines, prefix+"["+strconv.Itoa(idx)+"]", item, depth+1)
		}
		if len(scalars) > 0 {
			lines = append(lines, prefix+": "+strings.Join(scalars, ", "))
		}
	default:
		if scalar, ok := scalarText(typed); ok && scalar != "" {
			lines = append(lines, prefix+": "+scalar)
		}
	}

	return lines
}

func joinFieldPath(prefix, key string) string {
	if prefix == "" {
		return key
	}

	return prefix + "." + key
}

func scalarText(value any) (string, bool) {
	switch typed := value.(type) {
	case nil:
		return "", true
	case string:
		return stringutils.CollapseWhitespace(typed), true
	case bool:
		return strconv.FormatBool(typed), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), true
	case int:
		return strconv.Itoa(typed), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case fmt.Stringer:
		return typed.String(), true
	default:
		return "", false
	}
}

func emailSender(message *inboundmessage.InboundMessage) string {
	return stringutils.FormatEmailAddress(message.FromName, message.FromAddress)
}

func ChunkEmail(message *inboundmessage.InboundMessage) []Chunk {
	if message == nil {
		return nil
	}

	header := "Email from " + emailSender(message)
	if subject := strings.TrimSpace(message.Subject); subject != "" {
		header += "\nSubject: " + subject
	}

	windows := Windows(stringutils.MailBody(message.TextBody))
	if len(windows) == 0 {
		return sealChunks(EmailChunkerVersion, []Chunk{{Text: header}})
	}

	chunks := make([]Chunk, 0, min(len(windows), MaxEmailChunks))
	for _, window := range windows {
		if len(chunks) == MaxEmailChunks {
			break
		}
		chunks = append(chunks, Chunk{Text: header + "\n\n" + window, Body: window})
	}

	return sealChunks(EmailChunkerVersion, chunks)
}
