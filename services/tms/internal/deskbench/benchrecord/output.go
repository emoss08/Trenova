package benchrecord

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	lineLimit = 240
	DirMode   = 0o750
	FileMode  = 0o640
)

func VerdictLabel(call *ToolCallRecord) string {
	if call.Verdict != "" {
		return call.Verdict
	}
	if call.Failed {
		return VerdictFailed
	}
	if !call.Finished {
		return "unfinished"
	}

	return VerdictRan
}

func FirstLine(text string) string {
	return stringutils.Ellipsize(stringutils.CollapseWhitespace(text), lineLimit)
}

func JoinErrors(existing, next string) string {
	if existing == "" {
		return next
	}

	return existing + "; " + next
}

func WriteJSON(dir, name string, value any) error {
	encoded, err := JSON.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}

	return os.WriteFile(filepath.Join(dir, name), encoded, FileMode)
}

var JSON = sonic.Config{SortMapKeys: true, EscapeHTML: false}.Froze()

func UsageLine(usage Usage) string {
	line := fmt.Sprintf("%d model calls, %d in", usage.ModelCalls, usage.InputTokens)
	if usage.CacheReadTokens > 0 || usage.CacheWriteTokens > 0 {
		line += fmt.Sprintf(" + %d cache read + %d cache write", usage.CacheReadTokens,
			usage.CacheWriteTokens)
	}
	line += fmt.Sprintf(" / %d out tokens", usage.OutputTokens)
	if usage.ReasoningTokens > 0 {
		line += " (" + strconv.Itoa(usage.ReasoningTokens) + " thinking)"
	}
	if usage.CostUSD != nil {
		line += ", $" + usage.CostUSD.StringFixed(4)
		if usage.EstimatedCalls > 0 {
			line += "~"
		}
	}
	if usage.UnpricedCalls > 0 {
		line += fmt.Sprintf(" (%d unpriced)", usage.UnpricedCalls)
	}

	return line
}

func CompactJSON(value any, limit int) string {
	if value == nil {
		return "{}"
	}
	encoded, err := JSON.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}

	return stringutils.Ellipsize(string(encoded), limit)
}
