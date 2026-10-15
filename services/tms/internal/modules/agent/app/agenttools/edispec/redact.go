package edispec

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/shared/maputils"
)

const (
	keyQuote   = "quote"
	keySummary = "summary"
)

var citedLists = []string{ParamValues, ParamRequirements, ParamMappings, ParamGaps}

func RedactQuotes(params map[string]any) map[string]any {
	if params == nil {
		return nil
	}
	redacted := maputils.CloneDeep(params)
	for _, key := range citedLists {
		switch items := redacted[key].(type) {
		case []any:
			for _, item := range items {
				if entry, isMap := item.(map[string]any); isMap {
					redactEntry(key, entry)
				}
			}
		case []map[string]any:
			for _, entry := range items {
				redactEntry(key, entry)
			}
		}
	}
	return redacted
}

func redactEntry(key string, entry map[string]any) {
	if entry == nil {
		return
	}
	if key == ParamGaps {
		redactString(entry, keySummary)
	}
	if citation, cited := entry[keyCitation].(map[string]any); cited {
		redactString(citation, keyQuote)
	}
}

func redactString(entry map[string]any, key string) {
	if text, ok := entry[key].(string); ok {
		entry[key] = edi.RedactSecrets(text)
	}
}
