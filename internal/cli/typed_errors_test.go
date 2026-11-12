package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func TestSingleInputFailuresKeepTypedError(t *testing.T) {
	cases := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"pub search flat", newPubCmd, []string{"search", "--query", "applicant=IBM", "--flat"}},
		{"pub search summary", newPubCmd, []string{"search", "--query", "applicant=IBM", "--summary"}},
		{"pub recent summary", newPubCmd, []string{"recent", "--cpc", "G06N", "--summary"}},
		{"pub recent", newPubCmd, []string{"recent", "--cpc", "G06N"}},
		{"pub biblio flat", newPubCmd, []string{"biblio", "EP.9999999.A1", "--flat"}},
		{"pub images fetch", newPubCmd, []string{"images", "fetch", "EP/9999999/A1/fullimage"}},
		{"family get flat", newFamilyCmd, []string{"get", "EP.9999999.A1", "--flat"}},
		{"family summary", newFamilyCmd, []string{"summary", "EP.9999999.A1"}},
		{"legal get flat", newLegalCmd, []string{"get", "EP.9999999.A1", "--flat"}},
		{"number normalize", newNumberCmd, []string{"normalize", "EP9999999A1"}},
		{"register get summary", newRegisterCmd, []string{"get", "EP99999999", "--summary"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			installOPSStub(t, opsNotFound)

			out, err := runForTest(t, tc.cmd(), tc.args...)
			if err == nil {
				t.Fatalf("expected an error, got success output: %s", out)
			}
			if strings.TrimSpace(out) != "" {
				t.Fatalf("expected no success output on failure, got: %s", out)
			}
			cliErr := mapError(err)
			if cliErr.Type != "NOT_FOUND" || cliErr.Code != 404 {
				t.Fatalf("expected NOT_FOUND/404, got %s/%d (%s)", cliErr.Type, cliErr.Code, cliErr.Message)
			}
			if code := mapExitCode(cliErr); code != epoerrors.ExitNotFound {
				t.Fatalf("expected exit code %d, got %d", epoerrors.ExitNotFound, code)
			}
		})
	}
}

func TestAllModeFailureKeepsTypedError(t *testing.T) {
	cases := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"pub search all", newPubCmd, []string{"search", "--query", "applicant=IBM"}},
		{"register search all", newRegisterCmd, []string{"search", "--q", "pa=IBM"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			installOPSStub(t, opsNotFound)
			setGlobalFlag(t, &flagAll, true)

			_, err := runForTest(t, tc.cmd(), tc.args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			if cliErr := mapError(err); cliErr.Type != "NOT_FOUND" {
				t.Fatalf("expected NOT_FOUND, got %s (%s)", cliErr.Type, cliErr.Message)
			}
		})
	}
}

func TestMissingCredentialsSurfaceAsAuthFailure(t *testing.T) {
	installOPSStub(t, opsNotFound)
	t.Setenv("EPO_CLIENT_ID", "")
	t.Setenv("EPO_CLIENT_SECRET", "")
	t.Setenv("EPO_CONSUMER_KEY", "")
	t.Setenv("EPO_CONSUMER_SECRET", "")
	t.Setenv("CONSUMER_KEY", "")
	t.Setenv("CONSUMER_SECRET_KEY", "")

	out, err := runForTest(t, newPubCmd(), "search", "--query", "applicant=IBM", "--summary")
	if err == nil {
		t.Fatalf("expected an error, got success output: %s", out)
	}
	cliErr := mapError(err)
	if cliErr.Type != "AUTH_FAILURE" {
		t.Fatalf("expected AUTH_FAILURE, got %s", cliErr.Type)
	}
	if code := mapExitCode(cliErr); code != epoerrors.ExitAuthFailure {
		t.Fatalf("expected exit code %d, got %d", epoerrors.ExitAuthFailure, code)
	}
}

func TestBatchSummaryKeepsOneRowPerQuery(t *testing.T) {
	installOPSStub(t, opsNotFound)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runPublishedSearchSummary(&cobra.Command{Use: "search"}, []string{"applicant=IBM", "applicant=SAP"}, "biblio", "", false, pubSearchSortNone, false)
	})
	if runErr != nil {
		t.Fatalf("batch summary should keep per-item errors, got: %v", runErr)
	}

	var env struct {
		Results []struct {
			Query string `json:"query"`
			OK    bool   `json:"ok"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v: %s", err, out)
	}
	if len(env.Results) != 2 {
		t.Fatalf("expected one row per query, got %d: %s", len(env.Results), out)
	}
	for _, row := range env.Results {
		if row.OK {
			t.Fatalf("failed query %q must not report ok:true: %s", row.Query, out)
		}
	}
}

func TestOPSRequestsReuseOneToken(t *testing.T) {
	stub := installOPSStub(t, func(req *http.Request) (int, http.Header, string) {
		return http.StatusOK, nil, `{}`
	})

	for i := 0; i < 3; i++ {
		if _, err := executeOPSRequest(context.Background(), api.Request{Path: "/published-data/publication/epodoc/EP1000000/biblio"}); err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}
	}

	if got := len(stub.dataRequests()); got != 3 {
		t.Fatalf("expected 3 data requests, got %d", got)
	}
	if stub.tokenCalls != 1 {
		t.Fatalf("expected one token request for three OPS calls, got %d", stub.tokenCalls)
	}
}

func TestRetryAfterBeyondDeadlineIsRateLimited(t *testing.T) {
	stub := installOPSStub(t, func(req *http.Request) (int, http.Header, string) {
		return http.StatusTooManyRequests, http.Header{"Retry-After": {"60000"}}, `<error><code>429</code></error>`
	})
	setGlobalFlag(t, &flagTimeout, 2)

	_, err := runForTest(t, newFamilyCmd(), "get", "EP.1000000.A1")
	if err == nil {
		t.Fatal("expected an error")
	}
	cliErr := mapError(err)
	if cliErr.Type != "RATE_LIMITED" || mapExitCode(cliErr) != epoerrors.ExitRateLimited {
		t.Fatalf("expected RATE_LIMITED/exit 5, got %s/exit %d (%s)", cliErr.Type, mapExitCode(cliErr), cliErr.Message)
	}
	if !strings.Contains(cliErr.Hint, "60000 ms") {
		t.Fatalf("expected the Retry-After value in the hint, got %q", cliErr.Hint)
	}
	if got := len(stub.dataRequests()); got != 1 {
		t.Fatalf("expected a single request, got %d", got)
	}
}
