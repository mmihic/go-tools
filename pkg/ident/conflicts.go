// Package ident contains utilities for dealing with identifiers.
package ident

import (
	"go/ast"
	"path/filepath"
	"regexp"
	"strconv"
)

var (
	reNotIdent = regexp.MustCompile(`[^A-Za-z0-9_]`)
)

// Clean produces a clean identifier.
func Clean(ident string) string {
	return reNotIdent.ReplaceAllString(ident, "")
}

// HasConflict checks whether the given name conflicts with any
// types or declarations in the given node tree.
func HasConflict(root ast.Node, potentialName string, skip func(n ast.Node) bool) bool {
	if skip == nil {
		skip = func(_ ast.Node) bool { return false } // skip nothing
	}
	d := &conflictDetector{
		potentialName: potentialName,
		skip:          skip,
	}

	ast.Walk(d, root)
	return d.hasConflicts
}

type conflictDetector struct {
	potentialName string
	hasConflicts  bool
	skip          func(n ast.Node) bool
}

func (d *conflictDetector) Visit(nth ast.Node) ast.Visitor {
	if d.hasConflicts {
		return nil
	}

	if d.skip(nth) {
		return d
	}

	switch n := nth.(type) {
	case *ast.FuncDecl:
		d.checkConflict(n.Name)
	case *ast.FuncType:
		if n.Results != nil {
			for _, f := range n.Results.List {
				d.checkConflicts(f.Names)
			}
		}

		if n.Params != nil {
			for _, f := range n.Params.List {
				d.checkConflicts(f.Names)
			}
		}
	case *ast.AssignStmt:
		for _, lhs := range n.Lhs {
			d.checkConflict(lhs)
		}
	case *ast.ValueSpec:
		d.checkConflicts(n.Names)
	case *ast.TypeSpec:
		d.checkConflict(n.Name)
	case *ast.ImportSpec:
		path, _ := strconv.Unquote(n.Path.Value)
		_, importName := filepath.Split(path)
		if n.Name != nil {
			importName = n.Name.Name
		}

		if d.potentialName == importName {
			d.hasConflicts = true
		}
	}

	if d.hasConflicts {
		return nil
	}

	return d
}
func (d *conflictDetector) checkConflicts(idents []*ast.Ident) {
	for _, ident := range idents {
		if ident.Name == d.potentialName {
			d.hasConflicts = true
		}
	}
}

func (d *conflictDetector) checkConflict(expr ast.Expr) {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		d.checkConflict(sel.Sel)
	}

	if ident, ok := expr.(*ast.Ident); ok && ident != nil {
		d.hasConflicts = ident.Name == d.potentialName
	}
}
