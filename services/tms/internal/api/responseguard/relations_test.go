package responseguard

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v3"
)

const (
	modulePath     = "github.com/emoss08/trenova"
	helpersPackage = modulePath + "/internal/api/helpers"
	formulaPackage = modulePath + "/internal/api/formulavisibility"
)

// relationTarget is a struct type a REST response reaches through a keel
// relation, and one route path that shows how.
type relationTarget struct {
	pkgPath string
	name    string
	via     string
	fields  []string
}

func (r relationTarget) key() string { return r.pkgPath + "." + r.name }

var (
	restTypesOnce sync.Once
	restTypes     map[string]types.Type
	directWrites  []string
	restTypesErr  error
)

// ginWriters are the gin.Context methods that write a value as the response
// body, which a handler must leave to helpers.JSON.
var ginWriters = map[string]struct{}{
	"JSON":                {},
	"IndentedJSON":        {},
	"PureJSON":            {},
	"SecureJSON":          {},
	"AsciiJSON":           {},
	"JSONP":               {},
	"AbortWithStatusJSON": {},
	"Render":              {},
	"Negotiate":           {},
	"YAML":                {},
	"TOML":                {},
	"XML":                 {},
	"ProtoBuf":            {},
}

// responseTypes loads the handlers and returns the static type of every value
// a handler writes: the value given to helpers.JSON or formulavisibility, and
// the element type of each pagination helper a handler calls.
func responseTypes(t *testing.T) map[string]types.Type {
	t.Helper()

	restTypesOnce.Do(func() {
		cfg := &packages.Config{
			Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax |
				packages.NeedTypesInfo | packages.NeedFiles,
			Dir: "../../..",
		}
		pkgs, err := packages.Load(cfg, "./internal/api/handlers/...")
		if err != nil {
			restTypesErr = err
			return
		}
		restTypes = make(map[string]types.Type)
		for _, pkg := range pkgs {
			if len(pkg.Errors) > 0 {
				restTypesErr = fmt.Errorf("package %s: %w", pkg.PkgPath, pkg.Errors[0])
				return
			}
			for _, file := range pkg.Syntax {
				collectResponseTypes(pkg, file, restTypes)
				directWrites = append(directWrites, collectDirectWrites(pkg, file)...)
			}
		}
	})
	require.NoError(t, restTypesErr)
	require.NotEmpty(t, restTypes)

	return restTypes
}

func collectResponseTypes(pkg *packages.Package, file *ast.File, out map[string]types.Type) {
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident := calleeIdent(call.Fun)
		if ident == nil {
			return true
		}
		fn, ok := pkg.TypesInfo.Uses[ident].(*types.Func)
		if !ok || fn.Pkg() == nil {
			return true
		}
		position := pkg.Fset.Position(call.Pos()).String()
		switch {
		case fn.Pkg().Path() == helpersPackage && fn.Name() == "JSON" && len(call.Args) == 3:
			addType(out, position, pkg.TypesInfo.TypeOf(call.Args[2]))
		case fn.Pkg().Path() == formulaPackage && fn.Name() == "Respond" && len(call.Args) == 5:
			addType(out, position, pkg.TypesInfo.TypeOf(call.Args[3]))
		case fn.Pkg().Path() == paginationPackage:
			if instance, found := pkg.TypesInfo.Instances[ident]; found {
				for i := range instance.TypeArgs.Len() {
					addType(out, position, types.NewSlice(instance.TypeArgs.At(i)))
				}
			}
		}
		return true
	})
}

func collectDirectWrites(pkg *packages.Package, file *ast.File) []string {
	var out []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if _, writer := ginWriters[sel.Sel.Name]; !writer {
			return true
		}
		if isGinContext(pkg.TypesInfo.TypeOf(sel.X)) {
			out = append(out, shortPosition(pkg.Fset.Position(call.Pos()).String())+
				": "+sel.Sel.Name)
		}
		return true
	})

	return out
}

func isGinContext(typ types.Type) bool {
	if typ == nil {
		return false
	}
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	named, ok := typ.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}

	return named.Obj().Pkg().Path() == "github.com/gin-gonic/gin" && named.Obj().Name() == "Context"
}

func calleeIdent(fun ast.Expr) *ast.Ident {
	switch f := fun.(type) {
	case *ast.Ident:
		return f
	case *ast.SelectorExpr:
		return f.Sel
	case *ast.IndexExpr:
		return calleeIdent(f.X)
	case *ast.IndexListExpr:
		return calleeIdent(f.X)
	default:
		return nil
	}
}

func addType(out map[string]types.Type, position string, typ types.Type) {
	if typ == nil {
		return
	}
	out[position] = typ
}

// Every handler writes its response with helpers.JSON, which is what puts it
// through the caller's guard; a handler writing with gin directly would show
// the records it carries whole.
func TestHandlersWriteEveryResponseThroughTheGuard(t *testing.T) {
	responseTypes(t)
	slices.Sort(directWrites)
	require.Empty(t, directWrites,
		"write these responses with helpers.JSON:\n"+strings.Join(directWrites, "\n"))
}

// relationTargets walks every response type's JSON fields and returns each
// named struct type reached through a field that carries a keel relation.
func relationTargets(t *testing.T) map[string]relationTarget {
	t.Helper()

	out := make(map[string]relationTarget)
	seen := make(map[types.Type]struct{})
	positions := make([]string, 0, len(responseTypes(t)))
	for position := range responseTypes(t) {
		positions = append(positions, position)
	}
	slices.Sort(positions)
	for _, position := range positions {
		walkType(responseTypes(t)[position], shortPosition(position), false, seen, out)
	}

	return out
}

func shortPosition(position string) string {
	if i := strings.Index(position, "internal/api/handlers/"); i >= 0 {
		return position[i:]
	}
	return position
}

func walkType(
	typ types.Type,
	via string,
	relation bool,
	seen map[types.Type]struct{},
	out map[string]relationTarget,
) {
	switch u := types.Unalias(typ).(type) {
	case *types.Pointer:
		walkType(u.Elem(), via, relation, seen, out)
		return
	case *types.Slice:
		walkType(u.Elem(), via, relation, seen, out)
		return
	case *types.Array:
		walkType(u.Elem(), via, relation, seen, out)
		return
	case *types.Map:
		walkType(u.Elem(), via, relation, seen, out)
		return
	case *types.Named:
		st, ok := u.Underlying().(*types.Struct)
		if !ok {
			return
		}
		obj := u.Obj()
		if relation && obj.Pkg() != nil {
			target := relationTarget{
				pkgPath: obj.Pkg().Path(),
				name:    obj.Name(),
				via:     via,
				fields:  jsonNames(st),
			}
			if _, known := out[target.key()]; !known {
				out[target.key()] = target
			}
		}
		if _, done := seen[u]; done {
			return
		}
		seen[u] = struct{}{}
		walkStruct(st, via+" > "+obj.Name(), seen, out)
	case *types.Struct:
		walkStruct(u, via, seen, out)
	}
}

func walkStruct(
	st *types.Struct,
	via string,
	seen map[types.Type]struct{},
	out map[string]relationTarget,
) {
	for i := range st.NumFields() {
		f := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))
		if tag.Get("json") == "-" {
			continue
		}
		name, _ := parseTag(tag.Get("json"))
		if !f.Exported() && !f.Embedded() {
			continue
		}
		if name == "" {
			name = f.Name()
		}
		walkType(f.Type(), via+"."+name, isKeelRelation(tag.Get("keel")), seen, out)
	}
}

// carried are the struct types a response reaches through a relation that
// are no kind of record of their own: each is part of the record that carries
// it, shown or withheld with that record's field that leads to it.
var carried = map[string]string{
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry.JournalBatch":                               "the posting batch a journal entry was written in, part of the entry",
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry.JournalEntryLine":                           "a line of the journal entry that carries it",
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal.Line":                                      "a line of the manual journal request that carries it",
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation.Thread":                                          "the caller's own conversation a message belongs to",
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue.ApprovalRunItem":                               "one item of the billing queue approval run that carries it",
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment.Application":                                "how the payment that carries it was applied to an invoice",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice.Attachment":                                         "a file attached to the invoice that carries it",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice.DocumentShareToken":                                 "the share link of an invoice delivery attachment",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice.EmailAttempt":                                       "a delivery attempt of the invoice that carries it",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice.EmailAttemptAttachment":                             "a file sent with an invoice delivery attempt",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice.InvoiceLine":                                        "a line of the invoice that carries it",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment.InvoiceAdjustmentBatchItem":               "one item of the adjustment batch that carries it",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment.InvoiceAdjustmentDocumentReference":       "a document cited by the adjustment that carries it",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment.InvoiceAdjustmentLine":                    "a line of the adjustment that carries it",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment.InvoiceAdjustmentReconciliationException": "an exception raised reconciling the adjustment that carries it",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment.InvoiceAdjustmentSnapshot":                "the invoice as it stood when the adjustment that carries it was made",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun.InvoiceRunGroup":                                 "a group of the invoice run that carries it",
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun.InvoiceRunGroupItem":                             "an item of an invoice run group",
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture.CaptureItem":                                      "a scanned item of the capture batch that carries it",
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture.CapturePage":                                      "a page of a scanned capture item",
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi.EDIConnection":                                              "a connection of the EDI partner that carries it",
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi.EDIDocumentType":                                            "the document type an EDI message or communication profile is for",
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi.EDIMappingProfile":                                          "the field mapping of the EDI partner that carries it",
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi.EDIMappingProfileItem":                                      "one mapping of an EDI mapping profile",
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi.EDIMessageValidationError":                                  "a validation error of the EDI message that carries it",
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi.EDITransactionSet":                                          "the X12 transaction set an EDI document type names",
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi.TenderRecipient":                                            "who a tender is sent to on the EDI connection that carries it",
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission.ResourcePermission":                             "a grant of the role that carries it",
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission.UserRoleAssignment":                             "a role held by the user that carries it",
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant.OrganizationMembership":                             "an organization the user that carries it belongs to",
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate.UsState":                                             "reference data: a US state, the same for every tenant",
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention.DetentionEvidence":                             "evidence attached to the detention that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention.DetentionNotice":                               "a notice sent for the detention that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention.DetentionPolicyTier":                           "a tier of the detention policy that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/order.OrderCharge":                                       "a charge of the order that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment.AdditionalCharge":                               "an additional charge of the shipment that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment.Assignment":                                     "the equipment and workers assigned to the move that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment.CarrierAssignment":                              "the carrier assigned to the move that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment.CarrierAssignmentAccessorial":                   "an accessorial of a carrier assignment",
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment.ChargeAllocation":                               "how a charge of the shipment that carries it is split between bill-to customers",
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment.ShipmentCommentAcknowledgment":                  "who acknowledged the shipment comment that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment.ShipmentCommentMention":                         "who the shipment comment that carries it mentions",
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment.ShipmentCommodity":                              "a commodity line of the shipment that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment.ShipmentMoveJurisdictionMile":                   "the miles of the move that carries it in one jurisdiction",
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender.RoutingGuideEntry":                                "an entry of the routing guide that carries it",
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender.TenderOffer":                                      "an offer of the tender that carries it",
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier.CarrierContact":                                     "a contact of the carrier that carries it",
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier.CarrierEDIChannel":                                  "an EDI channel of the carrier that carries it",
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier.CarrierInsurancePolicy":                             "an insurance policy of the carrier that carries it",
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer.CustomerBillingProfile":                            "the billing settings of the customer that carries it",
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer.CustomerEmailProfile":                              "the email settings of the customer that carries it",
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceoverride.DistanceOverrideStop":                       "a stop of the distance override that carries it",
	"github.com/emoss08/trenova/internal/modules/places/domain/jurisdictionrule.EscortThreshold":                            "a threshold of the jurisdiction rule that carries it",
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate.FormulaTemplateVersion":                      "a version of the formula template that carries it",
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge.FuelIndex":                                     "the price index a fuel surcharge program follows",
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge.FuelSurchargeTableRow":                         "a row of the fuel surcharge program that carries it",
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement.RateAgreementAccessorial":                      "an accessorial of the rate agreement that carries it",
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement.RateAgreementFuelBinding":                      "the fuel surcharge binding of the rate agreement that carries it",
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement.RateAgreementRule":                             "a rule of the rate agreement that carries it",
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport.RateImportRow":                                    "a row of the rate import that carries it",
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix.RateMatrixDimension":                              "a dimension of the rate matrix that carries it",
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratesimulation.RateSimulationResult":                         "the result of the rate simulation that carries it",
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone.RateZoneMember":                                     "a member of the rate zone that carries it",
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker.WorkerProfile":                                     "the license and qualification details of the worker that carries it",
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email.Attachment":                                         "a file attached to the email message that carries it",
}

// Every struct a REST response reaches through a keel relation is either a
// kind of record, gated by its reference view where it is linked, or part of
// the record that carries it. A relation to a type that is neither would be
// shown whole wherever it appears.
func TestEveryRelationAResponseReachesIsClassified(t *testing.T) {
	registry := fieldsensitivity.DefaultRegistry()

	var missing []string
	for _, target := range relationTargets(t) {
		if kindOfName(registry, target.pkgPath, target.name) != "" {
			continue
		}
		if _, ok := carried[target.key()]; ok {
			continue
		}
		missing = append(missing, fmt.Sprintf(
			"%s (reached at %s): add it to namedKinds in kinds.go with the resource whose "+
				"records it is, or to carried in relations_test.go with why it is part of "+
				"the record that carries it",
			target.key(), target.via))
	}
	slices.Sort(missing)
	require.Empty(t, missing, "\n"+strings.Join(missing, "\n"))
}

func TestCarriedListsOnlyTypesWithNoKind(t *testing.T) {
	registry := fieldsensitivity.DefaultRegistry()
	targets := relationTargets(t)

	var stale []string
	for key := range carried {
		pkgPath, name := splitTypeKey(key)
		if kind := kindOfName(registry, pkgPath, name); kind != "" {
			stale = append(stale, fmt.Sprintf("%s is the %s kind; remove it from carried", key, kind))
			continue
		}
		if _, ok := targets[key]; !ok {
			stale = append(stale, key+" is no longer reached by a REST response; remove it")
		}
	}
	slices.Sort(stale)
	require.Empty(t, stale, "\n"+strings.Join(stale, "\n"))
}

func splitTypeKey(key string) (string, string) {
	i := strings.LastIndex(key, ".")
	return key[:i], key[i+1:]
}

// A Go type bound to a GraphQL type the field guard gates as a kind is gated
// as the same kind here, so a record shows the same fields on both surfaces.
func TestEveryKindAgreesWithTheGraphQLGuard(t *testing.T) {
	registry := fieldsensitivity.DefaultRegistry()
	schema, err := schemadiff.LoadDir("../graphql/schema")
	require.NoError(t, err)
	graphKinds, err := gqlext.KindTypes(schema, registry)
	require.NoError(t, err)
	bindings := gqlModels(t)

	var mismatched []string
	compared := 0
	for _, target := range relationTargets(t) {
		ours := kindOfName(registry, target.pkgPath, target.name)
		for _, graphType := range bindings[target.key()] {
			theirs, gated := graphKinds[graphType]
			if !gated {
				continue
			}
			compared++
			if theirs == ours {
				continue
			}
			mismatched = append(mismatched, fmt.Sprintf(
				"%s is %q here but GraphQL %s is %q; give it that resource in namedKinds",
				target.key(), ours, graphType, theirs))
		}
	}
	slices.Sort(mismatched)
	require.Empty(t, mismatched, "\n"+strings.Join(mismatched, "\n"))
	require.Greater(t, compared, 20, "gqlgen.yml bindings were not read")
}

// A kind no GraphQL type shows is gated on REST alone, so every field of its
// reference view must be a JSON field of a Go type a response carries it as,
// or a typo would silently show nothing.
func TestEveryReferenceFieldExistsOnItsRESTKind(t *testing.T) {
	registry := fieldsensitivity.DefaultRegistry()
	schema, err := schemadiff.LoadDir("../graphql/schema")
	require.NoError(t, err)
	graphKinds, err := gqlext.KindTypes(schema, registry)
	require.NoError(t, err)
	shown := make(map[permission.Resource]struct{}, len(graphKinds))
	for _, kind := range graphKinds {
		shown[kind] = struct{}{}
	}
	carriedAs := make(map[permission.Resource][]relationTarget)
	for _, target := range relationTargets(t) {
		kind := kindOfName(registry, target.pkgPath, target.name)
		carriedAs[kind] = append(carriedAs[kind], target)
	}

	var missing []string
	for _, def := range registry.All() {
		kind := permission.Resource(def.Resource)
		if _, ok := shown[kind]; ok || len(def.ReferenceView) == 0 {
			continue
		}
		targets := carriedAs[kind]
		if len(targets) == 0 {
			missing = append(missing, fmt.Sprintf(
				"%s declares a reference view, but no GraphQL type or REST response shows it",
				def.Resource))
			continue
		}
		for _, field := range def.ReferenceView {
			if !slices.ContainsFunc(targets, func(target relationTarget) bool {
				return slices.Contains(target.fields, field)
			}) {
				missing = append(missing, fmt.Sprintf(
					"%s.%s is not a JSON field of %s", def.Resource, field, targets[0].key()))
			}
		}
	}
	slices.Sort(missing)
	require.Empty(t, missing,
		"fix the field name in the kind's ReferenceView (resources_*.go):\n"+
			strings.Join(missing, "\n"))
}

func jsonNames(st *types.Struct) []string {
	out := make([]string, 0, st.NumFields())
	for i := range st.NumFields() {
		f := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))
		if tag.Get("json") == "-" || (!f.Exported() && !f.Embedded()) {
			continue
		}
		name, _ := parseTag(tag.Get("json"))
		if name == "" {
			name = f.Name()
		}
		out = append(out, name)
	}

	return out
}

// gqlModels maps each Go type gqlgen.yml binds to the GraphQL types bound to it.
func gqlModels(t *testing.T) map[string][]string {
	t.Helper()

	raw, err := os.ReadFile("../../../gqlgen.yml")
	require.NoError(t, err)
	var config struct {
		Models map[string]struct {
			Model yaml.Node `yaml:"model"`
		} `yaml:"models"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &config))

	out := make(map[string][]string)
	for graphType, model := range config.Models {
		var goTypes []string
		switch model.Model.Kind {
		case yaml.ScalarNode:
			goTypes = []string{model.Model.Value}
		case yaml.SequenceNode:
			require.NoError(t, model.Model.Decode(&goTypes))
		default:
			continue
		}
		for _, goType := range goTypes {
			out[goType] = append(out[goType], graphType)
		}
	}

	return out
}

// The kinds the table resolves are the ones the registry knows.
func TestNamedKindsAreRegisteredResources(t *testing.T) {
	t.Parallel()

	registry := fieldsensitivity.DefaultRegistry()
	for typ, kind := range namedKinds {
		require.Truef(t, registry.HasResource(kind.String()),
			"%s names unknown resource %s", typ, kind)
		require.Truef(t, isDomainPackage(typ.PkgPath()), "%s is not a domain type", typ)
		require.NotEqualf(t, permission.Resource(stringutils.ConvertCamelToSnake(typ.Name())), kind,
			"%s is spelled as its resource; it needs no entry", typ)
	}
}

// rootKindsOf is every kind a response written as typ returns as its own
// record: the value itself, each item of a list, or each item of a page.
func rootKindsOf(registry *permission.Registry, typ types.Type, depth int) []permission.Resource {
	if depth > maxDepth {
		return nil
	}
	switch t := typ.(type) {
	case *types.Pointer:
		return rootKindsOf(registry, t.Elem(), depth+1)
	case *types.Slice:
		return rootKindsOf(registry, t.Elem(), depth+1)
	case *types.Array:
		return rootKindsOf(registry, t.Elem(), depth+1)
	case *types.Map:
		return rootKindsOf(registry, t.Elem(), depth+1)
	case *types.Named:
		obj := t.Obj()
		if obj.Pkg() == nil {
			return nil
		}
		if kind := kindOfName(registry, obj.Pkg().Path(), obj.Name()); kind != "" {
			return []permission.Resource{kind}
		}
		if obj.Pkg().Path() != paginationPackage {
			return nil
		}
		st, ok := t.Underlying().(*types.Struct)
		if !ok {
			return nil
		}
		var out []permission.Resource
		for i := range st.NumFields() {
			out = append(out, rootKindsOf(registry, st.Field(i).Type(), depth+1)...)
		}
		return out
	}
	return nil
}

// Row-level security does not hide a shipment, order, invoice or worker the
// person cannot read, because a record that links to one shows its reference
// view. A route returns its records whole, so only the kind's own handler,
// whose every route requires an operation on the kind (or marks its records
// as references), may write one as the response's own record.
func TestEveryRESTRootOfADatabaseScopedKindIsWrittenByItsOwnHandler(t *testing.T) {
	registry := permission.NewRegistry()
	scoped := registry.DatabaseScopedResources()

	var outside []string
	reached := 0
	for position, typ := range responseTypes(t) {
		for _, kind := range rootKindsOf(registry, typ, 0) {
			if !slices.Contains(scoped, kind.String()) {
				continue
			}
			reached++
			own := "internal/api/handlers/" + strings.ReplaceAll(kind.String(), "_", "") + "handler/"
			if !strings.Contains(position, own) {
				outside = append(outside, fmt.Sprintf("%s writes %s", shortPosition(position), kind))
			}
		}
	}
	require.Positive(t, reached)
	slices.Sort(outside)
	require.Empty(t, outside,
		"these responses return records of a database-scoped kind outside its own handler; "+
			"return them inside another record (they are then linked) or move the route:\n"+
			strings.Join(outside, "\n"))
}

var keelRelationHeads = []string{"belongs:", "one:", "many:", "m2m:", "pick:"}

func isKeelRelation(tag string) bool {
	for _, head := range keelRelationHeads {
		if strings.HasPrefix(tag, head) {
			return true
		}
	}

	return false
}
