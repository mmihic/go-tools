package imports

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mmihic/go-tools/pkg/path"
)

func TestDisambiguateImportName(t *testing.T) {
	for _, tt := range []struct {
		name    string
		pkgPath string
		src     string
		want    string
	}{
		{
			name:    "name conflicts with other import",
			pkgPath: "github.com/tools/other",
			src: `
package whatever

import (
	"github.com/something/other"
)
`,
			want: "toolsother",
		},
		{
			name:    "name conflicts with function decl",
			pkgPath: "github.com/tools/other",
			src: `
package whatever

func other() {
}
`,
			want: "toolsother",
		},
		{
			name:    "name conflicts with type decl",
			pkgPath: "github.com/tools/other",
			src: `
package whatever

type other int
`,
			want: "toolsother",
		},
		{
			name:    "name conflicts with var inside function",
			pkgPath: "github.com/tools/other",
			src: `
package whatever

func something() {
  var other = 200
}
`,
			want: "toolsother",
		},
		{
			name:    "package already imported - should not show as conflict",
			pkgPath: "github.com/tools/other",
			src: `
package whatever

import (
	"github.com/tools/other"
)
`,
			want: "other",
		},
		{
			name:    "combo of name + parent conflicts with other import",
			pkgPath: "github.com/tools/other",
			src: `
package whatever

func something() {
  var other, toolsother = 200, 300
}
`,
			want: "other2",
		},
		{
			name:    "combo of name + parent ignored since parent is a common name",
			pkgPath: "github.com/pkg/other",
			src: `
package whatever

func something() {
  var other = 200
}
`,
			want: "other2",
		},
		{
			name:    "multiple conflicts",
			pkgPath: "github.com/pkg/other",
			src: `
package whatever

func something() {
  var other, other2, other3 = 200
}
`,
			want: "other4",
		},
		{
			name:    "cleans name",
			pkgPath: "github.com/pkg/this-other",
			src: `
package whatever

func something() {
}
`,
			want: "thisother",
		},
		{
			name:    "cleans parent",
			pkgPath: "github.com/mmihic-tools/other",
			src: `
package whatever

const other = 100
`,
			want: "mmihictoolsother",
		},
	} {
		t.Run(tt.name, func(_ *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.src, parser.ParseComments)
			if !assert.NoError(t, err) {
				return
			}

			name := DisambiguateImportName(file, path.NewPath(tt.pkgPath))
			assert.Equal(t, tt.want, name)
		})
	}
}
