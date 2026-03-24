package seedhelpers

import (
	"encoding/json" //nolint:depguard // this is fine
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type DataLoader struct {
	basePath string
}

func NewDataLoader(basePath string) *DataLoader {
	return &DataLoader{
		basePath: basePath,
	}
}

func (dl *DataLoader) LoadYAML(filename string, dest any) error {
	fullPath := filepath.Join(dl.basePath, filename)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", fullPath, err)
	}

	var raw any
	if err = yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal YAML from %s: %w", filename, err)
	}

	if raw == nil {
		return nil
	}

	jsonBytes, err := json.Marshal(convertKeysToStrings(raw))
	if err != nil {
		return fmt.Errorf("convert YAML to JSON from %s: %w", filename, err)
	}

	if err = json.Unmarshal(jsonBytes, dest); err != nil {
		return fmt.Errorf("unmarshal YAML from %s: %w", filename, err)
	}

	return nil
}

func convertKeysToStrings(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v := range val {
			out[k] = convertKeysToStrings(v)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, v := range val {
			out[i] = convertKeysToStrings(v)
		}
		return out
	default:
		return v
	}
}

func (dl *DataLoader) LoadJSON(filename string, dest any) error {
	fullPath := filepath.Join(dl.basePath, filename)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", fullPath, err)
	}

	if err = json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal JSON from %s: %w", filename, err)
	}

	return nil
}

func (dl *DataLoader) FileExists(filename string) bool {
	fullPath := filepath.Join(dl.basePath, filename)
	_, err := os.Stat(fullPath)
	return err == nil
}

func (dl *DataLoader) BasePath() string {
	return dl.basePath
}
