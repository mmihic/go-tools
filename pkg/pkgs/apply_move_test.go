package pkgs

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mmihic/go-tools/pkg/astio"
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
		{
			name:    "handles variables with new package name",
			pkgPath: "github.com/mmihic/go-tools/pkg/imports",
			src: `
// +build tools
package imports

import (
   "github.com/mmihic/go-tools/pkg/first"
)

func DoSomething() string {
	var first Conflict
	return first.DoSomething() 
}

func DoSomethingElse() string {
	var other Conflict
	return other.DoSomething() 
}

`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
			},
			want: strings.TrimLeft(`
// +build tools
package imports

import (
	other2 "github.com/mmihic/go-tools/pkg/other"
)

func DoSomething() string {
	var first Conflict
	return first.DoSomething()
}

func DoSomethingElse() string {
	var other Conflict
	return other.DoSomething()
}

`, "\n"),
		},


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
// Package first is a package that does some things. */

package first

func DoSomething() string { return "does something" }
`,
			rules: []string{
				"github.com/mmihic/go-tools/pkg/first:github.com/mmihic/go-tools/pkg/other",
			},
			want: strings.TrimLeft(`
// +build tools
// Package other is a package that does some things. */

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
			name:    "move self into package that is currently being imported; should strip import",
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

func Prepare() {
	var cfg other.Config
	myVal := other.MyConstant
}
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

func Prepare() {
	var cfg Config
	myVal := other.MyConstant
}

`, "\n"),
		},
		{
			name:    "removes self import",
			pkgPath: "github.com/foo/src/authgate/server",
			src: `
// Package server provides the server mainline for the authgate service.
package server

import (
	"net/http"

	"github.com/foo/src/authgate/authn"
	"github.com/foo/src/authgate/config"
	"github.com/foo/src/authgate/server/httpauth"
)

type RunOptions struct {
   ConfigFile string
}

func Run() error {
	var cfg config.Configuration
	if err := LoadConfigFile(runOpts.ConfigFile, &cfg); err != nil {
		Fatalf("could not load configuration file: %v", err)
	}
}
`,
			rules: []string{
				"github.com/foo/src/authgate/server/httpauth:github.com/foo/src/services/authgate",
				"github.com/foo/src/authgate/authn:github.com/foo/src/services/authgate/pkg/authn",
				"github.com/foo/src/authgate/server:github.com/foo/src/servers/authgate",
				"github.com/foo/src/authgate/config:github.com/foo/src/servers/authgate",
			},
			want: `
// Package authgate provides the server mainline for the authgate service.
package authgate

import (
	"net/http"

	servicesauthgate "github.com/foo/src/services/authgate"
	"github.com/foo/src/services/authgate/pkg/authn"
)

type RunOptions struct {
	ConfigFile string
}

func Run() error {
	var cfg Configuration
	if err := LoadConfigFile(runOpts.ConfigFile, &cfg); err != nil {
		Fatalf("could not load configuration file: %v", err)
	}
}

`,
		},
		{
			name:    "removes self import",
			pkgPath: "github.com/foo/src/authgate/server",
			src: `
// Package server provides the server mainline for the authgate service.
package server

import (
	"net/http"

	"github.com/foo/src/authgate/authn"
	"github.com/foo/src/authgate/config"
	"github.com/foo/src/authgate/server/httpauth"
)

type RunOptions struct {
   ConfigFile string
}

func Run() error {
	var config config.Configuration
	if err := LoadConfigFile(runOpts.ConfigFile, &config); err != nil {
		Fatalf("could not load configuration file: %v", err)
	}

	x := config.X
}
`,
			rules: []string{
				"github.com/foo/src/authgate/server/httpauth:github.com/foo/src/services/authgate",
				"github.com/foo/src/authgate/authn:github.com/foo/src/services/authgate/pkg/authn",
				"github.com/foo/src/authgate/server:github.com/foo/src/servers/authgate",
				"github.com/foo/src/authgate/config:github.com/foo/src/servers/authgate",
			},
			want: `
// Package authgate provides the server mainline for the authgate service.
package authgate

import (
	"net/http"

	servicesauthgate "github.com/foo/src/services/authgate"
	"github.com/foo/src/services/authgate/pkg/authn"
)

type RunOptions struct {
	ConfigFile string
}

func Run() error {
	var config Configuration
	if err := LoadConfigFile(runOpts.ConfigFile, &config); err != nil {
		Fatalf("could not load configuration file: %v", err)
	}

	x := config.X
}
`,
		},
		{
			name:    "handles _ import",
			pkgPath: "github.com/foo/src/authgate/server",
			src: `
// Package server provides the server mainline for the authgate service.
package server

import (
	"net/http"

	_ "github.com/foo/src/statik"
	_ "github.com/foo/src/authgate/authn"
)

func Run() error {
	return nil
}
`,
			rules: []string{
				"github.com/foo/src/authgate/server/httpauth:github.com/foo/src/services/authgate",
				"github.com/foo/src/authgate/authn:github.com/foo/src/services/authgate/pkg/authn",
				"github.com/foo/src/authgate/server:github.com/foo/src/servers/authgate",
				"github.com/foo/src/authgate/config:github.com/foo/src/servers/authgate",
			},
			want: `
// Package authgate provides the server mainline for the authgate service.
package authgate

import (
	"net/http"

	_ "github.com/foo/src/services/authgate/pkg/authn"
	_ "github.com/foo/src/statik"
)

func Run() error {
	return nil
}
`,
		},
	} {
		t.Run(tt.name, func(_ *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.src, parser.ParseComments)
			if !assert.NoError(t, err) {
				return
			}

			moves, err := ParseMoves(tt.rules)
			if !assert.NoError(t, err) {
				return
			}

			_, err = moves.Apply(fset, path.NewPath(tt.pkgPath), file)
			if !assert.NoError(t, err) {
				return
			}

			results, err := astio.String(fset, file)
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, strings.TrimSpace(tt.want), strings.TrimSpace(results))
		})
	}
}
