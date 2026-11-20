package partner

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	fileNameField   = regexp.MustCompile(`\{([^{}]*)\}`)
	fileNameFields  = []string{"set", "control", "group", "sender", "receiver", "date", "time"}
	fileNameLiteral = regexp.MustCompile(`^[A-Za-z0-9._-]*$`)
)

func validateFileName(check *problems, path, pattern string) {
	if pattern == "" {
		check.add(path, "is required")
		return
	}
	for _, match := range fileNameField.FindAllStringSubmatch(pattern, -1) {
		if !slices.Contains(fileNameFields, match[1]) {
			check.add(path, "{%s} is not one of %v", match[1], fileNameFields)
		}
	}
	if !strings.Contains(pattern, "{control}") {
		check.add(path, "needs {control}: two interchanges must never share a file name")
	}
	if literal := fileNameField.ReplaceAllString(pattern, ""); !fileNameLiteral.MatchString(
		literal,
	) {
		check.add(path, "%q may hold only letters, digits, dots, dashes and underscores", literal)
	}
}

func (p *Partner) FileName(written Written) (string, error) {
	if p.Transport.SFTP == nil {
		return "", fmt.Errorf("%s trades over AS2, which names no files", p.Name)
	}
	if len(written.Groups) == 0 || len(written.Groups[0].Transactions) == 0 {
		return "", fmt.Errorf("interchange %d holds no transaction", written.Interchange)
	}
	first := written.Groups[0]
	values := map[string]string{
		"set":      first.Transactions[0].Set,
		"control":  fmt.Sprintf("%09d", written.Interchange),
		"group":    strconv.FormatInt(first.Control, 10),
		"sender":   p.Them.ID,
		"receiver": p.Trenova.ID,
		"date":     written.At.Format("20060102"),
		"time":     written.At.Format("150405"),
	}
	return fileNameField.ReplaceAllStringFunc(p.Transport.SFTP.FileName, func(field string) string {
		return values[strings.Trim(field, "{}")]
	}), nil
}
