package imports

import (
	"fmt"
	"go/ast"
	"strconv"

	"github.com/mmihic/go-tools/pkg/ident"
	"github.com/mmihic/go-tools/pkg/path"
)

var (
	commonPkgNames = map[string]struct{}{
		"pkg":      {},
		"internal": {},
		"src":      {},
	}
)


// DisambiguateImportName finds a non-conflicting name for the given import path.
func DisambiguateImportName(root ast.Node, importPath path.Path) string {
	// Ignore conflicts with an import of ourselves
	skipSelf := func(n ast.Node) bool {
		imp, ok := n.(*ast.ImportSpec)
		if !ok {
			return false
		}

		return imp.Path.Value == strconv.Quote(importPath.String())
	}

	// First try the name itself
	pkgName := ident.Clean(importPath.PkgName())
	if !ident.HasConflict(root, pkgName, skipSelf) {
		return pkgName
	}

	// Next try a combination of our name + the parent name, if the parent is not a generic
	// name like [pkg, internal, src, etc]
	if len(importPath) > 1 {
		parentPkgName := ident.Clean(importPath[len(importPath)-2])
		if _, commonPkgName := commonPkgNames[parentPkgName]; !commonPkgName {
			comboPkgName := parentPkgName + pkgName
			if !ident.HasConflict(root, comboPkgName, skipSelf) {
				return comboPkgName
			}
		}
	}

	// Now start appending numbers to the package name until we find one that works
	n := 2
	for {
		importName := fmt.Sprintf("%s%d", pkgName, n)
		if !ident.HasConflict(root, importName, skipSelf) {
			return importName
		}

		n++
	}

}



