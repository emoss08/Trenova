package replycheck

import (
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
)

// ArtifactHrefScheme starts the address of a link that names an artifact in
// a reply. The Desk draws such a link as the artifact's badge where the
// sentence names it; anywhere else the link reads as its words.
const ArtifactHrefScheme = "artifact:"

// artifactLink matches a reply's link to an artifact: its words, and the id.
var artifactLink = regexp.MustCompile(
	`\[([^\[\]\n]*)\]\(` + ArtifactHrefScheme + `([A-Za-z0-9_]+)\)`,
)

// ArtifactRef is how the model names an artifact inside a sentence.
func ArtifactRef(shown *agentruntimeports.ShownArtifact) string {
	return artifactRefOf(shown.ID.String(), shown.Title)
}

func artifactRefOf(id, title string) string {
	return "[" + escapeLinkText(title) + "](" + ArtifactHrefScheme + id + ")"
}

// StripArtifactLinks leaves a reply's artifact links as their words, for
// somewhere that cannot open them: a transcript, a search result.
func StripArtifactLinks(text string) string {
	return artifactLink.ReplaceAllString(text, "$1")
}

func NameArtifactLinks(text string, titleOf func(id, words string) (string, bool)) string {
	if !strings.Contains(text, ArtifactHrefScheme) {
		return text
	}
	matches := artifactLink.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var b strings.Builder
	b.Grow(len(text))
	last := 0
	for _, match := range matches {
		b.WriteString(text[last:match[0]])
		words, id := text[match[2]:match[3]], text[match[4]:match[5]]
		if title, known := titleOf(id, words); known {
			b.WriteString(artifactRefOf(id, title))
		} else {
			b.WriteString(words)
		}
		last = match[1]
	}
	b.WriteString(text[last:])

	return b.String()
}

// ArtifactRefNote tells the model how to point at an artifact where its
// sentence mentions it, so the person can open it from the words that
// explain it rather than from a list under the reply.
func ArtifactRefNote(shown *agentruntimeports.ShownArtifact) string {
	ref := ArtifactRef(shown)

	return " To point to it, write " + ref + " as part of the sentence that says what it " +
		"shows, as in \"The details are in " + ref + ".\" It always reads as that title, so write " +
		"it exactly so, once, without adding words such as card or table to it. Never put " +
		"it on a line of its own, in a list, heading or table, or after a sentence that " +
		"has ended. Never make up an id."
}

var linkTextSpecials = regexp.MustCompile(`[\[\]]`)

func escapeLinkText(title string) string {
	return linkTextSpecials.ReplaceAllString(title, "")
}

// ArtifactRefIDs is every artifact a reply points to, by id, in the order the
// reply names them.
func ArtifactRefIDs(text string) []string {
	matches := artifactLink.FindAllStringSubmatch(text, -1)
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match[2])
	}

	return ids
}

// ArtifactLinksIn is every distinct artifact link a reply makes, as written,
// in the order it makes them.
func ArtifactLinksIn(text string) []string {
	matches := artifactLink.FindAllStringSubmatch(text, -1)
	links := make([]string, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, match := range matches {
		if seen[match[2]] {
			continue
		}
		seen[match[2]] = true
		links = append(links, match[0])
	}

	return links
}
