package commands

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	redacted = "[redacted]"
	// minSecret is the shortest secret value removed by value: anything
	// shorter ("true", a port) would blank ordinary words out of every line.
	minSecret = 8
)

// Patterns for secrets a line may carry that no Kubernetes Secret holds:
// credentials in a URL, a bearer token, and a value assigned to a name that
// says it is one.
var (
	urlCredentials = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^/\s:@]+:[^@\s/]+@`)
	bearer         = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}`)
	assigned       = regexp.MustCompile(
		`(?i)("?[a-z0-9_.-]*(password|passwd|secret|token|api[_-]?key|authorization|private[_-]?key|credential)[a-z0-9_.-]*"?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;}&]+)`,
	)
)

// redactor returns a function that removes from a line every value of every
// Secret in the namespace (as stored and as base64), and anything that
// looks like a credential. It reads the secrets once per collection.
func (r *Runner) redactor(ctx context.Context) (func(string) string, error) {
	secrets, err := r.kube.CoreV1().Secrets(r.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		// Without the values to remove, no line may leave.
		return nil, fmt.Errorf("list the namespace's secrets to remove them from the lines: %w", err)
	}
	seen := map[string]struct{}{}
	var values []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if len(v) < minSecret {
			return
		}
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			values = append(values, v)
		}
	}
	for i := range secrets.Items {
		for _, data := range secrets.Items[i].Data {
			add(string(data))
			add(base64.StdEncoding.EncodeToString(data))
			// A multi-line value (a key, a certificate) is quoted a line
			// at a time.
			for line := range strings.SplitSeq(string(data), "\n") {
				add(line)
			}
		}
	}
	// Longest first, so a value is not left half replaced by a shorter one
	// it contains.
	slices.SortFunc(values, func(a, b string) int { return len(b) - len(a) })
	return func(line string) string {
		for _, v := range values {
			line = strings.ReplaceAll(line, v, redacted)
		}
		line = urlCredentials.ReplaceAllString(line, "${1}"+redacted+"@")
		line = bearer.ReplaceAllString(line, "${1} "+redacted)
		line = assigned.ReplaceAllString(line, "${1}"+redacted)
		return line
	}, nil
}
