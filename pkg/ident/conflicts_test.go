package ident

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameHasConflicts(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  string
		want bool
	}{
		{
			name: "conflict with unnamed import",
			src: `
package whatever

import (
	"github.com/src/something/conflicts"
)
`,
			want: true,
		},
		{
			name: "conflict with named import",
			src: `
package whatever

import (
	conflicts "github.com/src/something/other"
)
`,
			want: true,
		},
		{
			name: "conflict with function",
			src: `
package whatever

func conflicts() error {
	return nil
}
`,
			want: true,
		},
		{
			name: "conflict with type decl",
			src: `
package whatever

type conflicts int
`,
			want: true,
		},
		{
			name: "conflict with function argument",
			src: `
package whatever

func doIt(conflicts bool) error {
   return nil
}
`,
			want: true,
		},
		{
			name: "conflict with function result",
			src: `
package whatever

func check() (conflicts bool, err error) {
	return true, nil
}
`,
			want: true,
		},
		{
			name: "conflict with const",
			src: `
package whatever

const conflicts = 100
`,
			want: true,
		},
		{
			name: "conflict with multiple consts",
			src: `
package whatever

const (
	foo = 200
	conflicts = 100
)
`,
			want: true,
		},
		{
			name: "conflict with global var",
			src: `
package whatever

var conflicts = true
`,
			want: true,
		},

		{
			name: "conflict with multiple var",
			src: `
package whatever

var (
	foo = 200
	conflicts = true
)
`,
			want: true,
		},
		{
			name: "conflict with nested var",
			src: `
package whatever

func doIt() {
	{
		something, conflicts := 200, 100
	}
}

`,
			want: true,
		},
		{
			name: "no conflict for struct field",
			src: `
package whatever

type Foo struct {
	conflicts int
}

`,
			want: false,
		},
		{
			name: "conflict with hidden type",
			src: `
package whatever

func doIt() {
	type conflicts struct {}
}

`,
			want: true,
		},
	} {
		t.Run(tt.name, func(_ *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.src, parser.ParseComments)
			if !assert.NoError(t, err) {
				return
			}

			b := HasConflict(file, "conflicts", nil)
			assert.Equal(t, tt.want, b)
		})
	}
}
