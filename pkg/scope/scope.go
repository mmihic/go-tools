// Package scope tracks scopes and declarations in scope.
package scope

import (
	"go/ast"

	"github.com/mmihic/go-tools/pkg/imports"
)

// Visitor visits nodes and has access to the scoping information at that node.
type Visitor interface {
	Visit(n ast.Node, scope *Scope) Visitor
}

// Walk visits nodes with scoping information.
func Walk(v Visitor, n ast.Node) {
	scope := &Scope{
		v:     v,
		decls: map[string]ast.Node{},
	}

	ast.Walk(scope, n)
}

// Inspect inspects the nodes with scoping information.
func Inspect(n ast.Node, f func(ast.Node, *Scope) bool) {
	Walk(inspector(f), n)
}

type inspector func(ast.Node, *Scope) bool

func (f inspector) Visit(node ast.Node, scope *Scope) Visitor {
	if f(node, scope) {
		return f
	}
	return nil
}

// Scope tracks declarations in scope.
type Scope struct {
	parent *Scope
	decls  map[string]ast.Node
	v      Visitor
}

// HasDecl returns true if the a decl is in scope.
func (s *Scope) HasDecl(name string) bool {
	return s.GetDecl(name) != nil
}

// GetDecl returns the in-scope declaration with the given name.
func (s *Scope) GetDecl(name string) ast.Node {
	if decl, ok := s.decls[name]; ok {
		return decl
	}

	if s.parent != nil {
		return s.parent.GetDecl(name)
	}

	return nil
}

// Decls returns all of the declarations in scope.
func (s *Scope) Decls() map[string]ast.Node {
	var decls map[string]ast.Node
	if s.parent != nil {
		decls = s.parent.Decls()
	} else {
		decls = map[string]ast.Node{}
	}

	for name, decl := range s.decls {
		decls[name] = decl
	}
	return decls
}

func (s *Scope) addDecl(name string, n ast.Node) {
	s.decls[name] = n
}

func (s *Scope) enter() *Scope {
	return &Scope{
		parent: s,
		decls:  map[string]ast.Node{},
		v:      s.v,
	}
}

func (s *Scope) withVisitor(visitor Visitor) *Scope {
	return &Scope{
		parent: s.parent,
		decls:  s.decls,
		v:      visitor,
	}
}

// Walk visits a node, adding declarations or pushing a new block onto the scope.
func (s *Scope) Visit(nth ast.Node) ast.Visitor {
	ret := s.visit(nth)
	visitor := ret.v.Visit(nth, ret)
	if visitor == nil {
		return nil
	}

	return ret.withVisitor(visitor)
}

func (s *Scope) visit(nth ast.Node) *Scope {
	switch n := nth.(type) {
	case *ast.FuncDecl:
		s.addDecl(n.Name.Name, n)
		return s.enter()
	case *ast.FuncType:
		if n.Params != nil {
			for _, f := range n.Params.List {
				for _, nm := range f.Names {
					s.addDecl(nm.Name, f)
				}
			}
		}

		if n.Results != nil {
			for _, f := range n.Results.List {
				for _, nm := range f.Names {
					s.addDecl(nm.Name, f)
				}
			}
		}
	case *ast.BlockStmt:
		return s.enter()
	case *ast.ValueSpec:
		for _, nm := range n.Names {
			s.addDecl(nm.Name, n)
		}
	case *ast.TypeSpec:
		s.addDecl(n.Name.Name, n)
	case *ast.ImportSpec:
		s.addDecl(imports.Name(n), n)
	case *ast.AssignStmt:
		for _, lhs := range n.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}

			s.addDecl(ident.Name, n)
		}
	}

	return s
}
