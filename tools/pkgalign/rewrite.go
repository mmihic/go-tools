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
)

// Rewrite rewrites a file to deal with the move of a package from one
// location to another, along with a possible package rename.
func Rewrite(fset *token.FileSet, pkgPath string, pkg *ast.Package, rules RewriteRules) {
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
	pkgPath string,
	files []*ast.File,
	rules RewriteRules,
	writeFile WriteFileFn,
) {
	for _, f := range files {
		rewriteFile(fset, pkgPath, f, rules, writeFile)
	}
}

func rewriteFile(
	fset *token.FileSet, pkgPath string, f *ast.File, rules RewriteRules, writeFile WriteFileFn,
) {
	changed := false
	if pkgPathMatch := rules.ExactMatch(NewPath(pkgPath)); pkgPathMatch != nil {
		if rewritePackage(f, pkgPathMatch) {
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

func rewritePackage(f *ast.File, rule *RewriteRule) bool {
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

	return true
}

func rewriteImports(f *ast.File, rules RewriteRules) bool {
	// Find the best match for each import, and then use this to rewrite all of the
	// references to that import.
	changed := false
	for _, imp := range f.Imports {
		importPathStr, _ := strconv.Unquote(imp.Path.Value)
		importPath := NewPath(importPathStr)
		importMatch := rules.BestMatch(importPath)
		if importMatch == nil {
			continue
		}

		oldName := importPath.PkgName()
		if imp.Name != nil {
			oldName = imp.Name.Name
		}

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

		fmt.Printf("rewriting %30s to %30s as %s\n", importPath, rewrittenPath, newName)
		rewriteRefs(f, oldName, newName)
		changed = true
	}

	return changed
}

func getImportedPkgNames(imports []*ast.ImportSpec, skipFilter func(importPath Path) bool) map[string]struct{} {
	importedPkgNames := make(map[string]struct{}, len(imports))
	for _, imp := range imports {
		pkgPath, _ := strconv.Unquote(imp.Path.Value)
		if skipFilter(NewPath(pkgPath)) {
			continue
		}

		if imp.Name != nil {
			importedPkgNames[imp.Name.Name] = struct{}{}
			continue
		}

		_, importName := filepath.Split(pkgPath)
		importedPkgNames[importName] = struct{}{}
	}

	return importedPkgNames
}

func disambiguateImportName(imports []*ast.ImportSpec, importPath Path) string {
	importedPkgNames := getImportedPkgNames(imports, func(p Path) bool {
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

// WriteFileFn is a function that writes a file.
type WriteFileFn func(filename string, content []byte) error
