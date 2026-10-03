package assistantartifact

import (
	"regexp"
	"slices"
	"strconv"
)

// The keys a document's payload holds. A document is markdown; the rest is
// what the Desk sets around it: what kind of write-up it is, who wrote this
// version and from what, and the sources its citation marks point to.
const (
	DocumentFormat      = "format"
	DocumentBody        = "body"
	DocumentType        = "docType"
	DocumentAuthor      = "author"
	DocumentBasis       = "basis"
	DocumentSources     = "sources"
	DocumentCitations   = "citations"
	DocumentEditedBy    = "editedBy"
	DocumentEditor      = "editor"
	DocumentVersionNote = "versionNote"

	// DocumentEditedByAgent and DocumentEditedByPerson say who wrote a
	// version: the agent that published or revised it, or the person who
	// edited, rewrote or restored it on the Desk.
	DocumentEditedByAgent  = "agent"
	DocumentEditedByPerson = "person"

	MaxDocumentBodyBytes = 60000
	MaxDocumentSources   = 30
)

// DocumentSource is what one citation mark in a document points to: the
// tool that found it, what it is, and the artifact to open when there is one.
type DocumentSource struct {
	N          int    `json:"n"`
	Tool       string `json:"tool"`
	Label      string `json:"label"`
	Detail     string `json:"detail,omitempty"`
	ArtifactID string `json:"artifactId,omitempty"`
}

var citationMark = regexp.MustCompile(`\[\^(\d{1,3})\]`)

// CitedNumbers is every source number the body cites, in the order first
// cited, so the Desk can number chips without parsing the body twice and a
// source nothing cites can be told apart from one that is.
func CitedNumbers(body string) []int {
	var out []int
	for _, match := range citationMark.FindAllStringSubmatch(body, -1) {
		n, err := strconv.Atoi(match[1])
		if err != nil || slices.Contains(out, n) {
			continue
		}
		out = append(out, n)
	}

	return out
}

// DocumentVersion builds the payload of the next version of a document: the
// same sources and framing with a new body, credited to whoever wrote it.
func DocumentVersion(
	previous map[string]any,
	body, editedBy, editor, note string,
) map[string]any {
	next := make(map[string]any, len(previous)+4)
	for key, value := range previous {
		next[key] = value
	}
	next[DocumentFormat] = "markdown"
	next[DocumentBody] = body
	next[DocumentCitations] = CitedNumbers(body)
	next[DocumentEditedBy] = editedBy
	next[DocumentEditor] = editor
	next[DocumentVersionNote] = note

	return next
}
