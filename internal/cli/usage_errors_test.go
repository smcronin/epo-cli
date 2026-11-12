package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
)

func TestMisuseIsAUsageError(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"mistyped subcommand", []string{"pub", "bogus"}},
		{"mistyped subcommand with input", []string{"register", "event", "EP99203729"}},
		{"mistyped nested subcommand", []string{"pub", "images", "fetsh", "EP/1000000/A1/fullimage"}},
		{"unknown top-level command", []string{"bogus"}},
		{"unknown flag", []string{"methods", "--bogus"}},
		{"missing required argument", []string{"status"}},
		{"too many arguments", []string{"cpc", "get", "H04W", "H04L"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := installOPSStub(t, opsNotFound)

			out, err := runForTest(t, rootCmd, tc.args...)
			if err == nil {
				t.Fatalf("expected a usage error, got exit 0 with output: %.200s", out)
			}
			if strings.TrimSpace(out) != "" {
				t.Fatalf("expected nothing on stdout before the error envelope, got: %.200s", out)
			}
			cliErr := mapError(err)
			if cliErr.Type != "VALIDATION_ERROR" {
				t.Fatalf("expected VALIDATION_ERROR, got %s (%s)", cliErr.Type, cliErr.Message)
			}
			if code := mapExitCode(cliErr); code != epoerrors.ExitUsageError {
				t.Fatalf("expected exit code %d, got %d", epoerrors.ExitUsageError, code)
			}
			if got := len(stub.dataRequests()); got != 0 {
				t.Fatalf("expected no OPS request, got %d", got)
			}
		})
	}
}

func TestUsageErrorEnvelopeGoesToStdoutInJSONMode(t *testing.T) {
	_, err := runForTest(t, rootCmd, "pub", "bogus")
	if err == nil {
		t.Fatal("expected a usage error")
	}

	exitCode := 0
	out := captureStdout(t, func() {
		exitCode = handleError(err)
	})
	if exitCode != epoerrors.ExitUsageError {
		t.Fatalf("expected exit code %d, got %d", epoerrors.ExitUsageError, exitCode)
	}
	var env struct {
		OK    bool `json:"ok"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if jsonErr := json.Unmarshal([]byte(out), &env); jsonErr != nil {
		t.Fatalf("stdout is not a JSON envelope: %v: %s", jsonErr, out)
	}
	if env.OK || env.Error.Type != "VALIDATION_ERROR" || !strings.Contains(env.Error.Message, `unknown command "bogus" for "epo pub"`) {
		t.Fatalf("unexpected envelope: %s", out)
	}
}

func TestCommandGroupWithoutArgsStillPrintsHelp(t *testing.T) {
	out, err := runForTest(t, rootCmd, "pub")
	if err != nil {
		t.Fatalf("expected help with exit 0, got: %v", err)
	}
	if !strings.Contains(out, "epo pub [command]") || !strings.Contains(out, "search") {
		t.Fatalf("expected group help, got: %.300s", out)
	}
}

func TestInvalidFormatRejectedBeforeAnyWork(t *testing.T) {
	stub := installOPSStub(t, func(req *http.Request) (int, http.Header, string) {
		return http.StatusOK, nil, `{}`
	})
	setGlobalFlag(t, &flagFormat, "json")

	out, err := runForTest(t, rootCmd, "family", "get", "EP.1000000.A1", "-f", "yaml")
	if err == nil {
		t.Fatalf("expected a format error, got output: %s", out)
	}
	if cliErr := mapError(err); cliErr.Type != "VALIDATION_ERROR" || !strings.Contains(cliErr.Message, `unsupported format "yaml"`) {
		t.Fatalf("unexpected error: %s (%s)", cliErr.Type, cliErr.Message)
	}
	if got := len(stub.dataRequests()); got != 0 {
		t.Fatalf("format must be validated before calling OPS, got %d request(s)", got)
	}
	if stub.tokenCalls != 0 {
		t.Fatalf("format must be validated before requesting a token, got %d token request(s)", stub.tokenCalls)
	}
}
