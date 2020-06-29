package pkgalign

import (
	"path/filepath"
	"strings"
)

// A Path is a path.
type Path []string

// NewPath creates a new path.
func NewPath(s string) Path {
	return strings.Split(s, "/")
}

// String returns a string form of the path matcher.
func (p Path) String() string {
	return filepath.Join(p...)
}

// Equal compares to paths for equality.
func (p Path) Equal(other Path) bool {
	if len(p) != len(other) {
		return false
	}

	for i, pathElt := range p {
		if pathElt != other[i] {
			return false
		}
	}

	return true
}

// Contains checks whether the path contains the other path.
func (p Path) Contains(other Path) bool {
	if len(p) > len(other) {
		return false
	}

	for i, pathElt := range p {
		if pathElt != other[i] {
			return false
		}
	}

	return true
}

// PkgName returns the name of the package.
func (p Path) PkgName() string {
	return p[len(p)-1]
}

