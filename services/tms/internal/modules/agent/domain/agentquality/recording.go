package agentquality

import (
	"strings"

	"github.com/bytedance/sonic"
)

type recordedRead struct {
	result any
	failed bool
}

type Recording struct {
	reads map[string]recordedRead
}

func (c *EvalCase) Recording() *Recording {
	reads := make(map[string]recordedRead, len(c.ToolFixtures))
	for _, fixture := range c.ToolFixtures {
		if fixture.Result == nil || IsRuntimeTool(fixture.Tool) ||
			holdsPlaceholder(fixture.Result) || holdsPlaceholder(fixture.Args) {
			continue
		}
		key, ok := recordingKey(fixture.Tool, fixture.Args)
		if !ok {
			continue
		}
		if _, answered := reads[key]; answered {
			continue
		}
		reads[key] = recordedRead{result: fixture.Result, failed: fixture.Failed}
	}
	if len(reads) == 0 {
		return nil
	}

	return &Recording{reads: reads}
}

func (r *Recording) Recorded(
	tool string,
	arguments map[string]any,
) (result any, failed, ok bool) {
	if r == nil {
		return nil, false, false
	}
	key, keyed := recordingKey(tool, arguments)
	if !keyed {
		return nil, false, false
	}

	read, ok := r.reads[key]
	if !ok {
		return nil, false, false
	}

	return copied(read.result), read.failed, true
}

func recordingKey(tool string, arguments map[string]any) (string, bool) {
	if strings.TrimSpace(tool) == "" {
		return "", false
	}
	if len(arguments) == 0 {
		return tool + "\n{}", true
	}
	encoded, err := sonic.ConfigStd.MarshalToString(arguments)
	if err != nil {
		return "", false
	}

	return tool + "\n" + encoded, true
}

func copied(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			out[key] = copied(nested)
		}

		return out
	case []any:
		out := make([]any, len(typed))
		for idx, nested := range typed {
			out[idx] = copied(nested)
		}

		return out
	default:
		return value
	}
}

func holdsPlaceholder(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.Contains(typed, RestrictedPlaceholderPrefix)
	case map[string]any:
		for _, nested := range typed {
			if holdsPlaceholder(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if holdsPlaceholder(nested) {
				return true
			}
		}
	}

	return false
}
