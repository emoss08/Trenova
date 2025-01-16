package rptmeta

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Parse reads and validates a report metadata YAML file
func Parse(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var metadata Metadata
	err = yaml.Unmarshal(data, &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}
