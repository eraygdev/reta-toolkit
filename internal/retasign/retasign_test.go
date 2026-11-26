package retasign

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestGenerateAndRoundTripSignature(t *testing.T) {
	sig, err := GenerateSignature()
	if err != nil {
		t.Fatalf("GenerateSignature failed: %v", err)
	}
	if !regexp.MustCompile(`^RETA\*rid[0-9]{12}r$`).MatchString(sig) {
		t.Fatalf("unexpected signature format: %q", sig)
	}

	dir := t.TempDir()
	id, err := WriteSignature(dir)
	if err != nil {
		t.Fatalf("WriteSignature failed: %v", err)
	}
	if !regexp.MustCompile(`^rid[0-9]{12}r$`).MatchString(id) {
		t.Fatalf("unexpected ID format: %q", id)
	}

	got := ReadSignatureID(dir)
	if got != id {
		t.Fatalf("ReadSignatureID() = %q, want %q", got, id)
	}
}

func TestReadSignatureEdgeCases(t *testing.T) {
	dir := t.TempDir()
	if got := ReadSignatureID(dir); got != "" {
		t.Fatalf("expected empty ID for missing .retasign, got %q", got)
	}

	signPath := filepath.Join(dir, SignFilename)
	if err := os.Mkdir(signPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ReadSignatureID(dir); got != "" {
		t.Fatalf("expected empty ID when .retasign is a directory, got %q", got)
	}
	_ = os.Remove(signPath)

	if err := os.WriteFile(signPath, []byte("\n\n  RETA*rid987654321012r  \r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReadSignatureID(dir); got != "rid987654321012r" {
		t.Fatalf("ReadSignatureID() with whitespace/CRLF = %q", got)
	}

	if err := os.WriteFile(signPath, []byte("INVALID\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReadSignatureID(dir); got != "" {
		t.Fatalf("expected empty ID for invalid content, got %q", got)
	}
}
