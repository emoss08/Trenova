package permissionlint_test

import (
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/emoss08/trenova/internal/api/permissionlint"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/stretchr/testify/require"
)

var actionOperations = map[string]permission.Operation{
	"approve":   permission.OpApprove,
	"reject":    permission.OpReject,
	"cancel":    permission.OpCancel,
	"archive":   permission.OpArchive,
	"restore":   permission.OpRestore,
	"submit":    permission.OpSubmit,
	"close":     permission.OpClose,
	"reopen":    permission.OpReopen,
	"lock":      permission.OpLock,
	"unlock":    permission.OpUnlock,
	"assign":    permission.OpAssign,
	"unassign":  permission.OpUnassign,
	"duplicate": permission.OpDuplicate,
	"import":    permission.OpImport,
	"export":    permission.OpExport,
	"activate":  permission.OpActivate,
	"resolve":   permission.OpResolve,
	"pin":       permission.OpPin,
	"unpin":     permission.OpUnpin,
}

var namedActions = map[string]string{
	"post":     "post",
	"unpost":   "post",
	"send":     "send",
	"resend":   "send",
	"void":     "void",
	"apply":    "apply",
	"reverse":  "reverse",
	"waive":    "waive",
	"dispute":  "dispute",
	"release":  "release",
	"publish":  "publish",
	"transmit": "transmit",
	"refund":   "refund",
	"finalize": "finalize",
}

func routeAction(route string) string {
	_, path, _ := strings.Cut(route, " ")
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(segments) - 1; i >= 0; i-- {
		segment := segments[i]
		if segment == "" || strings.HasPrefix(segment, ":") {
			continue
		}
		word, _, _ := strings.Cut(segment, "-")
		return strings.ToLower(word)
	}
	return ""
}

func resolverAction(function string) string {
	for i, r := range function {
		if i > 0 && unicode.IsUpper(r) {
			return strings.ToLower(function[:i])
		}
	}
	return strings.ToLower(function)
}

func operationLabel(
	registry *permissioncatalog.Registry,
	resource string,
	op permission.Operation,
) string {
	def, ok := registry.Get(resource)
	if !ok {
		return ""
	}
	for _, candidate := range def.Operations {
		if candidate.Operation == op {
			return strings.ToLower(candidate.DisplayName + " " + candidate.Description)
		}
	}
	return ""
}

func namesAction(
	registry *permissioncatalog.Registry,
	pair permissionlint.Pair,
	action string,
) bool {
	op := permission.Operation(pair.Operation)
	if natural, ok := actionOperations[action]; ok && natural == op {
		return true
	}
	word, named := namedActions[action]
	if !named {
		if _, natural := actionOperations[action]; !natural {
			return true
		}
		word = action
	}
	return strings.Contains(operationLabel(registry, pair.Resource, op), word)
}

func isExported(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

func isAction(action string) bool {
	_, natural := actionOperations[action]
	_, named := namedActions[action]
	return natural || named
}

func TestEveryActionIsGatedByAPermissionThatNamesIt(t *testing.T) {
	t.Parallel()

	consts, err := permissionlint.LoadConstants(permissionDir)
	require.NoError(t, err)
	result, err := permissionlint.Analyze(consts, handlersDir, resolverDir)
	require.NoError(t, err)

	registry := permissioncatalog.NewRegistry()
	type site struct {
		action string
		pairs  []permissionlint.Pair
	}
	sites := make(map[string]*site)
	for _, pair := range result.Pairs {
		var key, action string
		file, _, _ := strings.Cut(pair.Position, ":")
		switch {
		case pair.Route != "":
			if strings.HasPrefix(pair.Route, "GET ") {
				continue
			}
			action = routeAction(pair.Route)
			key = pair.Position
		case strings.HasSuffix(file, ".resolvers.go") && isExported(pair.Function):
			action = resolverAction(pair.Function)
			key = file + ":" + pair.Function
		default:
			continue
		}
		if !isAction(action) {
			continue
		}
		if sites[key] == nil {
			sites[key] = &site{action: action}
		}
		sites[key].pairs = append(sites[key].pairs, pair)
	}

	var unnamed []string
	for key, s := range sites {
		named := false
		for _, pair := range s.pairs {
			if namesAction(registry, pair, s.action) {
				named = true
				break
			}
		}
		if !named {
			pair := s.pairs[0]
			unnamed = append(
				unnamed,
				key+" "+pair.Route+" "+s.action+" -> "+pair.Resource+":"+pair.Operation,
			)
		}
	}
	sort.Strings(unnamed)

	require.Empty(t, unnamed,
		"these routes and resolvers perform an action the operation they check does not "+
			"name, so a role author looking for it in the role editor cannot find it; check "+
			"the action's own operation, or name the action in that operation's display name "+
			"or description for the resource in permission/resources_*.go")
}

func TestRouteActionReadsTheLastLiteralSegment(t *testing.T) {
	t.Parallel()

	require.Equal(t, "post", routeAction("POST /:invoiceID/post/"))
	require.Equal(t, "retry", routeAction("POST /:messageID/retry-delivery/"))
	require.Equal(t, "apply", routeAction("POST /rate-increase/apply/"))
	require.Equal(t, "", routeAction("PUT /:invoiceID/"))
	require.Equal(t, "send", resolverAction("SendInvoice"))
}
