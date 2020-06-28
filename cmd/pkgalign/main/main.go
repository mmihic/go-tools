package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	"github.com/mmihic/go-tools/tools/pkgalign"
)

const (
	loadAllSyntax = packages.NeedName |
		packages.NeedFiles |
		packages.NeedCompiledGoFiles |
		packages.NeedImports |
		packages.NeedTypes |
		packages.NeedTypesSizes |
		packages.NeedSyntax |
		packages.NeedTypesInfo |
		packages.NeedDeps
)

var (
	pkgName  string
	fromPath string
	toPath   string
)

func main() {
	flag.StringVar(&pkgName, "name", "", "the name of the package")
	flag.StringVar(&fromPath, "from", "", "the original package path")
	flag.StringVar(&toPath, "to", "", "the new package path")
	flag.Parse()

	progname := filepath.Base(os.Args[0])
	args := flag.Args()
	if len(args) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, `failed: %s`, progname)
		os.Exit(1)
	}

	initial, err := packages.Load(&packages.Config{
		Mode:  loadAllSyntax,
		Tests: true,
	}, args...)
	if err == nil {
		if n := packages.PrintErrors(initial); n > 1 {
			err = fmt.Errorf("%d errors during loading", n)
		} else if n == 1 {
			err = fmt.Errorf("error during loading")
		} else if len(initial) == 0 {
			err = fmt.Errorf("%s matched no packages", strings.Join(args, " "))
		}
	}

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}

	var wg sync.WaitGroup
	for _, pkg := range initial {
		pkg := pkg
		wg.Add(1)
		go func() {
			defer wg.Done()
			pkgalign.Rewrite(pkg, []*pkgalign.RewriteRule{
				{
					From:    fromPath,
					To:      toPath,
					PkgName: pkgName,
				},
			})
		}()
	}

	wg.Wait()
}
