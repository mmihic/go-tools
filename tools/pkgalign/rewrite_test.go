package pkgalign

import (
	"bytes"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mmihic/go-tools/pkg/path"
)

func TestRewritePostMove(t *testing.T) {
	for _, tt := range []struct {
		name    string
		pkgPath string
		src     string
		rules   []string
		want    string
	}{
		// ----------------
		{
			name:    "rewrite source package with line package comment",
			pkgPath: "github.com/mmihic/go-tools/pkg/first",
			src: `
// +build tools
// Package first is a package that does some things.

package first

func DoSomething() string { return "does something" }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
			},
			want: strings.TrimLeft(`
// +build tools
// Package other is a package that does some things.

package other

func DoSomething() string { return "does something" }
`, "\n"),
		},

		// ----------------
		{
			name:    "rewrite source package with block package comment",
			pkgPath: "github.com/mmihic/go-tools/pkg/first",
			src: `
// +build tools
/* Package first is a package that does some things. */

package first

func DoSomething() string { return "does something" }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
			},
			want: strings.TrimLeft(`
// +build tools
/* Package other is a package that does some things. */

package other

func DoSomething() string { return "does something" }
`, "\n"),
		},

		// ----------------
		{
			name:    "rewrite imported package without disambiguation",
			pkgPath: "github.com/mmihic/go-tools/pkg/imports",
			src: `
// +build tools
package imports

import (
   "github.com/mmihic/go-tools/pkg/first"
)

func DoSomething() string { return first.DoSomething() }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
			},
			want: strings.TrimLeft(`
// +build tools
package imports

import (
	"github.com/mmihic/go-tools/pkg/other"
)

func DoSomething() string { return other.DoSomething() }
`, "\n"),
		},

		// ----------------
		{
			name:
			"picks best match",
			pkgPath: "github.com/mmihic/go-tools/pkg/imports",
			src: `
// +build tools
package imports

import (
	"github.com/mmihic/go-tools/pkg/first"
	"github.com/mmihic/go-tools/pkg/first/something"
	"github.com/mmihic/go-tools/pkg/first/elise"
)

func DoSomething() string { return something.Do() }
func DoSomethingFirst() string { return first.DoSomething() }
func DoSomethingElise() string { return elise.DoSomething() }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
				"github.com/mmihic/go-tools/pkg/first/something:github.com/mmihic/go-tools/pkg/newpkg",
			},
			want: strings.TrimLeft(`
// +build tools
package imports

import (
	"github.com/mmihic/go-tools/pkg/newpkg"
	"github.com/mmihic/go-tools/pkg/other"
	"github.com/mmihic/go-tools/pkg/other/elise"
)

func DoSomething() string      { return newpkg.Do() }
func DoSomethingFirst() string { return other.DoSomething() }
func DoSomethingElise() string { return elise.DoSomething() }
`, "\n"),
		},

		// ----------------
		{
			name:
			"rewrite imported package when importing package has same name as imported package",
			pkgPath: "github.com/mmihic/go-tools/tools/first",
			src: `
// +build tools
package first

import (
   "github.com/mmihic/go-tools/pkg/first"
)

func DoSomething() string { return first.DoSomething() }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
			},
			want: strings.TrimLeft(`
// +build tools
package first

import (
	"github.com/mmihic/go-tools/pkg/other"
)

func DoSomething() string { return other.DoSomething() }
`, "\n"),
		},

		// ----------------
		{
			name:
			"rewrite imported package with conflicts",
			pkgPath: "github.com/mmihic/go-tools/tools/main",
			src: `
// +build tools
package main

import (
   "github.com/mmihic/go-tools/pkg/first"
   "github.com/other-repo/other"
)

func DoSomething() string { return first.DoSomething() }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
			},
			want: strings.TrimLeft(`
// +build tools
package main

import (
	other2 "github.com/mmihic/go-tools/pkg/other"
	"github.com/other-repo/other"
)

func DoSomething() string { return other2.DoSomething() }
`, "\n"),
		},

		// ----------------
		{
			name:
			"rewrite imported package with multiple conflicts",
			pkgPath: "github.com/mmihic/go-tools/tools/main",
			src: `
// +build tools
package main

import (
	"github.com/mmihic/go-tools/pkg/first"
	"github.com/other-repo/other"
	other2 "github.com/third-repo/other"
)

func DoSomething() string { return first.DoSomething() }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
			},
			want: strings.TrimLeft(`
// +build tools
package main

import (
	other3 "github.com/mmihic/go-tools/pkg/other"
	"github.com/other-repo/other"
	other2 "github.com/third-repo/other"
)

func DoSomething() string { return other3.DoSomething() }
`, "\n"),
		},

		// ----------------
		{
			name:
			"rewrite imported package with cross conflicts",
			pkgPath: "github.com/mmihic/go-tools/tools/main",
			src: `
// +build tools
package main

import (
	"github.com/mmihic/go-tools/pkg/first"
    "github.com/mmihic/go-tools/pkg/second"
)

func DoSomethingFirst() string  { return first.DoSomething() }
func DoSomethingSecond() string { return second.DoSomething() }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
				"github.com/mmihic/go-tools/pkg/second:github.com/mmihic/go-tools/pkg/second/other",
			},
			want: strings.TrimLeft(`
// +build tools
package main

import (
	"github.com/mmihic/go-tools/pkg/other"
	secondother "github.com/mmihic/go-tools/pkg/second/other"
)

func DoSomethingFirst() string  { return other.DoSomething() }
func DoSomethingSecond() string { return secondother.DoSomething() }
`, "\n"),
		},

		// ----------------
		{
			name:
			"rewritten imported package already imported",
			pkgPath: "github.com/mmihic/go-tools/tools/main",
			src: `
// +build tools
package main

import (
	"github.com/mmihic/go-tools/pkg/first"
    "github.com/mmihic/go-tools/pkg/second"
)

func DoSomethingFirst() string  { return first.DoSomething() }
func DoSomethingSecond() string { return second.DoSomething() }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/second",
			},
			want: strings.TrimLeft(`
// +build tools
package main

import (
	"github.com/mmihic/go-tools/pkg/second"
)

func DoSomethingFirst() string  { return second.DoSomething() }
func DoSomethingSecond() string { return second.DoSomething() }
`, "\n"),
		},

		// ----------------
		{
			name:
			"move self into package that is currently being imported; should strip import",
			pkgPath: "github.com/mmihic/go-tools/pkg/first",
			src: `
// +build tools
package first

import (
	"github.com/mmihic/go-tools/pkg/other"
)

type ArrayOfStuff []*other.Foo

type MapOfStuff map[other.Key]*other.Foo

type ChanOfStuff chan<-*other.Foo

type Config struct {
   other.Foo
   more *other.Foo
}

func DoOtherThing(l ...other.Foo) string  { return other.DoSomething() }

func DoSomethingElse() *other.Foo { return other.Wrap(DoOtherThing()) }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
			},
			want: strings.TrimLeft(`
// +build tools
package other

type ArrayOfStuff []*Foo

type MapOfStuff map[Key]*Foo

type ChanOfStuff chan<- *Foo

type Config struct {
	Foo

	more *Foo
}

func DoOtherThing(l ...Foo,) string { return DoSomething() }

func DoSomethingElse() *Foo { return Wrap(DoOtherThing()) }
`, "\n"),
		},
	} {
		t.Run(tt.name, func(_ *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.src, parser.ParseComments)
			if !assert.NoError(t, err) {
				return
			}

			rules, err := ParseRewriteRules(tt.rules)
			if !assert.NoError(t, err) {
				return
			}

			var buf bytes.Buffer
			rewriteFile(fset, path.NewPath(tt.pkgPath), file, rules, func(filename string, content []byte) error {
				_, err := buf.Write(content)
				return err
			})

			results := buf.String()
			assert.Equal(t, tt.want, results)
		})
	}
}
