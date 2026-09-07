package client

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunnerSendUsesCLI(t *testing.T) {
	var gotBin string
	var gotArgs []string
	r := New(Options{Bin: "httpyac-bin", Timeout: time.Second})
	r.exec = func(_ context.Context, bin string, args []string) ([]byte, []byte, error) {
		gotBin = bin
		gotArgs = append([]string(nil), args...)
		return []byte(`{"requests":[{"duration":5,"response":{"statusCode":200,"statusMessage":"OK","body":"{\"ok\":true}"}}]}`), nil, nil
	}

	res, err := r.Send(t.Context(), Input{
		File: "/tmp/api.http",
		Line: 4,
		Vars: map[string]string{"id": "7"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBin != "httpyac-bin" {
		t.Errorf("bin = %q", gotBin)
	}
	if !containsPair(gotArgs, "-l", "3") {
		t.Errorf("args = %#v", gotArgs)
	}
	if !containsPair(gotArgs, "--var", "id=7") {
		t.Errorf("vars missing: %#v", gotArgs)
	}
	if res.StatusCode != 200 {
		t.Errorf("status = %d", res.StatusCode)
	}
}

func TestRunnerSendPrefersJSONOnNonZeroExit(t *testing.T) {
	r := New(Options{})
	r.exec = func(context.Context, string, []string) ([]byte, []byte, error) {
		return []byte(`{"requests":[{"response":{"statusCode":418,"statusMessage":"I'm a teapot"}}]}`), []byte("failed tests"), errors.New("exit 1")
	}
	res, err := r.Send(t.Context(), Input{File: "a.http", Name: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 418 {
		t.Errorf("status = %d", res.StatusCode)
	}
}

func TestRunnerSendMissingBinary(t *testing.T) {
	r := New(Options{Bin: "httpyac-missing"})
	r.exec = func(context.Context, string, []string) ([]byte, []byte, error) {
		return nil, nil, &exec.Error{Name: "httpyac-missing", Err: exec.ErrNotFound}
	}
	_, err := r.Send(t.Context(), Input{File: "a.http", Line: 1})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunnerSendRequiresFile(t *testing.T) {
	_, err := New(Options{}).Send(t.Context(), Input{Line: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWrapRunErrorUsesStderr(t *testing.T) {
	err := wrapRunError("httpyac", errors.New("exit 2"), []byte(" cannot find file \n"))
	if !strings.Contains(err.Error(), "cannot find file") {
		t.Fatalf("err = %v", err)
	}
}

func containsPair(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func TestDefaultBin(t *testing.T) {
	r := New(Options{})
	if r.bin != "httpyac" {
		t.Fatalf("bin = %q", r.bin)
	}
	if r.Timeout() != 30*time.Second {
		t.Fatalf("timeout = %s", r.Timeout())
	}
}
