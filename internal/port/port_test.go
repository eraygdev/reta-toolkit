package port

import (
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func TestPortClassAndValidation(t *testing.T) {
	testutil.SilenceStdio(t)
	cases := []struct {
		port string
		pid  string
		want string
	}{
		{"80", "123", "system"},
		{"443", "456", "system"},
		{"5432", "789", "system"},
		{"6379", "999", "system"},
		{"8080", "docker", "docker"},
		{"8000", "1234", "user"},
		{"3000", "-", "user"},
	}
	for _, tc := range cases {
		if got := portClass(tc.port, tc.pid); got != tc.want {
			t.Fatalf("portClass(%q, %q) = %q, want %q", tc.port, tc.pid, got, tc.want)
		}
	}

	if !validatePort("1") || !validatePort("8080") || !validatePort("65535") {
		t.Fatal("expected valid ports to pass")
	}
	if validatePort("0") || validatePort("65536") || validatePort("-1") || validatePort("abc") {
		t.Fatal("expected invalid ports to fail")
	}
}

func TestServiceNameForWellKnownPorts(t *testing.T) {
	if got := serviceNameFor("5432"); got != "postgres" {
		t.Fatalf("serviceNameFor(5432) = %q, want postgres", got)
	}
	if got := serviceNameFor("6379"); got != "redis" {
		t.Fatalf("serviceNameFor(6379) = %q, want redis", got)
	}
}

func TestParseDockerPortMappingsIgnoresUDP(t *testing.T) {
	raw := "web|0.0.0.0:8080->80/tcp, [::]:8443->443/tcp\nmdns|0.0.0.0:5353->5353/udp\n"
	got := parseDockerPortMappings(raw)
	if got["8080"] != "web" || got["8443"] != "web" {
		t.Fatalf("unexpected TCP docker mappings: %v", got)
	}
	if _, exists := got["5353"]; exists {
		t.Fatalf("UDP mapping must not mask TCP port 5353: %v", got)
	}
}

func TestRunHelpAndUnknownSubcommand(t *testing.T) {
	testutil.SilenceStdio(t)
	if code := Run([]string{"--help"}); code != 0 {
		t.Fatalf("Run(--help) = %d, want 0", code)
	}
	if code := Run([]string{"invalid-subcmd"}); code != 1 {
		t.Fatalf("Run(invalid-subcmd) = %d, want 1", code)
	}
}
