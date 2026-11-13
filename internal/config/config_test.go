package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestMaskSecretRevealsNothing(t *testing.T) {
	if got := MaskSecret(""); got != "" {
		t.Fatalf("MaskSecret(\"\") = %q, want empty", got)
	}
	if got := MaskSecret("   "); got != "" {
		t.Fatalf("MaskSecret of blanks = %q, want empty", got)
	}

	secrets := []string{
		"qZ7xW",
		"qZ7xWv9KpL2mNb4R",
		"qZ7xWv9KpL2mNb4RtY6sHd8FgJ1cUe3AoI5nMb0V",
	}
	for _, secret := range secrets {
		got := MaskSecret(secret)
		if got != SecretMask {
			t.Fatalf("MaskSecret(%d chars) = %q, want the fixed mask %q", len(secret), got, SecretMask)
		}
		for i := 0; i+3 <= len(secret); i++ {
			if strings.Contains(got, secret[i:i+3]) {
				t.Fatalf("mask %q leaks %q from the secret", got, secret[i:i+3])
			}
		}
	}
}

func TestMaskKeepsIdentifierEnds(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"abc":          "***",
		"abcdef":       "******",
		"abcdefghij":   "abc****hij",
		"  abcdefg   ": "abc*efg",
	}
	for input, want := range cases {
		if got := Mask(input); got != want {
			t.Fatalf("Mask(%q) = %q, want %q", input, got, want)
		}
	}
}
