package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// searchPageStub serves a paged OPS search result set of the given size. It
// rejects ranges that start beyond the result set, wider than OPS allows, or
// past the OPS ceiling, the way the live service does.
func searchPageStub(t *testing.T, container string, total, ceiling int) func(req *http.Request) (int, http.Header, string) {
	t.Helper()
	return func(req *http.Request) (int, http.Header, string) {
		begin, end := 1, 25
		if raw := req.Header.Get("Range"); raw != "" {
			parts := strings.SplitN(raw, "-", 2)
			begin, _ = strconv.Atoi(parts[0])
			end, _ = strconv.Atoi(parts[1])
		}
		if begin > total || end-begin+1 > 100 || (ceiling > 0 && end > ceiling) {
			return http.StatusBadRequest, nil, `<error><code>400</code><message>invalid range</message></error>`
		}
		if end > total {
			end = total
		}

		search := map[string]any{
			"@total-result-count": strconv.Itoa(total),
			"ops:range":           map[string]any{"@begin": strconv.Itoa(begin), "@end": strconv.Itoa(end)},
		}
		if container == "ops:register-search" {
			docs := make([]any, 0, end-begin+1)
			for i := begin; i <= end; i++ {
				docs = append(docs, map[string]any{
					"reg:bibliographic-data": map[string]any{
						"reg:application-reference": map[string]any{
							"reg:document-id": map[string]any{
								"reg:country":    map[string]any{"$": "EP"},
								"reg:doc-number": map[string]any{"$": fmt.Sprintf("%08d", i)},
							},
						},
					},
				})
			}
			search["reg:register-documents"] = map[string]any{"reg:register-document": docs}
		} else {
			refs := make([]any, 0, end-begin+1)
			for i := begin; i <= end; i++ {
				refs = append(refs, map[string]any{
					"@family-id": strconv.Itoa(i),
					"document-id": map[string]any{
						"@document-id-type": "docdb",
						"country":           map[string]any{"$": "EP"},
						"doc-number":        map[string]any{"$": strconv.Itoa(1000000 + i)},
						"kind":              map[string]any{"$": "A1"},
					},
				})
			}
			search["ops:search-result"] = map[string]any{"ops:publication-reference": refs}
		}

		body, err := json.Marshal(map[string]any{"ops:world-patent-data": map[string]any{container: search}})
		if err != nil {
			t.Fatalf("marshal stub page: %v", err)
		}
		return http.StatusOK, nil, string(body)
	}
}

type searchEnvelope struct {
	OK         bool `json:"ok"`
	Pagination struct {
		Total        int   `json:"total"`
		Limit        int   `json:"limit"`
		PagesFetched int   `json:"pagesFetched"`
		Returned     int   `json:"returned"`
		HasMore      *bool `json:"hasMore"`
	} `json:"pagination"`
	Results  []json.RawMessage `json:"results"`
	Warnings []string          `json:"warnings"`
}

func parseSearchEnvelope(t *testing.T, out string) searchEnvelope {
	t.Helper()
	var env searchEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v: %.300s", err, out)
	}
	return env
}

func requestedRanges(stub *opsStub) []string {
	ranges := []string{}
	for _, req := range stub.dataRequests() {
		ranges = append(ranges, req.Header.Get("Range"))
	}
	return ranges
}

func TestPubSearchAllStopsAtOPSCeiling(t *testing.T) {
	stub := installOPSStub(t, searchPageStub(t, "ops:biblio-search", 5000, 2000))
	setGlobalFlag(t, &flagAll, true)

	out, err := runForTest(t, newPubCmd(), "search", "--query", "applicant=IBM", "--flat")
	if err != nil {
		t.Fatalf("--all over a >2000-hit query must succeed with the first 2000 rows, got: %v", mapError(err).Message)
	}

	ranges := requestedRanges(stub)
	if len(ranges) != 20 || ranges[0] != "1-100" || ranges[len(ranges)-1] != "1901-2000" {
		t.Fatalf("expected 20 pages of 100 from 1-100 to 1901-2000, got %d: %v", len(ranges), ranges)
	}

	env := parseSearchEnvelope(t, out)
	if env.Pagination.Total != 5000 || env.Pagination.Returned != 2000 || len(env.Results) != 2000 {
		t.Fatalf("unexpected pagination: %+v (rows=%d)", env.Pagination, len(env.Results))
	}
	if len(env.Warnings) != 1 || !strings.Contains(env.Warnings[0], "2000") || !strings.Contains(env.Warnings[0], "5000") {
		t.Fatalf("expected a truncation warning naming the ceiling and total, got %v", env.Warnings)
	}
}

func TestPubSearchAllFetchesEveryPageBelowCeiling(t *testing.T) {
	stub := installOPSStub(t, searchPageStub(t, "ops:biblio-search", 150, 2000))
	setGlobalFlag(t, &flagAll, true)

	out, err := runForTest(t, newPubCmd(), "search", "--query", "applicant=IBM", "--flat")
	if err != nil {
		t.Fatalf("unexpected error: %v", mapError(err).Message)
	}
	if ranges := requestedRanges(stub); len(ranges) != 2 || ranges[0] != "1-100" || ranges[1] != "101-200" {
		t.Fatalf("unexpected page requests: %v", ranges)
	}
	env := parseSearchEnvelope(t, out)
	if env.Pagination.Returned != 150 || env.Pagination.Total != 150 || len(env.Warnings) != 0 {
		t.Fatalf("unexpected result: %+v warnings=%v", env.Pagination, env.Warnings)
	}
}

func TestPubSearchAllHonoursExplicitRangeUpToCeiling(t *testing.T) {
	stub := installOPSStub(t, searchPageStub(t, "ops:biblio-search", 5000, 2000))
	setGlobalFlag(t, &flagAll, true)

	_, err := runForTest(t, newPubCmd(), "search", "--query", "applicant=IBM", "--range", "1-75", "--flat")
	if err != nil {
		t.Fatalf("unexpected error: %v", mapError(err).Message)
	}
	ranges := requestedRanges(stub)
	if ranges[0] != "1-75" || ranges[len(ranges)-1] != "1951-2000" {
		t.Fatalf("expected the explicit page size clamped at the ceiling, got %v", ranges)
	}
}

func TestPubSearchSummaryAllStopsAtOPSCeiling(t *testing.T) {
	stub := installOPSStub(t, searchPageStub(t, "ops:biblio-search", 5000, 2000))
	setGlobalFlag(t, &flagAll, true)

	_, err := runForTest(t, newPubCmd(), "search", "--query", "applicant=IBM", "--summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", mapError(err).Message)
	}
	ranges := requestedRanges(stub)
	if len(ranges) != 20 || ranges[len(ranges)-1] != "1901-2000" {
		t.Fatalf("expected 20 pages ending at 1901-2000, got %d: %v", len(ranges), ranges)
	}
}

func TestRegisterSearchAllStopsOnTotal(t *testing.T) {
	stub := installOPSStub(t, searchPageStub(t, "ops:register-search", 50, 0))
	setGlobalFlag(t, &flagAll, true)

	out, err := runForTest(t, newRegisterCmd(), "search", "--q", "pa=IBM")
	if err != nil {
		t.Fatalf("a result count that is an exact multiple of the page size must not fail, got: %v", mapError(err).Message)
	}
	if ranges := requestedRanges(stub); len(ranges) != 2 || ranges[0] != "1-25" || ranges[1] != "26-50" {
		t.Fatalf("expected exactly two page requests, got %v", ranges)
	}
	env := parseSearchEnvelope(t, out)
	if env.Pagination.Returned != 50 || env.Pagination.Total != 50 || env.Pagination.PagesFetched != 2 {
		t.Fatalf("unexpected pagination: %+v", env.Pagination)
	}
}

func TestRegisterSearchReportsTotalAndHasMore(t *testing.T) {
	installOPSStub(t, searchPageStub(t, "ops:register-search", 781, 0))

	out, err := runForTest(t, newRegisterCmd(), "search", "--q", "pa=IBM")
	if err != nil {
		t.Fatalf("unexpected error: %v", mapError(err).Message)
	}
	var env struct {
		Pagination struct {
			Total   int   `json:"total"`
			Offset  int   `json:"offset"`
			Limit   int   `json:"limit"`
			HasMore *bool `json:"hasMore"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if env.Pagination.Total != 781 || env.Pagination.Limit != 25 || env.Pagination.HasMore == nil || !*env.Pagination.HasMore {
		t.Fatalf("expected pagination total=781 limit=25 hasMore=true, got %+v", env.Pagination)
	}
}
