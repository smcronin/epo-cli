package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

// failingFromStub serves pages like searchPageStub until a request's range
// starts at or after failFrom, then answers with fail. A failFrom of 1 fails
// the first page.
func failingFromStub(t *testing.T, container string, total, ceiling, failFrom int, fail func(req *http.Request) (int, http.Header, string)) func(req *http.Request) (int, http.Header, string) {
	t.Helper()
	pages := searchPageStub(t, container, total, ceiling)
	return func(req *http.Request) (int, http.Header, string) {
		begin := 1
		if raw := req.Header.Get("Range"); raw != "" {
			begin, _ = strconv.Atoi(strings.SplitN(raw, "-", 2)[0])
		}
		if begin >= failFrom {
			return fail(req)
		}
		return pages(req)
	}
}

// opsThrottled is a 429 whose Retry-After cannot fit a short --timeout, so
// the client returns at once instead of sleeping.
func opsThrottled(req *http.Request) (int, http.Header, string) {
	return http.StatusTooManyRequests, http.Header{"Retry-After": {"60000"}}, `<error><code>429</code></error>`
}

type partialEnvelope struct {
	OK      bool `json:"ok"`
	Partial bool `json:"partial"`
	Error   struct {
		Code   int    `json:"code"`
		Type   string `json:"type"`
		Hint   string `json:"hint"`
		Reason string `json:"reason"`
	} `json:"error"`
	Command string `json:"command"`
	Service string `json:"service"`
	Request struct {
		Path string `json:"path"`
		All  bool   `json:"all"`
	} `json:"request"`
	Pagination struct {
		Offset       int    `json:"offset"`
		Limit        int    `json:"limit"`
		Total        int    `json:"total"`
		PagesFetched int    `json:"pagesFetched"`
		Returned     int    `json:"returned"`
		FailedRange  string `json:"failedRange"`
		ResumeRange  string `json:"resumeRange"`
	} `json:"pagination"`
	Results  json.RawMessage `json:"results"`
	Warnings []string        `json:"warnings"`
	Version  string          `json:"version"`
}

// errorEnvelopeFor captures what handleError prints for err in JSON mode.
func errorEnvelopeFor(t *testing.T, err error) (partialEnvelope, string, int) {
	t.Helper()
	exitCode := 0
	out := captureStdout(t, func() {
		exitCode = handleError(err)
	})
	var env partialEnvelope
	if jsonErr := json.Unmarshal([]byte(out), &env); jsonErr != nil {
		t.Fatalf("error output is not a JSON envelope: %v: %.300s", jsonErr, out)
	}
	return env, out, exitCode
}

func resultRows(t *testing.T, raw json.RawMessage) []map[string]any {
	t.Helper()
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("results is not a row list: %v: %.200s", err, string(raw))
	}
	return rows
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stderr = writer
	captured := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(reader)
		captured <- string(data)
	}()

	fn()

	os.Stderr = original
	_ = writer.Close()
	return <-captured
}

func TestPubSearchAllMidRunFailureKeepsFetchedRowsAndResumes(t *testing.T) {
	installOPSStub(t, failingFromStub(t, "ops:biblio-search", 450, 2000, 201, opsThrottled))
	setGlobalFlag(t, &flagAll, true)
	setGlobalFlag(t, &flagTimeout, 2)

	out, err := runForTest(t, newPubCmd(), "search", "--query", "x", "--flat")
	if err == nil {
		t.Fatalf("expected the run to fail, got: %.200s", out)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("the command itself must not print a success envelope, got: %.200s", out)
	}
	cliErr := mapError(err)
	if cliErr.Type != "RATE_LIMITED" || cliErr.Code != 429 {
		t.Fatalf("the error type must be unchanged, got %s/%d", cliErr.Type, cliErr.Code)
	}

	env, raw, exitCode := errorEnvelopeFor(t, err)
	if exitCode != epoerrors.ExitRateLimited {
		t.Fatalf("the exit code must be unchanged, got %d", exitCode)
	}
	if env.OK || !env.Partial || env.Error.Type != "RATE_LIMITED" || env.Error.Code != 429 || !strings.Contains(env.Error.Hint, "60000 ms") {
		t.Fatalf("unexpected error part of the envelope: %.400s", raw)
	}
	if env.Command != "pub search" || env.Service != "published-data" || !env.Request.All || env.Version == "" {
		t.Fatalf("unexpected envelope metadata: %.400s", raw)
	}
	p := env.Pagination
	if p.Offset != 1 || p.Limit != 100 || p.Total != 450 || p.PagesFetched != 2 || p.Returned != 200 {
		t.Fatalf("unexpected pagination: %+v", p)
	}
	if p.FailedRange != "201-300" || p.ResumeRange != "201-300" {
		t.Fatalf("expected failedRange/resumeRange 201-300, got %q / %q", p.FailedRange, p.ResumeRange)
	}
	partialRows := resultRows(t, env.Results)
	if len(partialRows) != 200 {
		t.Fatalf("expected the 200 rows fetched before the failure, got %d", len(partialRows))
	}
	if !hasWarning(env.Warnings, "--all stopped after 2 pages", "200 of 450 retrievable results", "--range 201-300 --all") {
		t.Fatalf("expected a resume warning, got %v", env.Warnings)
	}

	// Resuming with the reported range must start at that offset and fetch
	// exactly the rows the first run did not get.
	stub := installOPSStub(t, searchPageStub(t, "ops:biblio-search", 450, 2000))
	out, err = runForTest(t, newPubCmd(), "search", "--query", "x", "--flat", "--range", p.ResumeRange)
	if err != nil {
		t.Fatalf("resume failed: %v", mapError(err).Message)
	}
	if ranges := requestedRanges(stub); len(ranges) != 3 || ranges[0] != "201-300" || ranges[1] != "301-400" || ranges[2] != "401-500" {
		t.Fatalf("resume must start at the reported range, got %v", ranges)
	}
	resumed := parseSearchEnvelope(t, out)
	if len(resumed.Results) != 250 {
		t.Fatalf("expected the remaining 250 rows, got %d", len(resumed.Results))
	}

	seen := map[string]bool{}
	for _, row := range partialRows {
		seen[row["docNumber"].(string)] = true
	}
	for _, rawRow := range resumed.Results {
		var row map[string]any
		if err := json.Unmarshal(rawRow, &row); err != nil {
			t.Fatalf("parse resumed row: %v", err)
		}
		number := row["docNumber"].(string)
		if seen[number] {
			t.Fatalf("resume returned %s again", number)
		}
		seen[number] = true
	}
	if len(seen) != 450 || !seen["1000001"] || !seen["1000201"] || !seen["1000450"] {
		t.Fatalf("partial rows plus resumed rows must cover all 450 results, got %d", len(seen))
	}
}

func TestAllFirstPageFailureHasNoPartialEnvelope(t *testing.T) {
	cases := []struct {
		name      string
		container string
		cmd       func() *cobra.Command
		args      []string
	}{
		{"pub search", "ops:biblio-search", newPubCmd, []string{"search", "--query", "x", "--flat"}},
		{"pub search summary", "ops:biblio-search", newPubCmd, []string{"search", "--query", "x", "--summary"}},
		{"register search", "ops:register-search", newRegisterCmd, []string{"search", "--q", "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			installOPSStub(t, failingFromStub(t, tc.container, 450, 0, 1, opsThrottled))
			setGlobalFlag(t, &flagAll, true)
			setGlobalFlag(t, &flagTimeout, 2)

			_, err := runForTest(t, tc.cmd(), tc.args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			env, raw, exitCode := errorEnvelopeFor(t, err)
			if exitCode != epoerrors.ExitRateLimited || env.OK || env.Error.Type != "RATE_LIMITED" {
				t.Fatalf("unexpected error envelope: %s", raw)
			}
			for _, key := range []string{`"partial"`, `"results"`, `"pagination"`, `"warnings"`, `"command"`, `"request"`} {
				if strings.Contains(raw, key) {
					t.Fatalf("a first-page failure must keep the plain error envelope, found %s: %s", key, raw)
				}
			}
		})
	}
}

func TestRegisterSearchAllMidRunFailureKeepsFetchedRowsAndResumes(t *testing.T) {
	installOPSStub(t, failingFromStub(t, "ops:register-search", 120, 0, 51, opsThrottled))
	setGlobalFlag(t, &flagAll, true)
	setGlobalFlag(t, &flagTimeout, 2)

	_, err := runForTest(t, newRegisterCmd(), "search", "--q", "pa=IBM")
	if err == nil {
		t.Fatal("expected the run to fail")
	}
	env, raw, exitCode := errorEnvelopeFor(t, err)
	if exitCode != epoerrors.ExitRateLimited || env.OK || !env.Partial || env.Error.Type != "RATE_LIMITED" {
		t.Fatalf("unexpected error envelope: %.400s", raw)
	}
	if env.Service != "register" || env.Command != "register search" {
		t.Fatalf("unexpected envelope metadata: %.400s", raw)
	}
	p := env.Pagination
	if p.Limit != 25 || p.Total != 120 || p.PagesFetched != 2 || p.Returned != 50 || p.FailedRange != "51-75" || p.ResumeRange != "51-75" {
		t.Fatalf("unexpected pagination: %+v", p)
	}
	var docs []json.RawMessage
	if err := json.Unmarshal(env.Results, &docs); err != nil || len(docs) != 50 {
		t.Fatalf("expected the 50 register documents fetched so far, got %d (%v)", len(docs), err)
	}
	if !hasWarning(env.Warnings, "--all stopped after 2 pages", "50 of 120 retrievable results", "--range 51-75 --all") {
		t.Fatalf("expected a resume warning, got %v", env.Warnings)
	}

	stub := installOPSStub(t, searchPageStub(t, "ops:register-search", 120, 0))
	out, err := runForTest(t, newRegisterCmd(), "search", "--q", "pa=IBM", "--range", p.ResumeRange)
	if err != nil {
		t.Fatalf("resume failed: %v", mapError(err).Message)
	}
	if ranges := requestedRanges(stub); len(ranges) != 3 || ranges[0] != "51-75" || ranges[2] != "101-125" {
		t.Fatalf("resume must start at the reported range, got %v", ranges)
	}
	if resumed := parseSearchEnvelope(t, out); resumed.Pagination.Returned != 70 {
		t.Fatalf("expected the remaining 70 documents, got %d", resumed.Pagination.Returned)
	}
}

func TestPubSearchSummaryAllMidRunFailureKeepsSummaryOfFetchedRows(t *testing.T) {
	installOPSStub(t, failingFromStub(t, "ops:biblio-search", 450, 2000, 201, opsThrottled))
	setGlobalFlag(t, &flagAll, true)
	setGlobalFlag(t, &flagTimeout, 2)

	_, err := runForTest(t, newPubCmd(), "search", "--query", "x", "--summary", "--sort", "pub-date-desc")
	if err == nil {
		t.Fatal("expected the run to fail")
	}
	env, raw, exitCode := errorEnvelopeFor(t, err)
	if exitCode != epoerrors.ExitRateLimited || env.OK || !env.Partial || env.Error.Type != "RATE_LIMITED" {
		t.Fatalf("unexpected error envelope: %.400s", raw)
	}
	if env.Pagination.PagesFetched != 2 || env.Pagination.Returned != 200 || env.Pagination.ResumeRange != "201-300" {
		t.Fatalf("unexpected pagination: %+v", env.Pagination)
	}

	var summary struct {
		Query      string           `json:"query"`
		Total      int              `json:"total"`
		Examined   int              `json:"examined"`
		Complete   bool             `json:"complete"`
		SortedBy   string           `json:"sortedBy"`
		TopResults []map[string]any `json:"topResults"`
	}
	if err := json.Unmarshal(env.Results, &summary); err != nil {
		t.Fatalf("parse summary: %v: %.300s", err, raw)
	}
	if summary.Total != 450 || summary.Examined != 200 || summary.Complete || len(summary.TopResults) != 10 || summary.SortedBy != "pub-date-desc" {
		t.Fatalf("unexpected partial summary: %+v", summary)
	}
	if !hasWarning(env.Warnings, "--all stopped after 2 pages", "--range 201-300 --all", "topResults cover only the rows fetched so far") {
		t.Fatalf("expected a resume warning, got %v", env.Warnings)
	}
}

func TestPartialFailureCarriesQuotaReason(t *testing.T) {
	installOPSStub(t, failingFromStub(t, "ops:biblio-search", 450, 2000, 101, func(req *http.Request) (int, http.Header, string) {
		return http.StatusForbidden, http.Header{"X-Rejection-Reason": {"IndividualQuotaPerHour"}}, fairUseBody
	}))
	setGlobalFlag(t, &flagAll, true)

	_, err := runForTest(t, newPubCmd(), "search", "--query", "x", "--flat")
	if err == nil {
		t.Fatal("expected the run to fail")
	}
	env, raw, exitCode := errorEnvelopeFor(t, err)
	if exitCode != epoerrors.ExitRateLimited || !env.Partial || env.Error.Code != 403 || env.Error.Type != "RATE_LIMITED" || env.Error.Reason != "IndividualQuotaPerHour" {
		t.Fatalf("unexpected error envelope: %.400s", raw)
	}
	if env.Pagination.PagesFetched != 1 || env.Pagination.ResumeRange != "101-200" || len(resultRows(t, env.Results)) != 100 {
		t.Fatalf("unexpected salvage: %+v", env.Pagination)
	}
	if !hasWarning(env.Warnings, "--all stopped after 1 page (") {
		t.Fatalf("expected a resume warning, got %v", env.Warnings)
	}
}

func TestPartialFailureRowsHonourPick(t *testing.T) {
	installOPSStub(t, failingFromStub(t, "ops:biblio-search", 450, 2000, 201, opsThrottled))
	setGlobalFlag(t, &flagAll, true)
	setGlobalFlag(t, &flagTimeout, 2)
	setGlobalFlag(t, &flagPick, "docNumber")

	_, err := runForTest(t, newPubCmd(), "search", "--query", "x", "--flat")
	if err == nil {
		t.Fatal("expected the run to fail")
	}
	env, raw, _ := errorEnvelopeFor(t, err)
	rows := resultRows(t, env.Results)
	if !env.Partial || len(rows) != 200 {
		t.Fatalf("unexpected partial envelope: %.300s", raw)
	}
	if len(rows[0]) != 1 || rows[0]["docNumber"] != "1000001" {
		t.Fatalf("partial rows must be projected by --pick, got %v", rows[0])
	}
}

func TestPartialFailureNonJSONFormatsWriteRowsToStdoutAndErrorToStderr(t *testing.T) {
	cases := []struct {
		format    string
		wantLines int
		check     func(t *testing.T, lines []string)
	}{
		{"ndjson", 200, func(t *testing.T, lines []string) {
			var row map[string]any
			if err := json.Unmarshal([]byte(lines[0]), &row); err != nil || row["docNumber"] != "1000001" {
				t.Fatalf("first ndjson line is not the first row: %v: %s", err, lines[0])
			}
		}},
		{"csv", 201, func(t *testing.T, lines []string) {
			if !strings.Contains(lines[0], "docNumber") || !strings.Contains(lines[1], "1000001") {
				t.Fatalf("unexpected csv head: %q / %q", lines[0], lines[1])
			}
		}},
		{"table", 202, func(t *testing.T, lines []string) {
			if !strings.Contains(lines[0], "docNumber") || !strings.Contains(lines[2], "1000001") {
				t.Fatalf("unexpected table head: %q / %q", lines[0], lines[2])
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			installOPSStub(t, failingFromStub(t, "ops:biblio-search", 450, 2000, 201, opsThrottled))
			setGlobalFlag(t, &flagAll, true)
			setGlobalFlag(t, &flagTimeout, 2)
			setGlobalFlag(t, &flagFormat, tc.format)

			_, err := runForTest(t, newPubCmd(), "search", "--query", "x", "--flat")
			if err == nil {
				t.Fatal("expected the run to fail")
			}

			var stdout string
			exitCode := 0
			stderr := captureStderr(t, func() {
				stdout = captureStdout(t, func() {
					exitCode = handleError(err)
				})
			})
			if exitCode != epoerrors.ExitRateLimited {
				t.Fatalf("expected exit code %d, got %d", epoerrors.ExitRateLimited, exitCode)
			}

			lines := strings.Split(strings.TrimRight(stdout, "\r\n"), "\n")
			if len(lines) != tc.wantLines {
				t.Fatalf("expected %d stdout lines, got %d", tc.wantLines, len(lines))
			}
			tc.check(t, lines)
			if strings.Contains(stdout, "Error:") {
				t.Fatalf("the error must not be written to stdout: %.200s", stdout)
			}
			if !strings.Contains(stderr, "Error: API error (429)") || !strings.Contains(stderr, "Warning: --all stopped after 2 pages") || !strings.Contains(stderr, "--range 201-300 --all") {
				t.Fatalf("expected the error and the resume warning on stderr, got: %s", stderr)
			}
		})
	}
}

func TestStdinBatchAllKeepsPartialRowsOnTheFailedQuery(t *testing.T) {
	pages := searchPageStub(t, "ops:biblio-search", 450, 2000)
	installOPSStub(t, func(req *http.Request) (int, http.Header, string) {
		begin, _ := strconv.Atoi(strings.SplitN(req.Header.Get("Range"), "-", 2)[0])
		if req.URL.Query().Get("q") == "bad" && begin >= 201 {
			return opsThrottled(req)
		}
		return pages(req)
	})
	setGlobalFlag(t, &flagTimeout, 2)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runPublishedSearchAll(&cobra.Command{Use: "search"}, []string{"good", "bad"}, "biblio", "", false, pubSearchSortNone, true)
	})
	if runErr != nil {
		t.Fatalf("a batch keeps per-item errors, got: %v", runErr)
	}

	var env struct {
		OK      bool `json:"ok"`
		Results []struct {
			Query   string `json:"query"`
			OK      bool   `json:"ok"`
			Partial bool   `json:"partial"`
			Error   *struct {
				Type string `json:"type"`
			} `json:"error"`
			Pagination struct {
				Returned    int    `json:"returned"`
				ResumeRange string `json:"resumeRange"`
			} `json:"pagination"`
			Results  []map[string]any `json:"results"`
			Warnings []string         `json:"warnings"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v: %.300s", err, out)
	}
	if len(env.Results) != 2 {
		t.Fatalf("expected one row per query, got %d", len(env.Results))
	}
	good, bad := env.Results[0], env.Results[1]
	if !good.OK || good.Partial || len(good.Results) != 450 {
		t.Fatalf("the healthy query must be complete: ok=%v partial=%v rows=%d", good.OK, good.Partial, len(good.Results))
	}
	if bad.OK || !bad.Partial || bad.Error == nil || bad.Error.Type != "RATE_LIMITED" {
		t.Fatalf("the failed query must stay ok:false with its error and partial:true: %+v", bad)
	}
	if len(bad.Results) != 200 || bad.Pagination.Returned != 200 || bad.Pagination.ResumeRange != "201-300" {
		t.Fatalf("the failed query must keep its fetched rows and resume range: rows=%d %+v", len(bad.Results), bad.Pagination)
	}
	if !hasWarning(bad.Warnings, "--range 201-300 --all") {
		t.Fatalf("expected a resume warning on the failed row, got %v", bad.Warnings)
	}
}
