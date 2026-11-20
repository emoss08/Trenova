package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	modulePath    = "github.com/emoss08/trenova/"
	moduleRoot    = "../.."
	routerSource  = "router.go"
	featureMethod = "RequireFeature"
)

var publicSetups = []string{"setupRoutes", "setupPublicRoutes"}

var baseWaysIn = map[string]string{
	"publicConfigHandler.RegisterPublicRoutes": "the client's configuration before sign-in; it names no tenant",
	"versionHandler.RegisterPublicRoutes":      "the build the server runs",
	"emailHandler.RegisterPublicRoutes": "delivery events for mail the organization sent; " +
		"sending email is Administration, a base feature",
	"invoiceHandler.RegisterPublicRoutes": "an invoice's view link; invoices are Billing, a base feature",
	"captureHandler.RegisterPublicRoutes": "pairing a scanner; Capture files documents, a base feature",
	"captureHandler.RegisterDeviceRoutes": "a paired scanner, authenticated by its device token; " +
		"documents are a base feature",
	"supportEntryHandler.RegisterPublicRoutes": "Trenova staff entering under a grant; not a plan " +
		"feature but managed mode, and the handler registers nothing when managed.enabled is off",
}

// entryGated are the ways in whose plan check runs in the service once the
// request names its tenant: on a shared server a route has no tenant, so a
// route guard could only ask whether any organization holds the feature. Each
// names the package that checks and the feature it names; the test proves the
// package passes that feature to a check.
type entryCheck struct {
	dir     string
	feature string
}

var entryGated = map[string]entryCheck{
	"scimHandler.RegisterRoutes": {
		"internal/modules/identity/app/scimservice", "FeatureIdentity",
	},
	"authHandler.RegisterRoutes": {
		"internal/modules/identity/app/authservice", "FeatureIdentity",
	},
	"driverPortalHandler.RegisterRoutes": {
		"internal/modules/workforce/app/driverportalinvitationservice", "FeatureDriverPortal",
	},
	"telematicsHandler.RegisterPublicRoutes": {
		"internal/modules/workforce/app/telematicsservice", "FeatureSamsaraIntegration",
	},
	"accountingWebhookHandler.RegisterPublicRoutes": {
		"internal/modules/accounting/app/accountingconnectionwebhook", "FeatureAccounting",
	},
	"agentExtensionHandler.RegisterPublicRoutes": {
		"internal/modules/agent/app/extensionwebwatchservice", "FeatureAgentAutomation",
	},
	"ediHandler.RegisterPublicRoutes": {
		"internal/modules/edi/app/ediinboundas2", "FeatureEDIIntegration",
	},
	"inboundHandler.RegisterPublicRoutes": {
		"internal/modules/documents/app/inboundintake", "FeatureAITriage",
	},
	"tenderPublicHandler.RegisterPublicRoutes": {
		"internal/modules/operations/app/tenderresponse", "FeatureCarrierBrokerage",
	},
	"rateConfirmationPublicHandler.RegisterPublicRoutes": {
		"internal/modules/rating/app/rateconfirmationservice", "FeatureCarrierBrokerage",
	},
	"carrierTrackingPublicHandler.RegisterPublicRoutes": {
		"internal/modules/operations/app/checkcallservice", "FeatureCustomerTracking",
	},
	"shipmentTrackingPublicHandler.RegisterPublicRoutes": {
		"internal/modules/operations/app/customertrackingservice", "FeatureCustomerTracking",
	},
}

type registration struct {
	field  string
	method string
}

func (r registration) String() string {
	return r.field + "." + r.method
}

func TestEveryRouteWithoutASessionDeclaresItsFeature(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	router, err := parser.ParseFile(fset, routerSource, nil, parser.SkipObjectResolution)
	require.NoError(t, err)

	registrations := publicRegistrations(router)
	require.GreaterOrEqual(t, len(registrations), 15, "the walk found too few public registrations")
	packages := handlerPackages(router)

	seen := make(map[string]bool, len(registrations))
	for _, reg := range registrations {
		seen[reg.String()] = true
		dir, ok := packages[reg.field]
		require.True(t, ok, "no handler package found for %s", reg.field)

		declares := registrationDeclaresFeature(t, fset, dir, reg.method)
		_, base := baseWaysIn[reg.String()]
		entry, gated := entryGated[reg.String()]
		switch {
		case declares && (base || gated):
			t.Errorf(
				"%s declares a feature on its routes and is also listed; drop it from the list",
				reg,
			)
		case base && gated:
			t.Errorf("%s is listed both as base and as checked at its service entry", reg)
		case gated:
			assert.Truef(t, packageNamesFeature(t, fset, entry),
				"%s is listed as checked in %s, which passes platformcatalog.%s to no call",
				reg, entry.dir, entry.feature)
		case !declares && !base:
			t.Errorf(
				"%s registers routes with no signed-in caller and names no feature: "+
					"check it at the service entry under the request's tenant and list it in "+
					"entryGated, wrap the routes in FeatureGuard.RequireFeature, or list "+
					"them as base with a reason",
				reg,
			)
		}
	}

	for name := range baseWaysIn {
		assert.True(t, seen[name], "%s is listed as base but registers no public routes", name)
	}
	for name := range entryGated {
		assert.True(
			t,
			seen[name],
			"%s is listed as entry-gated but registers no public routes",
			name,
		)
	}
}

func publicRegistrations(router *ast.File) []registration {
	var found []registration
	for _, decl := range router.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !isPublicSetup(fn.Name.Name) {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, isCall := n.(*ast.CallExpr)
			if !isCall {
				return true
			}
			method, isSel := call.Fun.(*ast.SelectorExpr)
			if !isSel || !strings.HasPrefix(method.Sel.Name, "Register") {
				return true
			}
			field, isField := method.X.(*ast.SelectorExpr)
			if !isField {
				return true
			}
			if recv, isIdent := field.X.(*ast.Ident); !isIdent || recv.Name != "r" {
				return true
			}
			found = append(found, registration{field: field.Sel.Name, method: method.Sel.Name})
			return true
		})
	}
	return found
}

func isPublicSetup(name string) bool {
	for _, setup := range publicSetups {
		if name == setup {
			return true
		}
	}
	return false
}

func handlerPackages(router *ast.File) map[string]string {
	imports := make(map[string]string, len(router.Imports))
	for _, spec := range router.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || !strings.HasPrefix(path, modulePath) {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = filepath.Join(moduleRoot, strings.TrimPrefix(path, modulePath))
	}

	packages := make(map[string]string)
	ast.Inspect(router, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "Router" {
			return true
		}
		fields, isStruct := spec.Type.(*ast.StructType)
		if !isStruct {
			return false
		}
		for _, field := range fields.Fields.List {
			star, isStar := field.Type.(*ast.StarExpr)
			if !isStar {
				continue
			}
			sel, isSel := star.X.(*ast.SelectorExpr)
			if !isSel {
				continue
			}
			pkg, isIdent := sel.X.(*ast.Ident)
			if !isIdent {
				continue
			}
			dir, known := imports[pkg.Name]
			if !known {
				continue
			}
			for _, name := range field.Names {
				packages[name.Name] = dir
			}
		}
		return false
	})
	return packages
}

func registrationDeclaresFeature(
	t *testing.T,
	fset *token.FileSet,
	dir string,
	method string,
) bool {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	methods := make(map[string]*ast.FuncDecl)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(
			fset,
			filepath.Join(dir, name),
			nil,
			parser.SkipObjectResolution,
		)
		require.NoError(t, parseErr)
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil && fn.Body != nil {
				methods[fn.Name.Name] = fn
			}
		}
	}

	registration, ok := methods[method]
	require.True(t, ok, "%s has no method %s", dir, method)

	return callsFeatureGuard(registration, methods, 0)
}

func packageNamesFeature(t *testing.T, fset *token.FileSet, check entryCheck) bool {
	t.Helper()

	dir := filepath.Join(moduleRoot, check.dir)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(
			fset,
			filepath.Join(dir, name),
			nil,
			parser.SkipObjectResolution,
		)
		require.NoError(t, parseErr)
		if passesFeature(file, check.feature) {
			return true
		}
	}
	return false
}

func passesFeature(file *ast.File, feature string) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || found {
			return !found
		}
		for _, arg := range call.Args {
			sel, isSel := arg.(*ast.SelectorExpr)
			if !isSel || sel.Sel.Name != feature {
				continue
			}
			if pkg, isIdent := sel.X.(*ast.Ident); isIdent && pkg.Name == "platformcatalog" {
				found = true
			}
		}
		return !found
	})
	return found
}

func callsFeatureGuard(fn *ast.FuncDecl, methods map[string]*ast.FuncDecl, depth int) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return true
		}
		if sel.Sel.Name == featureMethod {
			found = true
			return false
		}
		if helper, isMethod := methods[sel.Sel.Name]; isMethod && depth == 0 && helper != fn {
			found = callsFeatureGuard(helper, methods, depth+1)
		}
		return !found
	})
	return found
}
