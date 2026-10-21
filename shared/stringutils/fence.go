package stringutils

import (
	"regexp"
	"strings"
	"sync"
)

// closeTagPatterns caches one pattern per fence close tag. A handful of tags
// exist, and every fenced payload asks for one.
var closeTagPatterns = struct {
	mu       sync.RWMutex
	patterns map[string]*regexp.Regexp
}{patterns: make(map[string]*regexp.Regexp)}

// closeTagPattern matches every spelling of closeTag a reader could take for
// it, whatever its case or the whitespace around the slash, with the
// backslashes NeutralizeCloseTag adds captured in the second group:
// "</untrusted_data>", "</UNTRUSTED_DATA >", "< / untrusted_data", and their
// neutralized forms "<\/untrusted_data>", "<\\/untrusted_data>".
func closeTagPattern(closeTag string) *regexp.Regexp {
	closeTagPatterns.mu.RLock()
	pattern, ok := closeTagPatterns.patterns[closeTag]
	closeTagPatterns.mu.RUnlock()
	if ok {
		return pattern
	}

	name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(closeTag, "</"), ">"))
	pattern = regexp.MustCompile(`(?i)(<\s*)(\\*)(\s*/\s*` + regexp.QuoteMeta(name) + `)`)

	closeTagPatterns.mu.Lock()
	closeTagPatterns.patterns[closeTag] = pattern
	closeTagPatterns.mu.Unlock()
	return pattern
}

// NeutralizeCloseTag makes text safe to place inside a fence that closeTag
// ends. Every spelling of the close tag gains one backslash before its slash,
// so none can end the fence; text that already held backslashes there gains
// one more, which is what lets RestoreCloseTag return exactly the original.
func NeutralizeCloseTag(text, closeTag string) string {
	if closeTag == "" || !strings.Contains(text, "<") {
		return text
	}
	return closeTagPattern(closeTag).ReplaceAllString(text, `${1}\${2}${3}`)
}

// RestoreCloseTag undoes NeutralizeCloseTag exactly, for reading the text back.
func RestoreCloseTag(text, closeTag string) string {
	if closeTag == "" || !strings.Contains(text, "<") {
		return text
	}
	return closeTagPattern(closeTag).ReplaceAllStringFunc(text, func(match string) string {
		groups := closeTagPattern(closeTag).FindStringSubmatch(match)
		if len(groups) != 4 || groups[2] == "" {
			return match
		}
		return groups[1] + groups[2][1:] + groups[3]
	})
}
