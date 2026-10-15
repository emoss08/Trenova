package permissionlint

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	clientResourceKey = regexp.MustCompile(`(?m)^\s*(\w+):\s*"([a-z0-9_]+)",?\s*$`)
	clientPairPattern = regexp.MustCompile(
		`Resource\.(\w+)\s*,\s*(?:requiredOperation:\s*)?Operation\.(\w+)`,
	)
)

func LoadClientResources(generatedFile string) (map[string]string, error) {
	data, err := os.ReadFile(generatedFile)
	if err != nil {
		return nil, err
	}
	resources := make(map[string]string)
	for _, match := range clientResourceKey.FindAllStringSubmatch(string(data), -1) {
		resources[match[1]] = match[2]
	}
	return resources, nil
}

func AnalyzeClient(resources map[string]string, dirs ...string) (*Result, error) {
	result := &Result{}
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" || entry.Name() == "generated" {
					return filepath.SkipDir
				}
				return nil
			}
			if !isClientSource(path) {
				return nil
			}
			return analyzeClientFile(result, resources, path)
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func isClientSource(path string) bool {
	if !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx") {
		return false
	}
	return !strings.Contains(path, ".test.") && !strings.Contains(path, "__tests__")
}

func analyzeClientFile(result *Result, resources map[string]string, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	source := string(data)
	for _, match := range clientPairPattern.FindAllStringSubmatchIndex(source, -1) {
		key := source[match[2]:match[3]]
		operation := strings.ToLower(source[match[4]:match[5]])
		line := strings.Count(source[:match[0]], "\n") + 1
		position := filepath.ToSlash(path) + ":" + strconv.Itoa(line)
		resource, ok := resources[key]
		if !ok {
			result.Unresolved = append(result.Unresolved, position+" Resource."+key)
			continue
		}
		result.Pairs = append(result.Pairs, Pair{
			Position:  position,
			Resource:  resource,
			Operation: operation,
		})
	}
	return nil
}
