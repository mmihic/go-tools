// Package pkgalign is a tool to align package imports and package names after
// a bulk package restructuring. It takes a source package path, a destination
// package path, and a target name for the resulting package. The source within
// the destination package has its package label set to the new name, and
// all sources that import the original package now import the new package
// at the destination directory and with the new name.
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

// RewriteRule tells us which imports to rewrite.
type RewriteRule struct {
	From    string
	To      string
	PkgName string
}

// RewritePostMove rewrites a file to deal with the move of a package from one
// location to another, along with a possible package rename.
func RewritePostMove(pkg *packages.Package, rules []*RewriteRule) {
	for _, f := range pkg.Syntax {
		if rewrite(pkg, f, rules) {
			write(pkg.Fset, f)
		}
	}
}

func rewrite(pkg *packages.Package, f *ast.File, rules []*RewriteRule) bool {
	changed := false
	for _, rule := range rules {
		if processRule(pkg, f, rule) {
			changed = true
		}
	}

	return changed
}

func processRule(pkg *packages.Package, f *ast.File, rule *RewriteRule) bool {
	if pkg.PkgPath == rule.From {
		return rewritePackage(pkg, f, rule)
	}

	return rewriteImport(f, rule)
}

func rewritePackage(pkg *packages.Package, f *ast.File, rule *RewriteRule) bool {
	// Change package decl
	newName := rule.PkgName
	if newName == "" {
		_, newName = filepath.Split(rule.To)
	}

	oldName := pkg.Name
	f.Name.Name = newName

	// Rewrite the package comments, if any
	for _, cg := range f.Comments {
		c := cg.List[0]
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

	return false
}

func rewriteImport(f *ast.File, rule *RewriteRule) bool {
	newName := rule.PkgName
	if newName == "" {
		_, newName = filepath.Split(rule.To)
	}

	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path != rule.From {
			continue
		}

		imp.Path.Value = strconv.Quote(rule.To)
		if imp.Name == nil {
			// They are using the default name, not an alias, so rewrite all references using that name
			rewriteRefs(f, filepath.Base(rule.From), newName)
		}
		return true
	}

	return false
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

var writeFile = func(filename string, content []byte) error {
	return ioutil.WriteFile(filename, content, 0644)
}

func write(fset *token.FileSet, f *ast.File) {
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
