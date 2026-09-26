package accountingsyncservice

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const enqueueCall = "EnqueueAccountingSync"

type postingPackage struct {
	name          string
	partyWrite    bool
	enqueuePoints []string
}

func postingPackages() []postingPackage {
	return []postingPackage{
		{name: "invoiceservice", enqueuePoints: []string{"Service.Post"}},
		{name: "invoiceadjustmentservice", enqueuePoints: []string{"Service.executeApprovedAdjustment"}},
		{name: "customerpaymentservice", enqueuePoints: []string{
			"Service.PostAndApply",
			"Service.ApplyUnapplied",
			"Service.Reverse",
			"Service.ApplyCreditMemo",
			"Service.UnapplyCreditMemoApplication",
		}},
		{name: "customerservice", partyWrite: true, enqueuePoints: []string{"Service.updateAndQueueSync"}},
		{name: "carriersettlementservice", enqueuePoints: []string{"Service.queueSync"}},
		{name: "driversettlementservice", enqueuePoints: []string{"Service.queueSync"}},
		{name: "carrierservice", partyWrite: true, enqueuePoints: []string{"Service.updateAndQueueSync"}},
		{name: "workerservice", partyWrite: true, enqueuePoints: []string{"Service.updateAndQueueSync"}},
	}
}

var postingAllowList = map[string]string{
	"customerpaymentservice.Service.planPost": "builds the payment a post would record without " +
		"saving it, for the preview and for PostAndApply; PostAndApply, the only caller that " +
		"saves it, enqueues the sync record",
	"invoiceservice.Service.planPost": "marks the loaded invoice Posted without saving it, for " +
		"PreviewPost and for Post; Post, the only caller that saves it, enqueues the sync " +
		"record in the same transaction",
}

var postedStatusPackages = []string{"invoice", "customerpayment"}

var settlementStatusPackages = []string{"carriersettlement", "driversettlement"}

var settlementSyncedStatuses = []string{"StatusPosted", "StatusPaid", "StatusVoided"}

type postingFunc struct {
	key      string
	pos      string
	sinks    []string
	enqueues bool
	callees  map[string]struct{}
}

type postingGraph struct {
	funcs   map[string]*postingFunc
	callers map[string][]string
}

func receiverOf(decl *ast.FuncDecl) (typeName, varName string) {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return "", ""
	}
	field := decl.Recv.List[0]
	expr := field.Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if generic, ok := expr.(*ast.IndexExpr); ok {
		expr = generic.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		typeName = ident.Name
	}
	if len(field.Names) > 0 {
		varName = field.Names[0].Name
	}
	return typeName, varName
}

func funcKey(decl *ast.FuncDecl) string {
	typeName, _ := receiverOf(decl)
	if typeName == "" {
		return decl.Name.Name
	}
	return typeName + "." + decl.Name.Name
}

func isPostedStatus(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if slices.Contains(settlementStatusPackages, pkg.Name) {
		return slices.Contains(settlementSyncedStatuses, sel.Sel.Name)
	}
	return sel.Sel.Name == "StatusPosted" && slices.Contains(postedStatusPackages, pkg.Name)
}

func isStatusField(expr ast.Expr) bool {
	switch node := expr.(type) {
	case *ast.SelectorExpr:
		return node.Sel.Name == "Status"
	case *ast.Ident:
		return node.Name == "Status"
	default:
		return false
	}
}

func isRepoUpdate(sel *ast.SelectorExpr, receiver string) bool {
	if sel.Sel.Name != "Update" {
		return false
	}
	inner, ok := sel.X.(*ast.SelectorExpr)
	if !ok || inner.Sel.Name != "repo" {
		return false
	}
	owner, ok := inner.X.(*ast.Ident)
	return ok && owner.Name == receiver
}

func inspectPostingFunc(
	fset *token.FileSet,
	decl *ast.FuncDecl,
	partyWrite bool,
) *postingFunc {
	typeName, receiver := receiverOf(decl)
	fn := &postingFunc{
		key:     funcKey(decl),
		pos:     fset.Position(decl.Pos()).String(),
		callees: map[string]struct{}{},
	}
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.CallExpr:
			switch callee := n.Fun.(type) {
			case *ast.SelectorExpr:
				switch {
				case callee.Sel.Name == enqueueCall:
					fn.enqueues = true
				case callee.Sel.Name == "CreatePosting":
					fn.sinks = append(fn.sinks, "creates a journal posting")
				case partyWrite && isRepoUpdate(callee, receiver):
					fn.sinks = append(fn.sinks, "writes a customer, carrier or worker")
				}
				if owner, ok := callee.X.(*ast.Ident); ok && receiver != "" && owner.Name == receiver {
					fn.callees[typeName+"."+callee.Sel.Name] = struct{}{}
				}
			case *ast.Ident:
				if callee.Name == enqueueCall {
					fn.enqueues = true
				}
				fn.callees[callee.Name] = struct{}{}
			}
		case *ast.AssignStmt:
			for idx, lhs := range n.Lhs {
				if idx < len(n.Rhs) && isStatusField(lhs) && isPostedStatus(n.Rhs[idx]) {
					fn.sinks = append(fn.sinks, "writes a synced status")
				}
			}
		case *ast.KeyValueExpr:
			if isStatusField(n.Key) && isPostedStatus(n.Value) {
				fn.sinks = append(fn.sinks, "writes a synced status")
			}
		}
		return true
	})
	return fn
}

func buildPostingGraph(fset *token.FileSet, files []*ast.File, partyWrite bool) *postingGraph {
	graph := &postingGraph{funcs: map[string]*postingFunc{}, callers: map[string][]string{}}
	for _, file := range files {
		for _, decl := range file.Decls {
			fnDecl, ok := decl.(*ast.FuncDecl)
			if !ok || fnDecl.Body == nil {
				continue
			}
			fn := inspectPostingFunc(fset, fnDecl, partyWrite)
			graph.funcs[fn.key] = fn
		}
	}
	for key, fn := range graph.funcs {
		for callee := range fn.callees {
			if _, known := graph.funcs[callee]; known && callee != key {
				graph.callers[callee] = append(graph.callers[callee], key)
			}
		}
	}
	return graph
}

func (g *postingGraph) enqueuesVia(key string, visiting map[string]bool) bool {
	fn := g.funcs[key]
	if fn.enqueues {
		return true
	}
	if visiting[key] {
		return false
	}
	visiting[key] = true
	defer delete(visiting, key)
	for callee := range fn.callees {
		if _, known := g.funcs[callee]; known && g.enqueuesVia(callee, visiting) {
			return true
		}
	}
	return false
}

func (g *postingGraph) covered(key string, visiting map[string]bool) bool {
	if g.enqueuesVia(key, map[string]bool{}) {
		return true
	}
	callers := g.callers[key]
	if len(callers) == 0 || visiting[key] {
		return false
	}
	visiting[key] = true
	defer delete(visiting, key)
	for _, caller := range callers {
		if !g.covered(caller, visiting) {
			return false
		}
	}
	return true
}

func (g *postingGraph) uncovered() []*postingFunc {
	out := []*postingFunc{}
	for key, fn := range g.funcs {
		if len(fn.sinks) > 0 && !g.covered(key, map[string]bool{}) {
			out = append(out, fn)
		}
	}
	slices.SortFunc(out, func(a, b *postingFunc) int { return strings.Compare(a.key, b.key) })
	return out
}

func parsePackageDir(t *testing.T, fset *token.FileSet, dir string) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	files := make([]*ast.File, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		require.NoError(t, parseErr)
		files = append(files, file)
	}
	require.NotEmpty(t, files, "no Go files in %s", dir)
	return files
}

func TestEveryPostingPathEnqueues(t *testing.T) {
	t.Parallel()

	usedAllowances := map[string]bool{}
	for _, pkg := range postingPackages() {
		t.Run(pkg.name, func(t *testing.T) {
			fset := token.NewFileSet()
			graph := buildPostingGraph(fset, parsePackageDir(t, fset, filepath.Join("..", pkg.name)), pkg.partyWrite)

			sinks := 0
			for _, fn := range graph.funcs {
				sinks += len(fn.sinks)
				if len(fn.sinks) > 0 {
					t.Logf("%s %s; enqueues itself: %t; callers: %v",
						fn.key, strings.Join(slices.Compact(fn.sinks), " and "), fn.enqueues, graph.callers[fn.key])
				}
			}
			require.Positive(t, sinks, "the guard found no posting in %s; the matchers are stale", pkg.name)

			for _, point := range pkg.enqueuePoints {
				fn, ok := graph.funcs[point]
				if assert.True(t, ok, "%s.%s is a named enqueue point but no longer exists", pkg.name, point) {
					assert.True(t, fn.enqueues, "%s.%s must call %s", pkg.name, point, enqueueCall)
				}
			}

			for _, fn := range graph.uncovered() {
				qualified := pkg.name + "." + fn.key
				if reason, allowed := postingAllowList[qualified]; allowed {
					usedAllowances[qualified] = true
					t.Logf("%s is allowed without an enqueue: %s", qualified, reason)
					continue
				}
				t.Errorf(
					"%s (%s) %s but neither it nor every caller calls %s; enqueue the sync record "+
						"in the same transaction, or add it to postingAllowList with a reason",
					qualified, fn.pos, strings.Join(slices.Compact(fn.sinks), " and "), enqueueCall,
				)
			}
		})
	}

	for qualified := range postingAllowList {
		assert.True(t, usedAllowances[qualified], "%s is allow-listed but no longer needs it", qualified)
	}
}

func TestPostingGuardFlagsAPathThatSkipsTheEnqueue(t *testing.T) {
	t.Parallel()

	const src = `package sample

type Service struct{ repo repo; journalRepo journal }

func (s *Service) Post() error {
	if err := s.createPosting(); err != nil {
		return err
	}
	return services.EnqueueAccountingSync(nil, nil, nil)
}

func (s *Service) createPosting() error {
	return s.journalRepo.CreatePosting(nil, params{})
}

func (s *Service) Repost() error {
	return s.createPosting()
}

func (s *Service) Adjust() *invoice.Invoice {
	return &invoice.Invoice{Status: invoice.StatusPosted}
}

func (s *Service) Settle(entity *customerpayment.Payment) {
	entity.Status = customerpayment.StatusPosted
	enqueue()
}

func (s *Service) Save(entity *customer.Customer) {
	s.repo.Update(nil, entity)
}

func (s *Service) Compare(entity *invoice.Invoice) bool {
	return entity.Status == invoice.StatusPosted
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.SkipObjectResolution)
	require.NoError(t, err)

	graph := buildPostingGraph(fset, []*ast.File{file}, true)
	flagged := []string{}
	for _, fn := range graph.uncovered() {
		flagged = append(flagged, fn.key)
	}

	assert.Equal(t, []string{"Service.Adjust", "Service.Save", "Service.Settle", "Service.createPosting"}, flagged,
		"a helper is covered only when every caller enqueues")
	assert.True(t, graph.funcs["Service.Post"].enqueues)
	assert.Empty(t, graph.funcs["Service.Compare"].sinks, "reading a Posted status is not a write")
}
