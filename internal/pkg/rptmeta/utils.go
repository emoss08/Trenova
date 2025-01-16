package rptmeta

import (
	"os"
	"path/filepath"
	"strings"
)

func LoadReportsFromDirectory(dir string) ([]*Metadata, error) {
	var reports []*Metadata

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".rptmeta.yaml") {
			report, rErr := Parse(filepath.Join(
				dir,
				file.Name(),
			))
			if rErr != nil {
				return nil, rErr
			}
			reports = append(reports, report)
		}
	}

	return reports, nil
}

func GetCurrentWorkingDirectory() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return dir, nil
}
