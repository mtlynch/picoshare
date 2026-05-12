package httperrorreturn

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Issue struct {
	Path    string
	Line    int
	Column  int
	Message string
}

type afterState struct {
	next        ast.Stmt
	mayContinue bool
}

func CheckPaths(paths ...string) ([]Issue, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	filePaths, err := goFiles(paths)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	var issues []Issue
	for _, path := range filePaths {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		issues = append(issues, checkFile(fset, path, file)...)
	}

	slices.SortFunc(issues, func(a, b Issue) int {
		if diff := strings.Compare(a.Path, b.Path); diff != 0 {
			return diff
		}
		if a.Line != b.Line {
			return a.Line - b.Line
		}
		return a.Column - b.Column
	})

	return issues, nil
}

func goFiles(paths []string) ([]string, error) {
	seen := map[string]bool{}
	var filePaths []string

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			if err := filepath.WalkDir(path, func(walkPath string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					if strings.HasPrefix(d.Name(), ".") && walkPath != "." {
						return filepath.SkipDir
					}
					return nil
				}
				if filepath.Ext(walkPath) != ".go" {
					return nil
				}
				if seen[walkPath] {
					return nil
				}
				seen[walkPath] = true
				filePaths = append(filePaths, walkPath)
				return nil
			}); err != nil {
				return nil, err
			}
			continue
		}
		if filepath.Ext(path) != ".go" {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		filePaths = append(filePaths, path)
	}

	slices.Sort(filePaths)
	return filePaths, nil
}

type checker struct {
	fset      *token.FileSet
	path      string
	httpNames map[string]bool
	issues    []Issue
}

func checkFile(fset *token.FileSet, path string, file *ast.File) []Issue {
	c := checker{
		fset:      fset,
		path:      path,
		httpNames: httpImportNames(file),
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			c.checkBlock(node.Body, afterState{})
		case *ast.FuncLit:
			c.checkBlock(node.Body, afterState{})
		}
		return true
	})

	return c.issues
}

func httpImportNames(file *ast.File) map[string]bool {
	names := map[string]bool{}
	for _, spec := range file.Imports {
		if spec.Path == nil || spec.Path.Value != `"net/http"` {
			continue
		}
		if spec.Name == nil {
			names["http"] = true
			continue
		}
		if spec.Name.Name == "." || spec.Name.Name == "_" {
			continue
		}
		names[spec.Name.Name] = true
	}
	return names
}

func (c *checker) checkBlock(block *ast.BlockStmt, after afterState) {
	if block == nil {
		return
	}
	c.checkStmtList(block.List, after)
}

func (c *checker) checkStmtList(stmts []ast.Stmt, after afterState) {
	for i, stmt := range stmts {
		next := after
		if i+1 < len(stmts) {
			next = afterState{
				next:        stmts[i+1],
				mayContinue: true,
			}
		}
		c.checkStmt(stmt, next)
	}
}

func (c *checker) checkStmt(stmt ast.Stmt, after afterState) {
	switch node := stmt.(type) {
	case *ast.BlockStmt:
		c.checkBlock(node, after)
	case *ast.IfStmt:
		c.checkBlock(node.Body, after)
		c.checkElse(node.Else, after)
	case *ast.ForStmt:
		c.checkBlock(node.Body, afterState{mayContinue: true})
	case *ast.RangeStmt:
		c.checkBlock(node.Body, afterState{mayContinue: true})
	case *ast.SwitchStmt:
		c.checkCaseClauses(node.Body.List, after)
	case *ast.TypeSwitchStmt:
		c.checkCaseClauses(node.Body.List, after)
	case *ast.SelectStmt:
		c.checkCommClauses(node.Body.List, after)
	case *ast.LabeledStmt:
		c.checkStmt(node.Stmt, after)
	case *ast.ExprStmt:
		if c.isHTTPErrorCall(node.X) && requiresTerminator(after) {
			position := c.fset.Position(node.Pos())
			c.issues = append(c.issues, Issue{
				Path:    c.path,
				Line:    position.Line,
				Column:  position.Column,
				Message: "http.Error call can fall through to later code; add a terminating statement",
			})
		}
	}
}

func (c *checker) checkElse(stmt ast.Stmt, after afterState) {
	switch node := stmt.(type) {
	case nil:
		return
	case *ast.BlockStmt:
		c.checkBlock(node, after)
	case *ast.IfStmt:
		c.checkStmt(node, after)
	default:
		c.checkStmt(node, after)
	}
}

func (c *checker) checkCaseClauses(items []ast.Stmt, after afterState) {
	for _, item := range items {
		clause, ok := item.(*ast.CaseClause)
		if !ok {
			continue
		}
		c.checkStmtList(clause.Body, after)
	}
}

func (c *checker) checkCommClauses(items []ast.Stmt, after afterState) {
	for _, item := range items {
		clause, ok := item.(*ast.CommClause)
		if !ok {
			continue
		}
		c.checkStmtList(clause.Body, after)
	}
}

func requiresTerminator(after afterState) bool {
	if after.next != nil {
		return !isTerminatingStmt(after.next)
	}
	return after.mayContinue
}

func isTerminatingStmt(stmt ast.Stmt) bool {
	switch node := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		switch node.Tok {
		case token.BREAK, token.CONTINUE, token.GOTO, token.FALLTHROUGH:
			return true
		}
	case *ast.BlockStmt:
		if len(node.List) == 0 {
			return false
		}
		return isTerminatingStmt(node.List[0])
	}
	return false
}

func (c *checker) isHTTPErrorCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if selector.Sel == nil || selector.Sel.Name != "Error" {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	return c.httpNames[ident.Name]
}
