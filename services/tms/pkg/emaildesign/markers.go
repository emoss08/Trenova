package emaildesign

import "strings"

type marker struct {
	open  string
	close string
}

var (
	preheaderMarker = marker{open: "<!--email:preheader-->", close: "<!--/email:preheader-->"}
	footerMarker    = marker{open: "<!--email:footer-->", close: "<!--/email:footer-->"}
	bandMarker      = marker{open: "<!--email:band-->", close: "<!--/email:band-->"}
)

func (m marker) wrap(inner string) string {
	return m.open + inner + m.close
}

func (m marker) count(content string) int {
	return strings.Count(content, m.open)
}

func (m marker) lift(content string) (first, rest string) {
	var out strings.Builder
	out.Grow(len(content))
	found := false
	for {
		start := strings.Index(content, m.open)
		if start < 0 {
			out.WriteString(content)
			break
		}
		end := strings.Index(content[start:], m.close)
		if end < 0 {
			out.WriteString(content)
			break
		}
		out.WriteString(content[:start])
		if !found {
			first = content[start+len(m.open) : start+end]
			found = true
		}
		content = content[start+end+len(m.close):]
	}
	return first, out.String()
}

type segment struct {
	band bool
	html string
}

func splitBands(content string) []segment {
	segments := make([]segment, 0, 3)
	for {
		start := strings.Index(content, bandMarker.open)
		if start < 0 {
			break
		}
		end := strings.Index(content[start:], bandMarker.close)
		if end < 0 {
			break
		}
		if flow := strings.TrimSpace(content[:start]); flow != "" {
			segments = append(segments, segment{html: flow})
		}
		segments = append(segments, segment{
			band: true,
			html: content[start+len(bandMarker.open) : start+end],
		})
		content = content[start+end+len(bandMarker.close):]
	}
	if flow := strings.TrimSpace(content); flow != "" {
		segments = append(segments, segment{html: flow})
	}
	return segments
}
