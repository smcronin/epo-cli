package integration

import (
	"encoding/json"
	"testing"
)

func TestExtendedCommandsIntegration(t *testing.T) {
	t.Parallel()
	requireIntegrationCredentials(t)

	cases := []struct {
		name        string
		args        []string
		wantCommand string
		wantService string
	}{
		{
			name:        "register_get",
			args:        []string{"register", "get", "EP99203729", "-f", "json", "-q"},
			wantCommand: "epo register get",
			wantService: "register",
		},
		{
			name:        "register_events",
			args:        []string{"register", "events", "EP99203729", "-f", "json", "-q"},
			wantCommand: "epo register events",
			wantService: "register",
		},
		{
			name:        "register_search",
			args:        []string{"register", "search", "--q", "pa=IBM", "--range", "1-2", "-f", "json", "-q"},
			wantCommand: "epo register search",
			wantService: "register",
		},
		{
			name:        "legal_get",
			args:        []string{"legal", "get", "EP.1000000.A1", "--ref-type", "publication", "--input-format", "docdb", "-f", "json", "-q"},
			wantCommand: "epo legal get",
			wantService: "legal",
		},
		{
			name:        "cpc_get",
			args:        []string{"cpc", "get", "H04W", "--depth", "1", "-f", "json", "-q"},
			wantCommand: "epo cpc get",
			wantService: "classification/cpc",
		},
		{
			name:        "cpc_search",
			args:        []string{"cpc", "search", "--q", "chemistry", "--range", "1-3", "-f", "json", "-q"},
			wantCommand: "epo cpc search",
			wantService: "classification/cpc",
		},
		{
			name:        "usage_stats",
			args:        []string{"usage", "stats", "--date", "01/03/2024", "-f", "json", "-q"},
			wantCommand: "epo usage stats",
			wantService: "usage",
		},
		{
			name:        "raw_get",
			args:        []string{"raw", "get", "/published-data/search", "--query", "q=applicant=IBM", "--query", "Range=1-1", "-f", "json", "-q"},
			wantCommand: "epo raw get",
			wantService: "raw",
		},
		{
			name:        "raw_post",
			args:        []string{"raw", "post", "/published-data/search", "--content-type", "text/plain", "--body", "q=applicant%3DIBM", "-f", "json", "-q"},
			wantCommand: "epo raw post",
			wantService: "raw",
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
