package main

import (
	"os"

	"github.com/alecthomas/kong"
)

var commands = struct {
	Run runCmd `cmd:"" help:"runs the rewrite tool"`
}{}

func main() {
	helpOpt := kong.ConfigureHelp(kong.HelpOptions{
		Tree: true,
	})
	parser, err := kong.New(&commands, helpOpt)
	if err != nil {
		panic(err)
	}

	parser.Model.HelpFlag.Short = 'h'

	kongCtx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)
	parser.FatalIfErrorf(kongCtx.Run())
}
