package rlslint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var scopedHelpers = map[string]bool{
	"Read": true, "Write": true, "ReadErr": true, "WriteErr": true, "Read2": true, "Write2": true,
}

type TxViolation struct {
	Key      string
	Position string
	Problem  string
}

func ScanRepositories(dir, root string) ([]TxViolation, error) {
	violations := make([]TxViolation, 0)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		found, scanErr := scanPackage(path, root)
		if scanErr != nil {
			return scanErr
		}
		violations = append(violations, found...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(violations, func(i, j int) bool {
		return violations[i].Position < violations[j].Position
	})

	return violations, nil
}

func scanPackage(dir, root string) ([]TxViolation, error) {
	fset := token.NewFileSet()
	files, err := parseSources(fset, dir)
	if err != nil {
		return nil, err
	}

	conns := connectionFields(files)
	violations := make([]TxViolation, 0)
	for path, file := range files {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil, relErr
		}
		violations = append(
			violations,
			scanRepositoryFile(fset, filepath.ToSlash(rel), file, conns)...,
		)
	}

	return violations, nil
}

func parseSources(fset *token.FileSet, dir string) (map[string]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	files := make(map[string]*ast.File, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(dir, name)
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, parseErr)
		}
		files[path] = file
	}

	return files, nil
}

func connectionFields(files map[string]*ast.File) map[string]map[string]bool {
	out := make(map[string]map[string]bool)
	for _, file := range files {
		for _, typeSpec := range structTypeSpecs(file) {
			if fields := structConnectionFields(typeSpec); len(fields) > 0 {
				out[typeSpec.Name.Name] = fields
			}
		}
	}

	return out
}

func structTypeSpecs(file *ast.File) []*ast.TypeSpec {
	specs := make([]*ast.TypeSpec, 0)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			if typeSpec, isType := spec.(*ast.TypeSpec); isType {
				specs = append(specs, typeSpec)
			}
		}
	}

	return specs
}

func structConnectionFields(typeSpec *ast.TypeSpec) map[string]bool {
	st, isStruct := typeSpec.Type.(*ast.StructType)
	if !isStruct {
		return nil
	}

	fields := make(map[string]bool)
	for _, field := range st.Fields.List {
		if !isConnectionType(field.Type) {
			continue
		}
		for _, name := range field.Names {
			fields[name.Name] = true
		}
	}

	return fields
}

func isConnectionType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.StarExpr:
		sel, ok := t.X.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		id, ok := sel.X.(*ast.Ident)
		return ok && id.Name == "postgres" &&
			(sel.Sel.Name == "Connection" || sel.Sel.Name == "ReportingConnection")
	case *ast.SelectorExpr:
		id, ok := t.X.(*ast.Ident)
		return ok && id.Name == "ports" && t.Sel.Name == "DBConnection"
	default:
		return false
	}
}

func scanRepositoryFile(
	fset *token.FileSet,
	rel string,
	file *ast.File,
	conns map[string]map[string]bool,
) []TxViolation {
	violations := make([]TxViolation, 0)

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		key := rel + ":" + funcName(fn)
		position := fmt.Sprintf("%s:%d", rel, fset.Position(fn.Pos()).Line)

		if usesRawPool(fn.Body) {
			violations = append(violations, TxViolation{
				Key:      key,
				Position: position,
				Problem:  "calls .DB(), which runs outside the scoped transaction; use DBForContext(ctx)",
			})
		}

		if fn.Recv == nil || !fn.Name.IsExported() || len(fn.Recv.List) != 1 ||
			len(fn.Recv.List[0].Names) == 0 {
			continue
		}

		recvName := fn.Recv.List[0].Names[0].Name
		fields := conns[receiverTypeName(fn.Recv.List[0].Type)]
		if len(fields) == 0 || takesTransaction(fn.Type) ||
			(!referencesConnection(fn.Body, recvName, fields) && !executesQuery(fn.Body)) {
			continue
		}

		if !runsInScopedTransaction(fn.Body, recvName, fields) {
			violations = append(violations, TxViolation{
				Key:      key,
				Position: position,
				Problem:  "touches the connection outside dbtx.Read/Write or WithTx",
			})
		}
	}

	return violations
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	default:
		return ""
	}
}

func usesRawPool(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 0 {
			return !found
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "DB" {
			return !found
		}
		if _, isSel := sel.X.(*ast.SelectorExpr); isSel {
			found = true
		}
		return !found
	})

	return found
}

func takesTransaction(ft *ast.FuncType) bool {
	for _, field := range ft.Params.List {
		sel, ok := field.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		pkg, isIdent := sel.X.(*ast.Ident)
		if isIdent && pkg.Name == "bun" && (sel.Sel.Name == "IDB" || sel.Sel.Name == "Tx") {
			return true
		}
	}

	return false
}

var queryTerminals = map[string]bool{
	"Scan": true, "ScanAndCount": true, "Count": true, "Exists": true, "Exec": true,
	"Rows": true, "QueryContext": true, "QueryRowContext": true, "ExecContext": true,
}

func executesQuery(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		if _, isLit := n.(*ast.FuncLit); isLit {
			return true
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !queryTerminals[sel.Sel.Name] {
			return true
		}
		if _, onCall := sel.X.(*ast.CallExpr); onCall {
			found = true
			return false
		}
		if ident, isIdent := sel.X.(*ast.Ident); isIdent && ident.Obj != nil {
			found = true
		}
		return !found
	})

	return found
}

func referencesConnection(body *ast.BlockStmt, recv string, fields map[string]bool) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, isIdent := sel.X.(*ast.Ident); isIdent && id.Name == recv &&
				fields[sel.Sel.Name] {
				found = true
			}
		}
		return !found
	})

	return found
}

func runsInScopedTransaction(body *ast.BlockStmt, recv string, fields map[string]bool) bool {
	if len(body.List) == 0 {
		return false
	}

	last, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	if !ok || len(last.Results) != 1 {
		return false
	}

	call, ok := last.Results[0].(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if pkg, isIdent := sel.X.(*ast.Ident); isIdent && pkg.Name == "dbtx" &&
		scopedHelpers[sel.Sel.Name] {
		return !connectionUsedBefore(body.List[:len(body.List)-1], recv, fields)
	}

	if pkg, isIdent := sel.X.(*ast.Ident); isIdent && pkg.Name == "postgres" &&
		sel.Sel.Name == "DeleteUnderAuditRetention" {
		return !connectionUsedBefore(body.List[:len(body.List)-1], recv, fields)
	}

	if sel.Sel.Name == "WithTx" || sel.Sel.Name == "RunScoped" || sel.Sel.Name == "RunDetached" {
		if inner, isSel := sel.X.(*ast.SelectorExpr); isSel && fields[inner.Sel.Name] {
			return !connectionUsedBefore(body.List[:len(body.List)-1], recv, fields)
		}
	}

	return false
}

func connectionUsedBefore(stmts []ast.Stmt, recv string, fields map[string]bool) bool {
	for _, stmt := range stmts {
		used := false
		ast.Inspect(stmt, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return !used
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "DBForContext" && sel.Sel.Name != "DB") {
				return !used
			}
			if inner, isSel := sel.X.(*ast.SelectorExpr); isSel {
				if id, isIdent := inner.X.(*ast.Ident); isIdent && id.Name == recv &&
					fields[inner.Sel.Name] {
					used = true
				}
			}
			return !used
		})
		if used {
			return true
		}
	}

	return false
}
