// Package comments contains functions for dealing with comments in an ast.
package comments

import (
	"go/ast"
	"go/token"
	"sort"
)

// A Map is a map of nodes to comments. Works better than an ast.CommentMap
type Map struct {
	comments map[token.Pos][]*ast.Comment
}

// NewMap creates a new map of comments to nodes. A comment is associated with a node
// if its position in the file precedes that node and there are no intervening nodes.
func NewMap(fset *token.FileSet, n ast.Node, groups []*ast.CommentGroup) *Map {
	var comments []*ast.Comment
	for _, group := range groups {
		for _, c := range group.List {
			comments = append(comments, c)
		}
	}

	sort.Slice(comments, func(i, j int) bool {
		return comments[i].Pos() < comments[j].Pos()
	})

	commentsByNodePos := map[token.Pos][]*ast.Comment{}
	ast.Inspect(n, func(nth ast.Node) bool {
		if nth == nil {
			return true
		}

		var nodeComments []*ast.Comment
		for len(comments) > 0 && comments[0].Pos() < nth.Pos() {
			nodeComments = append(nodeComments, comments[0])
			comments = comments[1:]
		}

		commentsByNodePos[nth.Pos()] = nodeComments
		return true
	})

	return &Map{
		comments: commentsByNodePos,
	}
}

// CommentsForNode returns the comments for the given node.
func (m *Map) CommentsForNode(n ast.Node) []*ast.Comment {
	return m.comments[n.Pos()]
}
