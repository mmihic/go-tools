// Package astio contains IO handling for ast elements.
package astio

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"os"
)

// String converts the given node to a string.
func String(fset *token.FileSet, n ast.Node) (string, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, n); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// WriteFile writes the given file.
func WriteFile(fset *token.FileSet, f *ast.File) error {
	fname := fset.File(f.Pos())
	file, err := os.Create(fname.Name())
	if err != nil {
		return err
	}

	defer func() {
		_ = file.Close()
	}()

	return format.Node(file, fset, f)
}
