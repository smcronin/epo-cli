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

func TestNormalizePubSearchSort(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":              pubSearchSortNone,
		"none":          pubSearchSortNone,
		"pub-date-asc":  pubSearchSortDateAsc,
		"date-asc":      pubSearchSortDateAsc,
		"asc":           pubSearchSortDateAsc,
		"pub-date-desc": pubSearchSortDateDesc,
		"date-desc":     pubSearchSortDateDesc,
		"desc":          pubSearchSortDateDesc,
	}
	for in, want := range tests {
		got, err := normalizePubSearchSort(in)
		if err != nil {
			t.Fatalf("normalizePubSearchSort(%q) error: %v", in, err)
		}
		if got != want {
			t.Fatalf("normalizePubSearchSort(%q) = %q, want %q", in, got, want)
		}
	}

	if _, err := normalizePubSearchSort("bad-order"); err == nil {
		t.Fatal("expected unsupported sort mode error")
	}
}

func TestFlattenPublishedSearchItemExchangeDocument(t *testing.T) {
	t.Parallel()

	item := map[string]any{
		"exchange-document": map[string]any{
			"@family-id": "12345",
			"@system":    "ops.epo.org",
			"bibliographic-data": map[string]any{
				"invention-title": map[string]any{"$": "Test title", "@lang": "en"},
				"publication-reference": map[string]any{
					"document-id": []any{
						map[string]any{
							"@document-id-type": "docdb",
							"country":           map[string]any{"$": "EP"},
							"doc-number":        map[string]any{"$": "1234567"},
							"kind":              map[string]any{"$": "A1"},
							"date":              map[string]any{"$": "20250115"},
						},
					},
				},
			},
		},
	}

	row := flattenPublishedSearchItem(item)
	if row["country"] != "EP" || row["docNumber"] != "1234567" || row["kind"] != "A1" {
		t.Fatalf("unexpected flattened reference: %#v", row)
	}
	if row["pubDate"] != "20250115" {
		t.Fatalf("unexpected pubDate: %#v", row["pubDate"])
	}
	if row["title"] != "Test title" {
		t.Fatalf("unexpected title: %#v", row["title"])
	}
}

func TestSortPublishedSearchItemsByDateDesc(t *testing.T) {
	t.Parallel()

	items := []any{
		map[string]any{
			"document-id": map[string]any{
				"country":    map[string]any{"$": "EP"},
				"doc-number": map[string]any{"$": "1000001"},
				"kind":       map[string]any{"$": "A1"},
				"date":       map[string]any{"$": "20240101"},
			},
		},
		map[string]any{
			"document-id": map[string]any{
				"country":    map[string]any{"$": "EP"},
				"doc-number": map[string]any{"$": "1000002"},
				"kind":       map[string]any{"$": "A1"},
				"date":       map[string]any{"$": "20250101"},
			},
		},
	}

	sortPublishedSearchItems(items, pubSearchSortDateDesc)
	first := flattenPublishedSearchItem(items[0])
	if first["docNumber"] != "1000002" {
		t.Fatalf("unexpected first sorted docNumber: %v", first["docNumber"])
	}
}
