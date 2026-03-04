package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCredentialsFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := `
# comment
export EPO_CLIENT_ID="abc123"
EPO_CLIENT_SECRET='secret-xyz'
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}

	cfg, err := LoadCredentialsFromDotEnv(path)
	if err != nil {
		t.Fatalf("LoadCredentialsFromDotEnv error: %v", err)
	}
	if cfg.ClientID != "abc123" {
		t.Fatalf("ClientID = %q, want %q", cfg.ClientID, "abc123")
	}
	if cfg.ClientSecret != "secret-xyz" {
		t.Fatalf("ClientSecret = %q, want %q", cfg.ClientSecret, "secret-xyz")
	}
}
