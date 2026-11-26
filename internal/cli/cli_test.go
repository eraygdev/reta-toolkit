package cli

import (
	"errors"
	"reflect"
	"testing"
)

func TestExitErrorFormattingAndUnwrap(t *testing.T) {
	err := Exit(1, "custom %s", "message")
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.Code != 1 || exitErr.Error() != "custom message" {
		t.Fatalf("unexpected ExitError: code=%d msg=%q", exitErr.Code, exitErr.Error())
	}

	wrapped := Wrap(2, errors.New("underlying"), "wrapper error")
	if !errors.Is(wrapped, errors.New("underlying")) && errors.Unwrap(wrapped).Error() != "underlying" {
		t.Fatal("expected Unwrap to expose underlying error")
	}
}

func TestParserFlagsAndPositional(t *testing.T) {
	p := NewParser()

	var force bool
	var dryRun bool
	var output string
	var includes []string

	p.Bool(&force, "force", "f")
	p.Bool(&dryRun, "dry-run", "n")
	p.String(&output, "output", "o")
	p.StringSlice(&includes, "include", "")

	args := []string{"-f", "--output=dist/out.md", "--include", "src/*.go", "--include", "pkg/*.go", "-n", "pos1", "pos2"}
	pos, err := p.Parse(args)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !force || !dryRun {
		t.Fatalf("expected boolean flags true: force=%v dryRun=%v", force, dryRun)
	}
	if output != "dist/out.md" {
		t.Fatalf("output = %q, want dist/out.md", output)
	}
	if !reflect.DeepEqual(includes, []string{"src/*.go", "pkg/*.go"}) {
		t.Fatalf("includes = %v", includes)
	}
	if !reflect.DeepEqual(pos, []string{"pos1", "pos2"}) {
		t.Fatalf("positional = %v", pos)
	}
}

func TestParserUnknownFlagAndMissingArg(t *testing.T) {
	p := NewParser()
	var val string
	p.String(&val, "name", "n")

	_, err := p.Parse([]string{"--unknown"})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Fatalf("expected exit code 2 for unknown flag, got %v", err)
	}

	_, err = p.Parse([]string{"--name"})
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Fatalf("expected exit code 2 for missing argument, got %v", err)
	}
}
