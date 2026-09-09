package main

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

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
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr, tui.Run))
}

func newRootCmd(stdout, stderr io.Writer, start func(tui.Model) error) *cobra.Command {
	var opts options
	cmd := &cobra.Command{
		Use:   "httpyat [file.http|dir]",
		Short: "Browse and send httpYac requests",
		Long:  "httpYat is a TUI over httpYac .http files. The TUI lists requests; httpyac send executes them.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return launch(path, opts, start)
		},
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 30*time.Second, "httpyac request timeout")
	cmd.Flags().StringVar(&opts.bin, "httpyac", "httpyac", "httpyac CLI binary")
	return cmd
}

func execute(args []string, stdout, stderr io.Writer, start func(tui.Model) error) int {
	cmd := newRootCmd(stdout, stderr, start)
	cmd.SetArgs(args)
	if err := fang.Execute(context.Background(), cmd, fang.WithVersion(version)); err != nil {
		return 1
	}
	return 0
}

func launch(path string, opts options, start func(tui.Model) error) error {
	docs, err := httpfile.LoadPath(path)
	if err != nil {
		return err
	}
	model := tui.NewWorkspace(docs, client.New(client.Options{Bin: opts.bin, Timeout: opts.timeout}), opts.timeout)
	if err := start(model); err != nil {
		return err
	}
	return nil
}
