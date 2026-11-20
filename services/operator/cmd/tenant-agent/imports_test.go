package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var forbidden = []string{
	"k8s.io/",
	"sigs.k8s.io/",
	"helm.sh/",
	"github.com/emoss08/trenova/operator/internal/reconcile",
	"github.com/emoss08/trenova/operator/internal/release",
	"github.com/emoss08/trenova/operator/internal/values",
}

func TestTheTenantAgentImportsNoKubernetesOrHelm(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(t.Context(), "go", "list", "-deps", "-f", "{{.ImportPath}}", ".")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	require.NoError(t, err)

	found := make([]string, 0)
	for _, pkg := range strings.Fields(string(out)) {
		for _, prefix := range forbidden {
			if strings.HasPrefix(pkg, prefix) {
				found = append(found, pkg)
			}
		}
	}
	require.Empty(t, found,
		"the tenant agent runs beside Docker Compose on a shared server and must not link "+
			"Kubernetes or Helm; move what it needs into a package that does not import them:\n%s",
		strings.Join(found, "\n"))
}
