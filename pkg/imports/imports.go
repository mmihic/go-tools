// Package imports has functions for dealing with imports.
package imports

import (
	"go/ast"
	"strconv"

	"github.com/mmihic/go-tools/pkg/ident"
	"github.com/mmihic/go-tools/pkg/path"
)

// Name returns the name of the import.
func Name(imp *ast.ImportSpec) string {
	if imp.Name != nil {
		return imp.Name.Name
	}
	return ident.Clean(Path(imp).PkgName())
}

// Path returns the path of the import.
func Path(imp *ast.ImportSpec) path.Path {
	val, _ := strconv.Unquote(imp.Path.Value)
	return path.NewPath(val)
}
