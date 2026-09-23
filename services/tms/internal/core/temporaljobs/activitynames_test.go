package temporaljobs_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

/*
Two workers that share a task queue share Temporal's activity registry, which
is keyed by method name across the whole queue. Both the watchtower and the
briefing worker sit on system-queue and both called their nightly sweep
RetentionActivity, so the second one to register took the process down with a
panic at startup — after the graph had resolved, after the database was
connected, and with a message that named the activity but neither worker.

Nothing before startup could catch it: each package compiles, each is tested,
and the collision only exists in the pair. So the property is asserted here
over the source itself. Reading the tree rather than the built registries is
deliberate: a package added tomorrow is covered without being listed anywhere,
which is exactly how this one slipped in.
*/

var (
	taskQueuePattern = regexp.MustCompile(`TaskQueue:\s*(.+?),`)
	jobsRoot         = filepath.Join("..", "..", "core", "temporaljobs")
)

type activityOwner struct {
	pkg      string
	activity string
}

func TestActivityNamesDoNotCollideOnAQueue(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(filepath.Join("..", "temporaljobs"))
	require.NoError(t, err)

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	byQueue := map[string][]activityOwner{}
	packages := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		queue, ok := taskQueueOf(t, dir)
		if !ok {
			continue
		}
		packages++
		for _, activity := range activityMethodsIn(t, dir) {
			byQueue[queue] = append(
				byQueue[queue],
				activityOwner{pkg: entry.Name(), activity: activity},
			)
		}
	}

	require.Greater(t, packages, 10, "no job packages were scanned; the layout must have moved")

	for queue, owners := range byQueue {
		holders := map[string]string{}
		for _, owner := range owners {
			previous, taken := holders[owner.activity]
			require.Falsef(t, taken,
				"task queue %s: %s and %s both register an activity named %s; "+
					"Temporal keys activities by name per queue, so one of them must be renamed",
				queue, previous, owner.pkg, owner.activity,
			)
			holders[owner.activity] = owner.pkg
		}
	}
}

// taskQueueOf reads the queue a jobs package registers on. A directory
// without a registry.go is not a worker package.
func taskQueueOf(t *testing.T, dir string) (string, bool) {
	t.Helper()

	source, err := os.ReadFile(filepath.Join(dir, "registry.go"))
	if err != nil {
		return "", false
	}
	match := taskQueuePattern.FindSubmatch(source)
	if match == nil {
		return "", false
	}

	return strings.TrimSpace(string(match[1])), true
}

// activityMethodsIn is every exported method on the package's Activities
// struct, which is the set Temporal registers from it.
func activityMethodsIn(t *testing.T, dir string) []string {
	t.Helper()

	fileSet := token.NewFileSet()
	pkgs, err := parser.ParseDir(fileSet, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	require.NoError(t, err)

	var names []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || !fn.Name.IsExported() {
					continue
				}
				if receiverTypeName(fn.Recv) == "Activities" {
					names = append(names, fn.Name.Name)
				}
			}
		}
	}

	return names
}

func receiverTypeName(recv *ast.FieldList) string {
	if len(recv.List) == 0 {
		return ""
	}
	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}

	return ident.Name
}
