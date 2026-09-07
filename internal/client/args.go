package client

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
)

func (in Input) args(timeout time.Duration) []string {
	args := []string{"send", "--json", "--raw", "--output", "response", "--no-color"}
	if timeout > 0 {
		args = append(args, "--timeout", strconv.FormatInt(timeout.Milliseconds(), 10))
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		args = append(args, "-n", name)
	} else if in.Line > 0 {
		// httpyac region lines are 0-based. Pass the value as a string so
		// --line 0 still matches (the CLI treats a numeric 0 as missing).
		args = append(args, "-l", strconv.Itoa(in.Line-1))
	}
	// The file precedes --var: httpyac declares --var as variadic, so it
	// swallows any argument after it, including the file itself.
	args = append(args, in.File)
	for _, name := range slices.Sorted(maps.Keys(in.Vars)) {
		if name == "" {
			continue
		}
		args = append(args, "--var", name+"="+in.Vars[name])
	}
	return args
}
