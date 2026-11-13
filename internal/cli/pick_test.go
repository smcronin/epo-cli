package cli

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func TestPickKeepsListShapeForSingleHit(t *testing.T) {
	for _, total := range []int{1, 3} {
		installOPSStub(t, searchPageStub(t, "ops:biblio-search", total, 2000))
		setGlobalFlag(t, &flagPick, "docNumber,country")

		out, err := runForTest(t, newPubCmd(), "search", "--query", "x", "--flat")
		if err != nil {
			t.Fatalf("total=%d: unexpected error: %v", total, err)
		}
		env := parseSearchEnvelope(t, out)
		if len(env.Results) != total {
			t.Fatalf("total=%d: expected a %d-element results array, got %d: %s", total, total, len(env.Results), out)
		}
		var row map[string]any
		if err := json.Unmarshal(env.Results[0], &row); err != nil {
			t.Fatalf("total=%d: parse row: %v", total, err)
		}
		if row["country"] != "EP" || row["docNumber"] == nil || len(row) != 2 {
			t.Fatalf("total=%d: unexpected projected row: %v", total, row)
		}
		if len(env.Warnings) != 0 {
			t.Fatalf("total=%d: unexpected warnings: %v", total, env.Warnings)
		}
	}
}

func TestPickRowFieldsWinOverEnvelopeNames(t *testing.T) {
	setGlobalFlag(t, &flagPick, "command,summary")

	env := buildSuccessEnvelope(&cobra.Command{Use: "methods"}, responsePayload{
		Service: "catalog",
		Results: buildMethodCatalog(),
	})

	rows, ok := env.Results.([]map[string]any)
	if !ok {
		t.Fatalf("expected a list of rows, got %T", env.Results)
	}
	if len(rows) != len(buildMethodCatalog()) {
		t.Fatalf("expected one row per catalog entry, got %d", len(rows))
	}
	for i, row := range rows {
		if row["command"] == nil || row["summary"] == nil {
			t.Fatalf("row %d lost a picked field: %v", i, row)
		}
	}
	if len(env.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", env.Warnings)
	}
}

func TestPickSingleObjectStaysObject(t *testing.T) {
	setGlobalFlag(t, &flagPick, "ok,tokenType")

	env := buildSuccessEnvelope(&cobra.Command{Use: "check"}, responsePayload{
		Results: map[string]any{"ok": true, "tokenType": "BearerToken", "expiresIn": "1200"},
	})
	row, ok := env.Results.(map[string]any)
	if !ok {
		t.Fatalf("expected an object, got %T", env.Results)
	}
	if row["ok"] != true || row["tokenType"] != "BearerToken" || len(row) != 2 {
		t.Fatalf("unexpected projection: %v", row)
	}
}

func TestPickNumberConversionStaysObject(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "number_convert_input.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var input map[string]any
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	setGlobalFlag(t, &flagPick, "outputDocNumber,outputKind")

	env := buildSuccessEnvelope(&cobra.Command{Use: "convert"}, responsePayload{Results: input})
	row, ok := env.Results.(map[string]any)
	if !ok {
		t.Fatalf("a number conversion is one record and must stay an object, got %T", env.Results)
	}
	if _, has := row["outputDocNumber"]; !has {
		t.Fatalf("unexpected projection: %v", row)
	}
}

func TestPickEnvelopePathStillProjectsEnvelope(t *testing.T) {
	setGlobalFlag(t, &flagPick, "results.suggested_retrieval_commands")

	commands := []string{"epo pub claims EP.1000000.A1 -f json -q"}
	env := buildSuccessEnvelope(&cobra.Command{Use: "fulltext"}, responsePayload{
		Results: map[string]any{
			"suggested_retrieval_commands": commands,
			"other":                        "dropped",
		},
	})
	row, ok := env.Results.(map[string]any)
	if !ok {
		t.Fatalf("expected an envelope-level object, got %T", env.Results)
	}
	got, ok := row["results.suggested_retrieval_commands"].([]string)
	if !ok || len(got) != 1 || len(row) != 1 {
		t.Fatalf("unexpected envelope projection: %v", row)
	}
	if len(env.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", env.Warnings)
	}
}

func TestPickMixedEnvelopeAndRowFieldsProjectsRows(t *testing.T) {
	setGlobalFlag(t, &flagPick, "pagination.total,reference")

	env := buildSuccessEnvelope(&cobra.Command{Use: "search"}, responsePayload{
		Pagination: map[string]any{"total": 2},
		Results: []map[string]any{
			{"reference": "EP1000000A1", "title": "One"},
			{"reference": "EP1000001A1", "title": "Two"},
		},
	})
	rows, ok := env.Results.([]map[string]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("expected two projected rows, got %#v", env.Results)
	}
	if rows[0]["reference"] != "EP1000000A1" {
		t.Fatalf("row field was dropped: %v", rows[0])
	}
	if len(env.Warnings) != 1 || !strings.Contains(env.Warnings[0], `"pagination.total"`) {
		t.Fatalf("expected one warning about the unapplied envelope path, got %v", env.Warnings)
	}
}

func TestPickUnmatchedFieldAddsWarning(t *testing.T) {
	setGlobalFlag(t, &flagPick, "reference,noSuchField")

	env := buildSuccessEnvelope(&cobra.Command{Use: "search"}, responsePayload{
		Results:  []map[string]any{{"reference": "EP1000000A1"}},
		Warnings: []string{"existing warning"},
	})
	if len(env.Warnings) != 2 || env.Warnings[0] != "existing warning" {
		t.Fatalf("expected the existing warning plus one pick warning, got %v", env.Warnings)
	}
	if !strings.Contains(env.Warnings[1], `"noSuchField"`) || strings.Contains(env.Warnings[1], `"reference"`) {
		t.Fatalf("unexpected pick warning: %q", env.Warnings[1])
	}
}

func TestPickWithNoRowsDoesNotWarn(t *testing.T) {
	setGlobalFlag(t, &flagPick, "reference")

	env := buildSuccessEnvelope(&cobra.Command{Use: "search"}, responsePayload{Results: []map[string]any{}})
	if len(env.Warnings) != 0 {
		t.Fatalf("no rows means nothing to match against, got %v", env.Warnings)
	}
	if rows, ok := env.Results.([]map[string]any); !ok || len(rows) != 0 {
		t.Fatalf("expected an empty list, got %#v", env.Results)
	}
}

func TestPickKeepsFailedBatchRows(t *testing.T) {
	failure := &epoerrors.CLIError{Code: 404, Type: "NOT_FOUND", Message: "nope"}
	batch := []map[string]any{
		{
			"query": "good",
			"ok":    true,
			"results": []map[string]any{
				{"reference": "EP1000000A1", "title": "One"},
			},
		},
		{"query": "bad", "ok": false, "error": failure},
	}

	projector := newPickProjector([]string{"reference"})
	rows, ok := projector.project(batch).([]map[string]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("expected two batch rows, got %#v", rows)
	}

	good, ok := rows[0]["results"].([]map[string]any)
	if !ok || len(good) != 1 || good[0]["reference"] != "EP1000000A1" || len(good[0]) != 1 {
		t.Fatalf("good row lost its projected results: %#v", rows[0])
	}
	if rows[0]["query"] != "good" || rows[0]["ok"] != true {
		t.Fatalf("good row lost its identity keys: %#v", rows[0])
	}
	if rows[1]["query"] != "bad" || rows[1]["ok"] != false || rows[1]["error"] != failure {
		t.Fatalf("failed row must keep query/ok/error, got %#v", rows[1])
	}
	if warnings := projector.warnings(false); len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
}

func TestPickKeepsFailedRowsInStdinBatch(t *testing.T) {
	installOPSStub(t, func(req *http.Request) (int, http.Header, string) {
		if req.URL.Query().Get("q") == "bad" {
			return opsNotFound(req)
		}
		return searchPageStub(t, "ops:biblio-search", 2, 2000)(req)
	})
	setGlobalFlag(t, &flagPick, "docNumber")

	var runErr error
	out := captureStdout(t, func() {
		runErr = runPublishedSearchFlat(&cobra.Command{Use: "search"}, []string{"good", "bad"}, "biblio", "", false, pubSearchSortNone)
	})
	if runErr != nil {
		t.Fatalf("a batch keeps per-item errors, got: %v", runErr)
	}

	var env struct {
		Results []struct {
			Query   string              `json:"query"`
			OK      bool                `json:"ok"`
			Error   *epoerrors.CLIError `json:"error"`
			Results []map[string]any    `json:"results"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v: %s", err, out)
	}
	if len(env.Results) != 2 {
		t.Fatalf("expected one row per query, got %d: %s", len(env.Results), out)
	}
	good, bad := env.Results[0], env.Results[1]
	if good.Query != "good" || !good.OK || len(good.Results) != 2 || good.Results[0]["docNumber"] == nil {
		t.Fatalf("good query lost its projected rows: %s", out)
	}
	if bad.Query != "bad" || bad.OK || bad.Error == nil || bad.Error.Type != "NOT_FOUND" {
		t.Fatalf("failed query must keep query/ok/error: %s", out)
	}
}

func TestPickAppliesToSummaryTopResults(t *testing.T) {
	installOPSStub(t, searchPageStub(t, "ops:biblio-search", 3, 2000))
	setGlobalFlag(t, &flagPick, "")

	out, err := runForTest(t, newPubCmd(), "search", "--query", "x", "--summary", "--flat-pick", "docNumber,country")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env struct {
		Results struct {
			Query      string           `json:"query"`
			Total      int              `json:"total"`
			TopResults []map[string]any `json:"topResults"`
		} `json:"results"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v: %s", err, out)
	}
	if env.Results.Query != "x" || env.Results.Total != 3 {
		t.Fatalf("summary keys must survive --pick: %s", out)
	}
	if len(env.Results.TopResults) != 3 {
		t.Fatalf("expected three projected rows, got %d: %s", len(env.Results.TopResults), out)
	}
	for _, row := range env.Results.TopResults {
		if row["country"] != "EP" || row["docNumber"] == nil || len(row) != 2 {
			t.Fatalf("unexpected projected row %v: %s", row, out)
		}
	}
	for _, warning := range env.Warnings {
		if strings.Contains(warning, "--pick") {
			t.Fatalf("unexpected pick warning: %q", warning)
		}
	}
}

func TestIsEnvelopePickPath(t *testing.T) {
	cases := map[string]bool{
		"results.suggested_retrieval_commands": true,
		"results[0].title":                     true,
		"pagination.total":                     true,
		"quota.hourUsed":                       true,
		"throttle.system":                      true,
		"request.path":                         true,
		"warnings":                             true,
		"warnings[0]":                          true,
		"results":                              false,
		"pagination":                           false,
		"ok":                                   false,
		"command":                              false,
		"service":                              false,
		"version":                              false,
		"reference":                            false,
	}
	for field, want := range cases {
		if got := isEnvelopePickPath(field); got != want {
			t.Fatalf("isEnvelopePickPath(%q) = %v, want %v", field, got, want)
		}
	}
}
