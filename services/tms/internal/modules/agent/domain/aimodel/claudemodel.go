package aimodel

import (
	"regexp"
	"strconv"
	"strings"
)

// ClaudeFamily is the line a Claude model belongs to, as its id names it.
type ClaudeFamily string

const (
	ClaudeFamilyOpus   = ClaudeFamily("opus")
	ClaudeFamilySonnet = ClaudeFamily("sonnet")
	ClaudeFamilyHaiku  = ClaudeFamily("haiku")
	ClaudeFamilyFable  = ClaudeFamily("fable")
	ClaudeFamilyMythos = ClaudeFamily("mythos")
)

// ClaudeModel is a Claude model id read into its family and version. Version
// is the major version times ten plus the minor, so claude-opus-4-6 is 46 and
// claude-sonnet-5 is 50.
type ClaudeModel struct {
	Family  ClaudeFamily
	Version int
}

const (
	claudeWindow        = 200_000
	claudeMillionWindow = 1_000_000
)

var (
	// claudeID is family, major and an optional minor, as in claude-opus-5-5
	// or claude-haiku-4-5. The older claude-3-7-sonnet shape does not match.
	claudeID = regexp.MustCompile(`^claude-([a-z]+)-(\d+)(?:-(\d))?$`)
	// claudeDateSuffix is a snapshot date: claude-haiku-4-5-20251001.
	claudeDateSuffix = regexp.MustCompile(`-\d{8}$`)
	// bedrockVersionSuffix is Bedrock's model version: ...-v1:0.
	bedrockVersionSuffix = regexp.MustCompile(`-v\d+(?::\d+)?$`)
)

// ParseClaudeModel reads a Claude model id in whatever form the endpoint that
// serves it uses: the Claude API's plain id, Bedrock's anthropic.-prefixed one
// (with a region in front for an inference profile), Vertex's @date one, a
// dated snapshot, or a gateway's vendor/ prefix. An id it cannot read, such as
// claude-3-7-sonnet or an alias a gateway made up, reports false.
func ParseClaudeModel(model string) (ClaudeModel, bool) {
	id := strings.ToLower(strings.TrimSpace(model))
	if at := strings.IndexByte(id, '@'); at >= 0 {
		id = id[:at]
	}
	start := strings.Index(id, "claude-")
	if start < 0 {
		return ClaudeModel{}, false
	}
	id = bedrockVersionSuffix.ReplaceAllString(id[start:], "")
	id = claudeDateSuffix.ReplaceAllString(id, "")

	match := claudeID.FindStringSubmatch(id)
	if match == nil {
		return ClaudeModel{}, false
	}
	major, err := strconv.Atoi(match[2])
	if err != nil {
		return ClaudeModel{}, false
	}
	minor := 0
	if match[3] != "" {
		minor, err = strconv.Atoi(match[3])
		if err != nil {
			return ClaudeModel{}, false
		}
	}

	return ClaudeModel{Family: ClaudeFamily(match[1]), Version: major*10 + minor}, true
}

// ContextWindow is the model's context window in tokens. Opus and Sonnet from
// 4.6, Haiku from 5.5, and every Fable and Mythos take a million tokens
// without asking; the models before them take 200,000 unless a beta header
// this system does not send unlocks more. A family this does not know gets the
// smaller window, since erring small only compacts a conversation sooner than
// it had to.
func (m ClaudeModel) ContextWindow() int {
	switch m.Family {
	case ClaudeFamilyFable, ClaudeFamilyMythos:
		return claudeMillionWindow
	case ClaudeFamilyOpus, ClaudeFamilySonnet:
		if m.Version >= 46 {
			return claudeMillionWindow
		}
	case ClaudeFamilyHaiku:
		if m.Version >= 55 {
			return claudeMillionWindow
		}
	}

	return claudeWindow
}
