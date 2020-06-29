// Package pkgalign is a tool to align package imports and package names after
// a bulk package restructuring. It takes a source package path and a destination
// package path. The source within the destination package has its package
// set to the name of the destination directory, and all sources that import the
// original package now import the new package at the destination directory.
package pkgalign

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/mmihic/go-tools/pkg/path"
)

// Rewrite rewrites a file to deal with the move of a package from one
// location to another, along with a possible package rename.
func Rewrite(fset *token.FileSet, pkgPath path.Path, pkg *ast.Package, rules RewriteRules) {
	files := make([]*ast.File, 0, len(pkg.Files))
	for _, file := range pkg.Files {
		files = append(files, file)
	}

	rewriteFiles(fset, pkgPath, files, rules, func(filename string, content []byte) error {
		return ioutil.WriteFile(filename, content, 0644)
	})
}

func rewriteFiles(
	fset *token.FileSet,
	pkgPath path.Path,
	files []*ast.File,
	rules RewriteRules,
	writeFile WriteFileFn,
) {
	for _, f := range files {
		rewriteFile(fset, pkgPath, f, rules, writeFile)
	}
}

func rewriteFile(
	fset *token.FileSet, pkgPath path.Path, f *ast.File, rules RewriteRules, writeFile WriteFileFn,
) {
	changed := false
	if pkgPathMatch := rules.ExactMatch(pkgPath); pkgPathMatch != nil {
		if rewritePackage(fset, f, pkgPathMatch) {
			changed = true
		}
	}

	if rewriteImports(f, rules) {
		changed = true
	}

	if !changed {
		return
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		log.Printf("failed to pretty-print syntax tree: %v", err)
		return
	}
	tokenFile := fset.File(f.Pos())
	if err := writeFile(tokenFile.Name(), buf.Bytes()); err != nil {
		log.Printf("failed to write file %s: %v", tokenFile.Name(), err)
	}
}

func rewritePackage(fset *token.FileSet, f *ast.File, rule *RewriteRule) bool {
	// Change package decl
	oldName := f.Name.Name
	newName := rule.To.PkgName()
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

	// Check to see if we import our new path - if so strip that import.
	for _, imp := range f.Imports {
		importPath := getImportPath(imp)
		if !rule.To.Equal(importPath) {
			continue
		}

		astutil.DeleteImport(fset, f, importPath.String())
		removeImportPrefix(f, getImportName(imp))
	}

	return true
}

func rewriteImports(f *ast.File, rules RewriteRules) bool {
	// Find the best match for each import, and then use this to rewrite all of the
	// references to that import.
	changed := false
	for _, imp := range f.Imports {
		importPath := getImportPath(imp)
		importMatch := rules.BestMatch(importPath)
		if importMatch == nil {
			continue
		}

		oldName := getImportName(imp)

		rewrittenPath, _ := importMatch.Rewrite(importPath)
		imp.Path.Value = strconv.Quote(rewrittenPath.String())

		newName := disambiguateImportName(f.Imports, rewrittenPath)
		if newName == rewrittenPath.PkgName() {
			// Can just rely on the default package name
			imp.Name = nil
		} else {
			// we need to use an alias to disambiguate
			imp.Name = &ast.Ident{
				Name: newName,
			}
		}

		rewriteRefs(f, oldName, newName)
		changed = true
	}

	return changed
}

func getImportPath(imp *ast.ImportSpec) path.Path {
	importPathStr, _ := strconv.Unquote(imp.Path.Value)
	return path.NewPath(importPathStr)
}

func getImportName(imp *ast.ImportSpec) string {
	if imp.Name != nil {
		return imp.Name.Name
	}
	return getImportPath(imp).PkgName()
}

func getImportedPkgNames(imports []*ast.ImportSpec, skipFilter func(importPath path.Path) bool) map[string]struct{} {
	importedPkgNames := make(map[string]struct{}, len(imports))
	for _, imp := range imports {
		pkgPath := getImportPath(imp)
		if skipFilter(pkgPath) {
			continue
		}

		importName := getImportName(imp)
		importedPkgNames[importName] = struct{}{}
	}

	return importedPkgNames
}

func disambiguateImportName(imports []*ast.ImportSpec, importPath path.Path) string {
	importedPkgNames := getImportedPkgNames(imports, func(p path.Path) bool {
		return importPath.Equal(p)
	})

	// Try the raw name
	if _, conflicts := importedPkgNames[importPath.PkgName()]; !conflicts {
		return importPath.PkgName()
	}

	// Try the name combined with the parent if the parent isn't something generic like internal or pkg
	if parentName := importPath[len(importPath)-2]; parentName != "pkg" && parentName != "internal" {
		nameWithParent := fmt.Sprintf("%s%s", parentName, importPath.PkgName())
		if _, conflicts := importedPkgNames[nameWithParent]; !conflicts {
			return nameWithParent
		}
	}

	n := 2
	for {
		name := fmt.Sprintf("%s%d", importPath.PkgName(), n)
		if _, conflicts := importedPkgNames[name]; !conflicts {
			return name
		}
		n++
	}
}

func rewriteRefs(f *ast.File, oldName, newName string) {
	ast.Inspect(f, func(n ast.Node) bool {
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

func removeImportPrefix(f *ast.File, name string) {
	ast.Inspect(f, func(nth ast.Node) bool {
		switch n := nth.(type) {
		case *ast.Field:
			maybeRemoveImportPrefix(&n.Type, name)
		case *ast.StarExpr:
			maybeRemoveImportPrefix(&n.X, name)
		case *ast.Ellipsis:
			maybeRemoveImportPrefix(&n.Elt, name)
		case *ast.ArrayType:
			maybeRemoveImportPrefix(&n.Elt, name)
		case *ast.ChanType:
			maybeRemoveImportPrefix(&n.Value, name)
		case *ast.MapType:
			maybeRemoveImportPrefix(&n.Key, name)
		case *ast.CallExpr:
			maybeRemoveImportPrefix(&n.Fun, name)
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

// WriteFileFn is a function that writes a file.
type WriteFileFn func(filename string, content []byte) error
