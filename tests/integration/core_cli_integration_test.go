package integration

import (
	"encoding/json"
	"testing"
)

func TestCoreCommandsIntegration(t *testing.T) {
	t.Parallel()
	requireIntegrationCredentials(t)

	cases := []struct {
		name        string
		args        []string
		wantCommand string
		wantService string
	}{
		{
			name:        "auth_check",
			args:        []string{"auth", "check", "-f", "json", "-q"},
			wantCommand: "epo auth check",
			wantService: "",
		},
		{
			name:        "auth_token",
			args:        []string{"auth", "token", "-f", "json", "-q"},
			wantCommand: "epo auth token",
			wantService: "",
		},
		{
			name:        "family_get",
			args:        []string{"family", "get", "EP.1000000.A1", "--ref-type", "publication", "--input-format", "docdb", "-f", "json", "-q"},
			wantCommand: "epo family get",
			wantService: "family",
		},
		{
			name:        "number_convert",
			args:        []string{"number", "convert", "EP.1000000.A1", "--ref-type", "publication", "--from-format", "docdb", "--to-format", "epodoc", "-f", "json", "-q"},
			wantCommand: "epo number convert",
			wantService: "number-service",
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
			if env.Service != tc.wantService {
				t.Fatalf("unexpected service: got %q want %q", env.Service, tc.wantService)
			}
		})
	}
}
