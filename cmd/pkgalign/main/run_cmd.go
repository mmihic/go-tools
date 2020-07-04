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

	"github.com/mmihic/go-tools/pkg/astio"
	"github.com/mmihic/go-tools/pkg/path"
	"github.com/mmihic/go-tools/pkg/pkgs"
)

type runCmd struct {
	File         string `short:"f" required:"" help:"name of the configuration file"`
	LocalPkgRoot string `short:"r" required:"" help:"the local package root"`
	Dir          string `arg:"" required:"" help:"the directory to start from"`
	MaxParallel  int    `arg:"" default:"10" help:"max parallelism"`
}

// Run runs the rewrite tool
func (cmd *runCmd) Run() error {
	contents, err := ioutil.ReadFile(cmd.File)
	if err != nil {
		return err
	}

	type config struct {
		PkgMoves pkgs.Moves `yaml:"packages"`
	}

	var cfg config
	if err := yaml.Unmarshal(contents, &cfg); err != nil {
		return fmt.Errorf("unable to parse config: %v", err)
	}

	rules := cfg.PkgMoves.ApplyPrefix(path.NewPath(cmd.LocalPkgRoot))

	var (
		wg      sync.WaitGroup
		errorCh = make(chan error, 1000)
		dirsCh  = make(chan string, 1000)
		allDone = make(chan struct{})
		allErr  error
	)

	// Process directories in parallel
	for i := 0; i < cmd.MaxParallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for dir := range dirsCh {
				if err := cmd.processDir(dir, rules); err != nil {
					errorCh <- err
				}
			}
		}()
	}

	// Combine errors
	go func() {
		for err := range errorCh {
			allErr = multierr.Append(allErr, err)
		}
		close(allDone)
	}()

	// Feed in all the directories
	if err := filepath.Walk(cmd.Dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			dirsCh <- path
		}
		return nil
	}); err != nil {
		return err
	}
	close(dirsCh)

	// Wait for everything to complete
	wg.Wait()
	close(errorCh)
	<-allDone
	return allErr
}

func (cmd *runCmd) processDir(dir string, moves pkgs.Moves) error {
	fset := token.NewFileSet()
	pkgPath := path.NewPath(filepath.Join(cmd.LocalPkgRoot, dir))
	packages, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("could not parse %s: %v", dir, err)
	}

	for _, pkg := range packages {
		for _, file := range pkg.Files {
			fname := fset.File(file.Pos())
			fmt.Printf("processing %s\n", fname.Name())
			changed, err := moves.Apply(fset, pkgPath, file)
			if err != nil {
				return fmt.Errorf("error applying moves to %s: %v", fname.Name(), err)
			}

			if changed {
				if err := astio.WriteFile(fset, file); err != nil {
					return fmt.Errorf("error applying moves to %s: %v", fname.Name(), err)
				}
			}
		}
	}

	return nil
}
