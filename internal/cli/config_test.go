package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smcronin/epo-cli/internal/config"
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

const (
	fakeClientID     = "FakeConsumerKey0123456789"
	fakeClientSecret = "qZ7xWv9KpL2mNb4RtY6s"
)

// useTempConfig points the CLI at a throwaway config file and clears every
// credential source, so these tests never read or write the real config.
func useTempConfig(t *testing.T) string {
	t.Helper()
	resetCredentialFlags(t)
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv(config.ConfigFileEnvVar, path)
	for _, key := range []string{
		"EPO_CLIENT_ID", "EPO_CONSUMER_KEY", "CONSUMER_KEY",
		"EPO_CLIENT_SECRET", "EPO_CONSUMER_SECRET", "CONSUMER_SECRET_KEY",
	} {
		t.Setenv(key, "")
	}
	return path
}

// assertSecretNotLeaked fails when out contains any 3-character run of the
// fake secret, and checks the fixed mask sits under the expected key.
func assertSecretNotLeaked(t *testing.T, out, maskKey string) {
	t.Helper()
	for i := 0; i+3 <= len(fakeClientSecret); i++ {
		if fragment := fakeClientSecret[i : i+3]; strings.Contains(out, fragment) {
			t.Fatalf("output leaks %q from the client secret: %s", fragment, out)
		}
	}

	var env struct {
		Results map[string]any `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v: %s", err, out)
	}
	if got := env.Results[maskKey]; got != config.SecretMask {
		t.Fatalf("%s = %v, want the fixed mask %q", maskKey, got, config.SecretMask)
	}
	if got := env.Results["clientIDMasked"]; got != config.Mask(fakeClientID) {
		t.Fatalf("clientIDMasked = %v, want the 3+3 mask %q", got, config.Mask(fakeClientID))
	}
}

func TestConfigShowDoesNotRevealSecret(t *testing.T) {
	useTempConfig(t)
	if err := config.Save(config.Config{ClientID: fakeClientID, ClientSecret: fakeClientSecret}); err != nil {
		t.Fatalf("save temp config: %v", err)
	}

	out, err := runForTest(t, newConfigCmd(), "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	assertSecretNotLeaked(t, out, "secretMasked")

	var env struct {
		Results struct {
			SecretConfigured bool `json:"secretConfigured"`
			Configured       bool `json:"configured"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if !env.Results.SecretConfigured || !env.Results.Configured {
		t.Fatalf("expected secretConfigured and configured to be true: %s", out)
	}
}

func TestConfigShowWithoutSecretHasEmptyMask(t *testing.T) {
	useTempConfig(t)

	out, err := runForTest(t, newConfigCmd(), "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	var env struct {
		Results struct {
			SecretConfigured bool   `json:"secretConfigured"`
			SecretMasked     string `json:"secretMasked"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v: %s", err, out)
	}
	if env.Results.SecretConfigured || env.Results.SecretMasked != "" {
		t.Fatalf("an unset secret must not show a mask: %s", out)
	}
}

func TestConfigSetCredsDoesNotRevealSecret(t *testing.T) {
	path := useTempConfig(t)

	out, err := runForTest(t, newConfigCmd(), "set-creds", fakeClientID, fakeClientSecret)
	if err != nil {
		t.Fatalf("config set-creds: %v", err)
	}
	assertSecretNotLeaked(t, out, "clientSecretMask")

	saved, err := config.Load()
	if err != nil {
		t.Fatalf("load temp config: %v", err)
	}
	if saved.ClientSecret != fakeClientSecret {
		t.Fatalf("set-creds must still store the real secret in %s", path)
	}
}

func TestAuthConfigureDoesNotRevealSecret(t *testing.T) {
	useTempConfig(t)
	setGlobalFlag(t, &flagClientID, fakeClientID)
	setGlobalFlag(t, &flagClientSecret, fakeClientSecret)

	out, err := runForTest(t, newAuthCmd(), "configure")
	if err != nil {
		t.Fatalf("auth configure: %v", err)
	}
	assertSecretNotLeaked(t, out, "clientSecretMask")
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
