package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nnutter/httpYat/internal/client"
	"github.com/nnutter/httpYat/internal/httpfile"
	"github.com/nnutter/httpYat/internal/tui"
)

var version = "dev"

type options struct {
	bin     string
	timeout time.Duration
}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr, tui.Run))
}

func run(args []string, stdout, stderr io.Writer, start func(tui.Model) error) int {
	fs := flag.NewFlagSet("httpyat", flag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "httpyac request timeout")
	bin := fs.String("httpyac", "httpyac", "httpyac CLI binary")
	showVersion := fs.Bool("version", false, "print version")
	fs.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "usage: httpyat [flags] <file.http|dir>\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	path := fs.Arg(0)
	opts := options{bin: *bin, timeout: *timeout}
	return launch(path, opts, stderr, start)
}

func launch(path string, opts options, stderr io.Writer, start func(tui.Model) error) int {
	docs, err := httpfile.LoadPath(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "httpyat: %v\n", err)
		return 1
	}
	model := tui.NewWorkspace(docs, client.New(client.Options{Bin: opts.bin, Timeout: opts.timeout}), opts.timeout)
	if err := start(model); err != nil {
		_, _ = fmt.Fprintf(stderr, "httpyat: %v\n", err)
		return 1
	}
	return 0
}
