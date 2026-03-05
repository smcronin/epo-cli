package cli

import (
	"strings"
	"testing"
)

func TestSplitProjectionPathWithArrayIndices(t *testing.T) {
	got := splitProjectionPath("results.environments[0].dimensions[1].metrics")
	want := []string{"results", "environments", "0", "dimensions", "1", "metrics"}
	if len(got) != len(want) {
		t.Fatalf("unexpected segment count: got=%d want=%d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("segment %d mismatch: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestProjectBatchResultsByFields(t *testing.T) {
	batch := []any{
		map[string]any{
			"input": "q1",
			"ok":    true,
			"results": []any{
				map[string]any{"reference": "EP1000000A1", "title": "One"},
				map[string]any{"reference": "EP1000001A1", "title": "Two"},
			},
		},
	}
	projected, ok := projectBatchResultsByFields(batch, []string{"reference"})
	if !ok {
		t.Fatal("expected batch projection to match")
	}
	rows := projected.([]map[string]any)
	if len(rows) != 1 {
		t.Fatalf("expected one batch row, got %d", len(rows))
	}
	results, ok := rows[0]["results"].([]map[string]any)
	if !ok || len(results) != 2 {
		t.Fatalf("expected projected inner results, got %#v", rows[0]["results"])
	}
	if results[0]["reference"] != "EP1000000A1" {
		t.Fatalf("unexpected projected value: %#v", results[0])
	}
}

func TestProjectEnvelopeIfRequested(t *testing.T) {
	prev := flagPick
	flagPick = "quota.hourUsed,results.environments[0].dimensions[0].metrics[0].value"
	defer func() { flagPick = prev }()

	env := successEnvelope{
		Quota: map[string]any{"hourUsed": 7},
		Results: map[string]any{
			"environments": []any{
				map[string]any{
					"dimensions": []any{
						map[string]any{
							"metrics": []any{
								map[string]any{"value": 123},
							},
						},
					},
				},
			},
		},
	}
	projected, ok := projectEnvelopeIfRequested(env)
	if !ok {
		t.Fatal("expected envelope projection to succeed")
	}
	row := projected.(map[string]any)
	if row["quota.hourUsed"] != 7 {
		t.Fatalf("unexpected quota projection: %#v", row)
	}
	if row["results.environments[0].dimensions[0].metrics[0].value"] != 123 {
		t.Fatalf("unexpected nested projection: %#v", row)
	}
}

func TestValidateCQLDateSyntax(t *testing.T) {
	if err := validateCQLDateSyntax(`pa=IBM and pd within "20250101 20251231"`); err != nil {
		t.Fatalf("unexpected validation failure: %v", err)
	}
	if err := validateCQLDateSyntax("pa=IBM and pd>=20250101"); err == nil {
		t.Fatal("expected invalid pd>=YYYYMMDD syntax error")
	}
}

func TestResolvePubInputFormatAndClaimsRouting(t *testing.T) {
	if got := resolvePubInputFormat("EP.1000000.A1", "auto"); got != "docdb" {
		t.Fatalf("unexpected auto format: %s", got)
	}
	if got := resolvePubInputFormat("EP1000000A1", "auto"); got != "epodoc" {
		t.Fatalf("unexpected auto format for epodoc reference: %s", got)
	}

	format, reference := routeClaimsAndDescriptionInput("epodoc", "EP1000000A1", "claims")
	if format != "docdb" || reference != "EP.1000000.A1" {
		t.Fatalf("unexpected claims routing: %s %s", format, reference)
	}
}

func TestNormalizeImageFetchPath(t *testing.T) {
	got := normalizeImageFetchPath("https://ops.epo.org/rest-services/published-data/images/EP/1000000/A1/fullimage")
	if got != "EP/1000000/A1/fullimage" {
		t.Fatalf("unexpected normalized link path: %s", got)
	}
}

func TestWithImageFetchPaths(t *testing.T) {
	input := map[string]any{
		"document-instance": map[string]any{
			"@link": "published-data/images/EP/1000000/A1/fullimage",
		},
	}
	out := withImageFetchPaths(input).(map[string]any)
	di := asAnyMap(out["document-instance"])
	if di["fetch_path"] != "EP/1000000/A1/fullimage" {
		t.Fatalf("expected fetch_path, got %#v", di)
	}
}

func TestWithFulltextSuggestions(t *testing.T) {
	body := []byte("<root><kind>A1</kind><kind>B1</kind></root>")
	out := withFulltextSuggestions("/published-data/publication/epodoc/EP1000000/fulltext", body, map[string]any{}).(map[string]any)
	commands, ok := out["suggested_retrieval_commands"].([]string)
	if !ok || len(commands) == 0 {
		t.Fatalf("expected suggested retrieval commands, got %#v", out["suggested_retrieval_commands"])
	}
	if !strings.Contains(commands[0], "epo pub claims") {
		t.Fatalf("unexpected suggestion command: %s", commands[0])
	}
}

func TestStripMixedLayoutNodes(t *testing.T) {
	input := map[string]any{
		"reg:event-data": map[string]any{
			"mixed.layout": []any{"one", "two"},
			"kept":         true,
		},
	}
	out := stripMixedLayoutNodes(input).(map[string]any)
	eventData := asAnyMap(out["reg:event-data"])
	if _, exists := eventData["mixed.layout"]; exists {
		t.Fatalf("mixed.layout should be stripped: %#v", eventData)
	}
}

func TestDetectNumberFormat(t *testing.T) {
	if got := detectNumberFormat("EP.1000000.A1"); got != "docdb" {
		t.Fatalf("unexpected format: %s", got)
	}
	if got := detectNumberFormat("EP1000000A1"); got != "epodoc" {
		t.Fatalf("unexpected format: %s", got)
	}
	if got := detectNumberFormat("US.(08/921,321).A.19970829"); got != "original" {
		t.Fatalf("unexpected format: %s", got)
	}
}

func TestFlattenLegalEvents(t *testing.T) {
	input := map[string]any{
		"events": []any{
			map[string]any{
				"L001EP": "CODE",
				"L002EP": "Description",
				"L003EP": "DE",
				"L007EP": "20260101",
			},
		},
	}
	rows := flattenLegalEvents(input)
	if len(rows) != 1 {
		t.Fatalf("expected one legal row, got %d", len(rows))
	}
	if rows[0]["code"] != "CODE" || rows[0]["country"] != "DE" {
		t.Fatalf("unexpected legal row: %#v", rows[0])
	}
}

func TestSummarizeRegisterPayload(t *testing.T) {
	input := map[string]any{
		"ops:world-patent-data": map[string]any{
			"ops:register-search": map[string]any{
				"reg:register-documents": map[string]any{
					"reg:register-document": map[string]any{
						"reg:bibliographic-data": map[string]any{
							"reg:application-reference": map[string]any{
								"reg:document-id": map[string]any{
									"reg:country":    map[string]any{"$": "EP"},
									"reg:doc-number": map[string]any{"$": "123456"},
								},
							},
							"reg:publication-reference": map[string]any{
								"reg:document-id": map[string]any{
									"reg:country":    map[string]any{"$": "WO"},
									"reg:doc-number": map[string]any{"$": "20260001"},
									"reg:date":       map[string]any{"$": "20260101"},
								},
							},
						},
						"reg:ep-patent-statuses": map[string]any{
							"reg:ep-patent-status": map[string]any{"$": "Pending"},
						},
					},
				},
			},
			"reg:designated-state":  "DE",
			"reg:lapsed-in-country": "FR",
		},
	}
	summary := summarizeRegisterPayload(input)
	if summary["status"] != "Pending" {
		t.Fatalf("unexpected register status: %#v", summary)
	}
	if summary["application"] != "EP123456" {
		t.Fatalf("unexpected application ref: %#v", summary)
	}
}

func TestExtractUsageRowsWithMetrics(t *testing.T) {
	input := map[string]any{
		"environments": []any{
			map[string]any{
				"name": "prod",
				"dimensions": []any{
					map[string]any{
						"metrics": []any{
							map[string]any{
								"name": "message_count",
								"points": []any{
									map[string]any{"date": "20260304", "value": 50},
								},
							},
							map[string]any{
								"name": "total_response_size",
								"points": []any{
									map[string]any{"date": "20260304", "value": 4096},
								},
							},
						},
					},
				},
			},
		},
	}

	rows, ok := extractUsageRows(input)
	if !ok || len(rows) != 1 {
		t.Fatalf("expected flattened usage rows, got %#v", rows)
	}
	if rows[0]["message_count"] != "50" || rows[0]["total_response_size"] != "4096" {
		t.Fatalf("unexpected usage metrics row: %#v", rows[0])
	}
}

func TestWithUsageHumanDates(t *testing.T) {
	input := map[string]any{"date": "20260304"}
	out := withUsageHumanDates(input).(map[string]any)
	if out["date_human"] == "" {
		t.Fatalf("expected human date enrichment: %#v", out)
	}
}

func TestNormalizeCPCPayload(t *testing.T) {
	searchXML := []byte(`
<root>
  <classification-symbol>H04L45/00</classification-symbol>
  <class-title>Routing</class-title>
  <score>15.23</score>
</root>
`)
	searchRows := normalizeCPCPayload("search", "", "", "", searchXML)
	if len(searchRows) != 1 {
		t.Fatalf("expected one search row, got %d", len(searchRows))
	}
	if searchRows[0]["symbol"] != "H04L45/00" {
		t.Fatalf("unexpected search row: %#v", searchRows[0])
	}

	mapXML := []byte(`<root><classification-symbol>H04L45/00</classification-symbol><classification-symbol>H04L45/10</classification-symbol></root>`)
	mapRows := normalizeCPCPayload("map", "H04L45/00", "cpc", "ipc", mapXML)
	if len(mapRows) == 0 {
		t.Fatal("expected mapping rows")
	}
	if mapRows[0]["fromScheme"] != "CPC" || mapRows[0]["toScheme"] != "IPC" {
		t.Fatalf("unexpected map row: %#v", mapRows[0])
	}
}
