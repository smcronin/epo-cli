package cli

import "testing"

func TestParseSearchPagination(t *testing.T) {
	t.Parallel()

	results := map[string]any{
		"ops:world-patent-data": map[string]any{
			"ops:biblio-search": map[string]any{
				"@total-result-count": "10000",
				"ops:range": map[string]any{
					"@begin": "26",
					"@end":   "50",
				},
			},
		},
	}

	p := parseSearchPagination(results)
	if p == nil {
		t.Fatal("expected pagination")
	}
	if p["offset"] != 26 {
		t.Fatalf("expected offset 26, got %v", p["offset"])
	}
	if p["limit"] != 25 {
		t.Fatalf("expected limit 25, got %v", p["limit"])
	}
	if p["total"] != 10000 {
		t.Fatalf("expected total 10000, got %v", p["total"])
	}
	if p["hasMore"] != true {
		t.Fatalf("expected hasMore true, got %v", p["hasMore"])
	}
}

func TestMergePagination(t *testing.T) {
	t.Parallel()

	a := map[string]any{"offset": 1, "limit": 10}
	b := map[string]any{"total": 100, "hasMore": true}

	merged := mergePagination(a, b)
	if len(merged) != 4 {
		t.Fatalf("expected 4 keys, got %d", len(merged))
	}
	if merged["total"] != 100 {
		t.Fatalf("expected total 100, got %v", merged["total"])
	}
}
