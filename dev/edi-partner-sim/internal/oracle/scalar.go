package oracle

import (
	"errors"
	"strings"

	"github.com/goccy/go-yaml"
)

type scalar string

func (s *scalar) UnmarshalYAML(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" || text == "~" {
		*s = ""
		return nil
	}
	if text[0] == '"' || text[0] == '\'' {
		var v string
		if err := yaml.Unmarshal(data, &v); err != nil {
			return err
		}
		*s = scalar(v)
		return nil
	}
	if strings.ContainsAny(text, "\n[{") {
		return errors.New("must be a single value")
	}
	*s = scalar(text)
	return nil
}

func scalarsText(values []scalar) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = string(v)
	}
	return strings.Join(parts, " ")
}
