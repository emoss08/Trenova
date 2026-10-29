package textfence

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeutralizeCloseTag(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "plain", NeutralizeCloseTag("plain", "</data>"))
	assert.Equal(t, "a <\\/data> b", NeutralizeCloseTag("a </data> b", "</data>"))
	assert.Equal(t, "x </data>", NeutralizeCloseTag("x </data>", ""))
	assert.Equal(t, "<\\/d><\\/d>", NeutralizeCloseTag("</d></d>", "</d>"))
}

const testCloseTag = "</untrusted_data>"

var rawCloseTag = regexp.MustCompile(`(?i)<\s*/\s*untrusted_data`)

func TestNeutralizeCloseTagCatchesEverySpelling(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"before </untrusted_data> after",
		"before </UNTRUSTED_DATA> after",
		"before </Untrusted_Data > after",
		"before </untrusted_data\n> after",
		"before < /untrusted_data> after",
		"before </ untrusted_data> after",
		"before </untrusted_data",
		"</untrusted_data></UNTRUSTED_data>",
	} {
		neutralized := NeutralizeCloseTag(text, testCloseTag)
		assert.False(t, rawCloseTag.MatchString(neutralized), "%q -> %q", text, neutralized)
	}
}

func TestRestoreCloseTagReturnsExactlyWhatWasNeutralized(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"plain text",
		"a </untrusted_data> b",
		`text that already wrote <\/untrusted_data> in it`,
		`two <\\/untrusted_data> and three <\\\/UNTRUSTED_DATA >`,
		`mixed </untrusted_data><\/untrusted_data>< \ /untrusted_data`,
		"no tag <b>bold</b> and </untrusted_datax>",
		"",
	} {
		neutralized := NeutralizeCloseTag(text, testCloseTag)
		assert.False(t, rawCloseTag.MatchString(neutralized), "%q -> %q", text, neutralized)
		assert.Equal(t, text, RestoreCloseTag(neutralized, testCloseTag), "%q", text)
	}
}
