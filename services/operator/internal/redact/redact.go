package redact

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	Marker    = "[redacted]"
	minSecret = 8
)

var (
	urlCredentials = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^/\s:@]+:[^@\s/]+@`)
	bearer         = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}`)
	assigned       = regexp.MustCompile(
		`(?i)("?[a-z0-9_.-]*(password|passwd|secret|token|api[_-]?key|authorization|private[_-]?key|credential)[a-z0-9_.-]*"?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;}&]+)`,
	)
)

type Redactor struct {
	values *strings.Replacer
}

type Secrets struct {
	Kube      kubernetes.Interface
	Namespace string
}

func (s Secrets) Load(ctx context.Context) (*Redactor, error) {
	return Load(ctx, s.Kube, s.Namespace)
}

func Load(ctx context.Context, kube kubernetes.Interface, namespace string) (*Redactor, error) {
	secrets, err := kube.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list the namespace's secrets to remove them from the lines: %w", err)
	}
	var raw [][]byte
	for i := range secrets.Items {
		for _, data := range secrets.Items[i].Data {
			raw = append(raw, data)
		}
	}
	return New(raw), nil
}

func New(secrets [][]byte) *Redactor {
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
	for _, data := range secrets {
		add(string(data))
		add(base64.StdEncoding.EncodeToString(data))
		for line := range strings.SplitSeq(string(data), "\n") {
			add(line)
		}
	}
	slices.SortFunc(values, func(a, b string) int {
		if d := len(b) - len(a); d != 0 {
			return d
		}
		return strings.Compare(a, b)
	})
	pairs := make([]string, 0, 2*len(values))
	for _, v := range values {
		pairs = append(pairs, v, Marker)
	}
	return &Redactor{values: strings.NewReplacer(pairs...)}
}

func (r *Redactor) Line(line string) string {
	line = r.values.Replace(line)
	line = urlCredentials.ReplaceAllString(line, "${1}"+Marker+"@")
	line = bearer.ReplaceAllString(line, "${1} "+Marker)
	return assigned.ReplaceAllString(line, "${1}"+Marker)
}
