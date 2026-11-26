package main

import (
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func TestCommandsRegistryAndHelp(t *testing.T) {
	testutil.SilenceStdio(t)
	t.Setenv("NO_COLOR", "1")
	t.Setenv("RETA_HINTS", "off")

	seen := make(map[string]bool)
	for _, cmd := range commands {
		if cmd.Name == "" || cmd.Desc == "" || cmd.Group == "" || cmd.Run == nil {
			t.Fatalf("invalid command metadata: %+v", cmd)
		}
		if seen[cmd.Name] {
			t.Fatalf("duplicate command name: %s", cmd.Name)
		}
		seen[cmd.Name] = true

		if findCommand(cmd.Name) == nil {
			t.Fatalf("findCommand(%q) returned nil", cmd.Name)
		}
		if code := cmd.Run([]string{"--help"}); code != 0 {
			t.Fatalf("command %q --help returned %d, want 0", cmd.Name, code)
		}
	}

	if findCommand("does-not-exist") != nil {
		t.Fatal("expected nil for unknown command")
	}
}
