package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/multierr"
	"gopkg.in/yaml.v2"

	"github.com/mmihic/go-tools/tools/pkgalign"
)

type runCmd struct {
	File         string `short:"f" required:"" help:"name of the configuration file"`
	LocalPkgRoot string `short:"r" required:"" help:"the local package root"`
	Dir          string `arg:"" required:"" help:"the directory to start from"`
}

// Run runs the rewrite tool
func (cmd *runCmd) Run() error {
	contents, err := ioutil.ReadFile(cmd.File)
	if err != nil {
		return err
	}

	fmt.Println(string(contents))

	type config struct {
		Rules []*pkgalign.RewriteRule `yaml:"packages"`
	}

	var cfg config
	if err := yaml.Unmarshal(contents, &cfg); err != nil {
		return fmt.Errorf("unable to parse config: %v", err)
	}

	for _, rule := range cfg.Rules {
		rule.To, rule.From = filepath.Join(cmd.LocalPkgRoot, rule.To), filepath.Join(cmd.LocalPkgRoot, rule.From)
	}

	var dirs []string
	if err := filepath.Walk(cmd.Dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	}); err != nil {
		return err
	}

	var (
		wg      sync.WaitGroup
		errorCh = make(chan error, len(dirs))
	)
	for _, dir := range dirs {
		dir := dir
		wg.Add(1)
		go func() {
			defer wg.Done()

			fset := token.NewFileSet()
			pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
			if err != nil {
				errorCh <- fmt.Errorf("could not parse %s: %v", dir, err)
				return
			}

			for _, pkg := range pkgs {
				pkgPath := filepath.Join(cmd.LocalPkgRoot, dir)
				pkgalign.Rewrite(fset, pkgPath, pkg, cfg.Rules)
			}
		}()
	}

	wg.Wait()
	close(errorCh)
	var allErr error
	for err := range errorCh {
		allErr = multierr.Append(allErr, err)
	}
	return allErr
}
