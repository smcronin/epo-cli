package cli

import "testing"

func TestNormalizeRowsExtractCommands(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"commands": []any{
			map[string]any{"command": "epo auth check", "service": "auth"},
			map[string]any{"command": "epo pub search", "service": "published-data"},
		},
	}

	rows, ok := normalizeRows(input)
	if !ok {
		t.Fatal("expected rows")
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[1]["command"] != "epo pub search" {
		t.Fatalf("unexpected row value: %v", rows[1]["command"])
	}
}

func TestNormalizeRowsExtractSearchResults(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"ops:world-patent-data": map[string]any{
			"ops:biblio-search": map[string]any{
				"ops:search-result": map[string]any{
					"ops:publication-reference": []any{
						map[string]any{
							"@family-id": "123",
							"@system":    "ops.epo.org",
							"document-id": map[string]any{
								"@document-id-type": "docdb",
								"country":           map[string]any{"$": "EP"},
								"doc-number":        map[string]any{"$": "1000000"},
								"kind":              map[string]any{"$": "A1"},
							},
						},
					},
				},
			},
		},
	}

	rows, ok := normalizeRows(input)
	if !ok {
		t.Fatal("expected extracted rows")
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row["country"] != "EP" || row["docNumber"] != "1000000" || row["kind"] != "A1" {
		t.Fatalf("unexpected row: %#v", row)
	}
}

func TestNormalizeRowsExtractRegisterResults(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"ops:world-patent-data": map[string]any{
			"ops:register-search": map[string]any{
				"reg:register-documents": map[string]any{
					"reg:register-document": []any{
						map[string]any{
							"reg:bibliographic-data": map[string]any{
								"reg:application-reference": map[string]any{
									"reg:document-id": map[string]any{
										"reg:country":    map[string]any{"$": "EP"},
										"reg:doc-number": map[string]any{"$": "25734468"},
									},
								},
								"reg:publication-reference": map[string]any{
									"reg:document-id": map[string]any{
										"reg:country":    map[string]any{"$": "WO"},
										"reg:date":       map[string]any{"$": "20260108"},
										"reg:doc-number": map[string]any{"$": "2026009065"},
									},
								},
								"reg:invention-title": map[string]any{"$": "Test title", "@lang": "en"},
							},
							"reg:ep-patent-statuses": map[string]any{
								"reg:ep-patent-status": map[string]any{"$": "Pending"},
							},
						},
					},
				},
			},
		},
	}

	rows, ok := normalizeRows(input)
	if !ok || len(rows) != 1 {
		t.Fatalf("expected 1 row, got %v", len(rows))
	}
	row := rows[0]
	if row["appDocNumber"] != "25734468" || row["pubDocNumber"] != "2026009065" || row["title"] != "Test title" {
		t.Fatalf("unexpected register row: %#v", row)
	}
}

func TestNormalizeRowsExtractUsageResults(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"environments": []any{
			map[string]any{"name": "prod"},
		},
		"metaData": map[string]any{
			"notices": []any{"one", "two"},
		},
	}

	rows, ok := normalizeRows(input)
	if !ok || len(rows) != 1 {
		t.Fatalf("expected usage row, got %d", len(rows))
	}
	if rows[0]["environment"] != "prod" {
		t.Fatalf("unexpected environment: %#v", rows[0])
	}
	if rows[0]["notices"] != "one | two" {
		t.Fatalf("unexpected notices: %#v", rows[0])
	}
}
