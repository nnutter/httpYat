package client

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const defaultBin = "httpyac"

// Sender runs a request from an .http file.
type Sender interface {
	Send(ctx context.Context, in Input) (Result, error)
}

// SendFunc adapts a function to Sender.
type SendFunc func(ctx context.Context, in Input) (Result, error)

// Send implements Sender.
func (f SendFunc) Send(ctx context.Context, in Input) (Result, error) {
	return f(ctx, in)
}

// Input selects one httpYac request and optional variable overlays.
type Input struct {
	File    string
	Line    int // 1-based source line of the request
	Name    string
	Vars    map[string]string
	Timeout time.Duration
}

// Options configure the httpyac CLI runner.
type Options struct {
	Bin     string
	Timeout time.Duration
}

type execFunc func(ctx context.Context, bin string, args []string) (stdout, stderr []byte, err error)

// Runner invokes the httpyac CLI.
type Runner struct {
	bin     string
	timeout time.Duration
	exec    execFunc
}

// New returns a Sender that runs httpyac.
func New(opts Options) *Runner {
	return &Runner{
		bin:     cmp.Or(opts.Bin, defaultBin),
		timeout: opts.Timeout,
		exec:    runCLI,
	}
}

// Timeout is the default CLI connection timeout.
func (r *Runner) Timeout() time.Duration {
	if r == nil || r.timeout <= 0 {
		return 30 * time.Second
	}
	return r.timeout
}

// Send runs one request with httpyac send --json.
func (r *Runner) Send(ctx context.Context, in Input) (Result, error) {
	if r == nil {
		return Result{}, errors.New("httpyac runner is not configured")
	}
	if in.File == "" {
		return Result{}, errors.New("missing .http file path")
	}
	if in.Name == "" && in.Line <= 0 {
		return Result{}, errors.New("missing request name or line")
	}

	timeout := in.Timeout
	if timeout <= 0 {
		timeout = r.Timeout()
	}
	args := in.args(timeout)

	runCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, timeout+2*time.Second)
	}
	defer cancel()

	stdout, stderr, err := r.exec(runCtx, r.bin, args)
	res, parseErr := parseOutput(stdout)
	if parseErr == nil {
		return res, nil
	}
	if err != nil {
		return Result{}, wrapRunError(r.bin, err, stderr)
	}
	if len(bytes.TrimSpace(stderr)) > 0 {
		return Result{}, fmt.Errorf("httpyac: %s", bytes.TrimSpace(stderr))
	}
	return Result{}, parseErr
}

func runCLI(ctx context.Context, bin string, args []string) (stdout, stderr []byte, err error) {
	if _, err := exec.LookPath(bin); err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err = cmd.Run()
	return out.Bytes(), errb.Bytes(), err
}

func wrapRunError(bin string, err error, stderr []byte) error {
	if ee, ok := errors.AsType[*exec.Error](err); ok && errors.Is(ee.Err, exec.ErrNotFound) {
		return fmt.Errorf("httpyac CLI not found (%s); install httpyac or pass -httpyac", bin)
	}
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("httpyac CLI not found (%s); install httpyac or pass -httpyac", bin)
	}
	msg := bytes.TrimSpace(stderr)
	if len(msg) > 0 {
		return fmt.Errorf("httpyac: %s", msg)
	}
	return fmt.Errorf("httpyac: %w", err)
}
