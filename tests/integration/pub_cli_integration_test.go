package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

type cliEnvelope struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Service string `json:"service"`
	Error   *struct {
		Code    int    `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func TestPubCommandsIntegration(t *testing.T) {
	t.Parallel()
	requireIntegrationCredentials(t)

	cases := []struct {
		name        string
		args        []string
		wantCommand string
	}{
		{
			name:        "biblio",
			args:        []string{"pub", "biblio", "EP1000000.A1", "-f", "json", "-q"},
			wantCommand: "epo pub biblio",
		},
		{
			name:        "abstract",
			args:        []string{"pub", "abstract", "EP1000000.A1", "-f", "json", "-q"},
			wantCommand: "epo pub abstract",
		},
		{
			name:        "claims",
			args:        []string{"pub", "claims", "EP1000000.A1", "-f", "json", "-q"},
			wantCommand: "epo pub claims",
		},
		{
			name:        "description",
			args:        []string{"pub", "description", "EP1000000.A1", "-f", "json", "-q"},
			wantCommand: "epo pub description",
		},
		{
			name:        "equivalents",
			args:        []string{"pub", "equivalents", "EP1000000.A1", "--constituents", "biblio", "-f", "json", "-q"},
			wantCommand: "epo pub equivalents",
		},
		{
			name:        "search",
			args:        []string{"pub", "search", "--query", "applicant=IBM", "--range", "1-2", "-f", "json", "-q"},
			wantCommand: "epo pub search",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out := runEPO(t, tc.args...)
			var env cliEnvelope
			if err := json.Unmarshal(out, &env); err != nil {
				t.Fatalf("parse JSON output: %v\noutput: %s", err, string(out))
			}

			if !env.OK {
				t.Fatalf("command failed: %+v\noutput: %s", env.Error, string(out))
			}
			if env.Command != tc.wantCommand {
				t.Fatalf("unexpected command path: got %q want %q", env.Command, tc.wantCommand)
			}
			if env.Service != "published-data" {
				t.Fatalf("unexpected service: got %q", env.Service)
			}
		})
	}
}

func requireIntegrationCredentials(t *testing.T) {
	t.Helper()
	if os.Getenv("EPO_INTEGRATION") != "1" {
		t.Skip("set EPO_INTEGRATION=1 to run live OPS integration tests")
	}

	if firstSetEnv([]string{"EPO_CLIENT_ID", "EPO_CONSUMER_KEY", "CONSUMER_KEY"}) == "" ||
		firstSetEnv([]string{"EPO_CLIENT_SECRET", "EPO_CONSUMER_SECRET", "CONSUMER_SECRET_KEY"}) == "" {
		t.Skip("missing OPS credentials in environment")
	}
}

func firstSetEnv(keys []string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

func runEPO(t *testing.T, args ...string) []byte {
	t.Helper()

	command := exec.Command("go", append([]string{"run", "./cmd/epo"}, args...)...)
	command.Dir = repoRoot(t)
	command.Env = os.Environ()
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, string(out))
	}
	return out
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
