package agentruntime

import (
	"regexp"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

// ArtifactHrefScheme starts the address of a link that names an artifact in
// a reply. The Desk draws such a link as the artifact's badge where the
// sentence names it; anywhere else the link reads as its words.
const ArtifactHrefScheme = "artifact:"

// artifactLink matches a reply's link to an artifact: its words, and the id.
var artifactLink = regexp.MustCompile(`\[([^\[\]\n]+)\]\(` + ArtifactHrefScheme + `([A-Za-z0-9_]+)\)`)

// ArtifactRef is how the model names an artifact inside a sentence.
func ArtifactRef(shown *services.ShownArtifact) string {
	return "[" + escapeLinkText(shown.Title) + "](" + ArtifactHrefScheme + shown.ID.String() + ")"
}

// StripArtifactLinks leaves a reply's artifact links as their words, for
// somewhere that cannot open them: a transcript, a search result.
func StripArtifactLinks(text string) string {
	return artifactLink.ReplaceAllString(text, "$1")
}

// artifactRefNote tells the model how to point at an artifact where its
// sentence mentions it, so the person can open it from the words that
// explain it rather than from a list under the reply.
func artifactRefNote(shown *services.ShownArtifact) string {
	return " To point to it, write " + ArtifactRef(shown) +
		" inside the sentence that mentions it, exactly as written, never on a line of its " +
		"own or after a table; it is shown as a button that opens it. Use it once, and never " +
		"make up an id."
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
