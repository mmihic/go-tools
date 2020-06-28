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
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// RewriteRule tells us which imports to rewriteFile.
type RewriteRule struct {
	From string
	To   string
}

// PkgName returns the name of the package.
func (rule *RewriteRule) PkgName() string {
	_, pkgName := filepath.Split(rule.To)
	return pkgName
}

// Rewrite rewrites a file to deal with the move of a package from one
// location to another, along with a possible package rename.
func Rewrite(pkg *packages.Package, rules []*RewriteRule) {
	rewriteFiles(pkg.Fset, pkg.PkgPath, pkg.Syntax, rules,
		func(filename string, content []byte) error {
			return ioutil.WriteFile(filename, content, 0644)
		})
}

func rewriteFiles(
	fset *token.FileSet,
	pkgPath string,
	files []*ast.File,
	rules []*RewriteRule,
	writeFile WriteFileFn,
) {
	for _, f := range files {
		rewriteFile(fset, pkgPath, f, rules, writeFile)
	}
}

func rewriteFile(
	fset *token.FileSet, pkgPath string, f *ast.File, rules []*RewriteRule, writeFile WriteFileFn,
) {
	changed := false
	for _, rule := range rules {
		if processRule(pkgPath, f, rule) {
			changed = true
		}
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

func processRule(pkgPath string, f *ast.File, rule *RewriteRule) bool {
	if pkgPath == rule.From {
		return rewritePackage(f, rule)
	}

	return rewriteImport(f, rule)
}

func rewritePackage(f *ast.File, rule *RewriteRule) bool {
	// Change package decl
	oldName := f.Name.Name
	newName := rule.PkgName()
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

	return true
}

func rewriteImport(f *ast.File, rule *RewriteRule) bool {
	// Find a non-conflicting alias to use for the package.
	newName := disambiguateImportName(f.Imports, rule.PkgName())
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path != rule.From {
			continue
		}

		imp.Path.Value = strconv.Quote(rule.To)
		if newName != rule.PkgName() {
			// we need to use an alias to disambiguate
			imp.Name = &ast.Ident{
				Name: newName,
			}
		}
		rewriteRefs(f, filepath.Base(rule.From), newName)
		return true
	}

	return false
}

func getImportedPkgNames(imports []*ast.ImportSpec) map[string]struct{} {
	importedPkgNames := make(map[string]struct{}, len(imports))
	for _, imp := range imports {
		if imp.Name != nil {
			importedPkgNames[imp.Name.Name] = struct{}{}
			continue
		}

		pkgPath, _ := strconv.Unquote(imp.Path.Value)
		_, importName := filepath.Split(pkgPath)
		importedPkgNames[importName] = struct{}{}
	}

	return importedPkgNames
}

func disambiguateImportName(imports []*ast.ImportSpec, originalName string) string {
	importedPkgNames := getImportedPkgNames(imports)

	var (
		name = originalName
		n    = 2
	)

	for {
		if _, conflicts := importedPkgNames[name]; !conflicts {
			return name
		}
		name = fmt.Sprintf("%s%d", originalName, n)
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

// WriteFileFn is a function that writes a file.
type WriteFileFn func(filename string, content []byte) error
