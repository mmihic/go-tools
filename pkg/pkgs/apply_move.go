package pkgs

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/mmihic/go-tools/pkg/imports"
	"github.com/mmihic/go-tools/pkg/path"
	"github.com/mmihic/go-tools/pkg/scope"
)

// Apply updates all of the imports in the given file to reflect the new package
// locations. Returns the set of modified files.
func (moves Moves) Apply(fset *token.FileSet, pkgPath path.Path, f *ast.File) (bool, error) {
	changed := false

	// NB(mmihic): The order here is important - we first need to change all of the imports, so that
	// when we rewrite our package we can identity and remove self-imports
	if moves.updateImports(fset, f) {
		changed = true
	}

	if pkgPathMatch := moves.ExactMatch(pkgPath); pkgPathMatch != nil {
		if pkgPathMatch.rewritePackage(fset, f) {
			changed = true
		}
	}

	return changed, nil
}

// updateImports updates the imports in the given file to match the set of moves.
func (moves Moves) updateImports(fset *token.FileSet, f *ast.File) bool {
	// Find the best match for each import, and then use this to rewrite all of the
	// references to that import.
	changed := false
	for _, imp := range f.Imports {
		importPath := imports.Path(imp)
		importMatch := moves.BestMatch(importPath)
		if importMatch == nil {
			continue
		}

		oldName := imports.Name(imp)
		rewrittenPath, _ := importMatch.Rewrite(importPath)
		imp.Path.Value = strconv.Quote(rewrittenPath.String())

		if oldName == "_" {
			continue
		}

		newName := imports.DisambiguateImportName(f, rewrittenPath)
		if newName == rewrittenPath.PkgName() {
			// Can just rely on the default package name
			imp.Name = nil
		} else {
			// we need to use an alias to disambiguate
			imp.Name = &ast.Ident{
				Name: newName,
			}
		}

		rewriteImportPrefix(f, oldName, newName)
		changed = true
	}

	return changed
}

// rewriteImportPrefix changes the alias used for an import from one name to another.
func rewriteImportPrefix(f *ast.File, oldName, newName string) {
	scope.Inspect(f, func(n ast.Node, s *scope.Scope) bool {
		if s.HasDecl(oldName) {
			return false
		}

		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if ident.Name == oldName {
					ident.Name = newName
				}
			}
		}
		return true
	})
}

// rewritePackage changes the package to which the given file belongs.
func (mv *Move) rewritePackage(fset *token.FileSet, f *ast.File) bool {
	// Change package decl
	oldName := f.Name.Name
	newName := mv.To.PkgName()
	f.Name.Name = newName

	// Rewrite the package comments, if any
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			lineCommentPrefix := fmt.Sprintf(`// Package %s `, oldName)
			if strings.HasPrefix(c.Text, lineCommentPrefix) {
				remaining := c.Text[len(lineCommentPrefix):]
				c.Text = fmt.Sprintf(`// Package %s %s`, newName, remaining)
				break
			}

			blockCommentPrefix := fmt.Sprintf(`/* Package %s `, oldName)
			if strings.HasPrefix(c.Text, blockCommentPrefix) {
				remaining := c.Text[len(blockCommentPrefix):]
				c.Text = fmt.Sprintf(`/* Package %s %s`, newName, remaining)
				break
			}
		}
	}

	removeSelfImport(fset, f, mv.To)
	return true
}

// removeSelfImport removes an import statement that now refers to the package
// in which the file resides, also removing any name qualifier on declarations
// that previously used the imported package.
func removeSelfImport(fset *token.FileSet, f *ast.File, pkgPath path.Path) {
	// Check to see if we import our new path - if so strip that import.
	for _, imp := range f.Imports {
		importPath := imports.Path(imp)
		if !pkgPath.Equal(importPath) {
			continue
		}

		astutil.DeleteImport(fset, f, importPath.String())
		removeImportPrefix(f, imports.Name(imp))
	}
}

func removeImportPrefix(f *ast.File, prefix string) {
	scope.Inspect(f, func(nth ast.Node, s *scope.Scope) bool {
		switch n := nth.(type) {
		case *ast.Field:
			maybeRemoveImportPrefix(&n.Type, prefix)
		case *ast.StarExpr:
			maybeRemoveImportPrefix(&n.X, prefix)
		case *ast.Ellipsis:
			maybeRemoveImportPrefix(&n.Elt, prefix)
		case *ast.ArrayType:
			maybeRemoveImportPrefix(&n.Elt, prefix)
		case *ast.ChanType:
			maybeRemoveImportPrefix(&n.Value, prefix)
		case *ast.MapType:
			maybeRemoveImportPrefix(&n.Key, prefix)
		case *ast.CallExpr:
			maybeRemoveImportPrefix(&n.Fun, prefix)
		case *ast.ValueSpec:
			maybeRemoveImportPrefix(&n.Type, prefix)
		}
		return true
	})
}

func maybeRemoveImportPrefix(expr *ast.Expr, name string) {
	if expr == nil {
		return
	}

	sel, ok := (*expr).(*ast.SelectorExpr)
	if !ok {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	if ident.Name == name {
		*expr = ast.NewIdent(sel.Sel.Name)
	}
}
