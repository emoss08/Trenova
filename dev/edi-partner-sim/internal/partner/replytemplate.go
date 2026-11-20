package partner

import (
	"path/filepath"
	"slices"
	"strings"
)

var ReplySets = []string{"210", "214", "990"}

func isReplyTemplate(p *Partner, path string) bool {
	set := strings.TrimSuffix(filepath.Base(path), yamlExtension)
	return p.Role == RoleCarrier && slices.Contains(ReplySets, set)
}

func CompileEntries(file string, entries []Entry) error {
	check := &problems{file: file}
	compileEntries(check, "segments", entries)
	return check.err()
}

func RenderEntries(entries []Entry, vars map[string]any) ([]Segment, error) {
	r := &renderer{}
	if err := r.entries(entries, &scope{vars: vars}); err != nil {
		return nil, err
	}
	return r.out, nil
}
