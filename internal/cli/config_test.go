package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigSetCredsInput(t *testing.T) {
	t.Run("args source", func(t *testing.T) {
		resetCredentialFlags(t)
		id, secret, source, err := resolveConfigSetCredsInput([]string{"id1", "sec1"}, false, "")
		if err != nil {
			t.Fatalf("resolveConfigSetCredsInput(args) error: %v", err)
		}
		if id != "id1" || secret != "sec1" || source != "args" {
			t.Fatalf("unexpected args result: id=%q secret=%q source=%q", id, secret, source)
		}
	})

	t.Run("flags source", func(t *testing.T) {
		resetCredentialFlags(t)
		flagClientID = "flag-id"
		flagClientSecret = "flag-secret"

		id, secret, source, err := resolveConfigSetCredsInput(nil, false, "")
		if err != nil {
			t.Fatalf("resolveConfigSetCredsInput(flags) error: %v", err)
		}
		if id != "flag-id" || secret != "flag-secret" || source != "flags" {
			t.Fatalf("unexpected flags result: id=%q secret=%q source=%q", id, secret, source)
		}
	})

	t.Run("from-env source", func(t *testing.T) {
		resetCredentialFlags(t)
		t.Setenv("EPO_CLIENT_ID", "env-id")
		t.Setenv("EPO_CLIENT_SECRET", "env-secret")

		id, secret, source, err := resolveConfigSetCredsInput(nil, true, "")
		if err != nil {
			t.Fatalf("resolveConfigSetCredsInput(from-env) error: %v", err)
		}
		if id != "env-id" || secret != "env-secret" {
			t.Fatalf("unexpected env result: id=%q secret=%q", id, secret)
		}
		if source == "" {
			t.Fatal("expected non-empty source for env")
		}
	})

	t.Run("from-dotenv source", func(t *testing.T) {
		resetCredentialFlags(t)
		dir := t.TempDir()
		dotenvPath := filepath.Join(dir, ".env")
		content := "EPO_CLIENT_ID=dot-id\nEPO_CLIENT_SECRET=dot-secret\n"
		if err := os.WriteFile(dotenvPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write dotenv: %v", err)
		}

		id, secret, source, err := resolveConfigSetCredsInput(nil, false, dotenvPath)
		if err != nil {
			t.Fatalf("resolveConfigSetCredsInput(from-dotenv) error: %v", err)
		}
		if id != "dot-id" || secret != "dot-secret" || source != "dotenv" {
			t.Fatalf("unexpected dotenv result: id=%q secret=%q source=%q", id, secret, source)
		}
	})

	t.Run("reject mixed sources", func(t *testing.T) {
		resetCredentialFlags(t)
		_, _, _, err := resolveConfigSetCredsInput([]string{"id", "secret"}, true, "")
		if err == nil {
			t.Fatal("expected error for mixed sources")
		}
	})
}

func resetCredentialFlags(t *testing.T) {
	t.Helper()
	prevID := flagClientID
	prevSecret := flagClientSecret
	flagClientID = ""
	flagClientSecret = ""
	t.Cleanup(func() {
		flagClientID = prevID
		flagClientSecret = prevSecret
	})
}
