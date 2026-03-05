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
	if got := resolvePubInputFormat("EP1000000A1", "auto"); got != "docdb" {
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
			"mixed.layout":  []any{"one", "two"},
			"@mixed.layout": []any{"three"},
			"kept":          true,
		},
	}
	out := stripMixedLayoutNodes(input).(map[string]any)
	eventData := asAnyMap(out["reg:event-data"])
	if _, exists := eventData["mixed.layout"]; exists {
		t.Fatalf("mixed.layout should be stripped: %#v", eventData)
	}
	if _, exists := eventData["@mixed.layout"]; exists {
		t.Fatalf("@mixed.layout should be stripped: %#v", eventData)
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
				"@code":      "AK",
				"@desc":      "DESIGNATED CONTRACTING STATES",
				"@infl":      "+",
				"ops:L001EP": map[string]any{"$": "EP"},
				"ops:L007EP": map[string]any{"$": "2003-02-12"},
				"ops:L507EP": map[string]any{"$": "AT BE CH"},
			},
		},
	}
	rows := flattenLegalEvents(input)
	if len(rows) != 1 {
		t.Fatalf("expected one legal row, got %d", len(rows))
	}
	if rows[0]["code"] != "AK" || rows[0]["country"] != "EP" {
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
									"reg:date":       map[string]any{"$": "19990101"},
								},
							},
							"reg:publication-reference": map[string]any{
								"reg:document-id": map[string]any{
									"reg:country":    map[string]any{"$": "EP"},
									"reg:doc-number": map[string]any{"$": "20260001"},
									"reg:date":       map[string]any{"$": "20260101"},
									"reg:kind":       map[string]any{"$": "A1"},
								},
							},
							"reg:designation-of-states": map[string]any{
								"reg:designation-pct": map[string]any{
									"reg:regional": map[string]any{
										"reg:country": []any{
											map[string]any{"$": "DE"},
											map[string]any{"$": "FR"},
										},
									},
								},
							},
							"reg:term-of-grant": []any{
								map[string]any{
									"reg:lapsed-in-country": []any{
										map[string]any{"reg:country": map[string]any{"$": "FR"}},
										map[string]any{"reg:country": map[string]any{"$": "IT"}},
									},
								},
							},
						},
						"reg:ep-patent-statuses": map[string]any{
							"reg:ep-patent-status": map[string]any{"$": "Pending"},
						},
					},
				},
			},
		},
	}
	summary := summarizeRegisterPayload(input)
	if summary["status"] != "Pending" {
		t.Fatalf("unexpected register status: %#v", summary)
	}
	if summary["application"] != "EP123456" {
		t.Fatalf("unexpected application ref: %#v", summary)
	}
	if summary["publication"] != "EP20260001A1" {
		t.Fatalf("unexpected publication ref: %#v", summary)
	}
	if len(summary["designatedStates"].([]string)) != 2 {
		t.Fatalf("unexpected designated states: %#v", summary["designatedStates"])
	}
	if len(summary["lapseList"].([]string)) != 2 {
		t.Fatalf("unexpected lapse list: %#v", summary["lapseList"])
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

func TestExtractUsageRowsWithValuesTimestamp(t *testing.T) {
	input := map[string]any{
		"environments": []any{
			map[string]any{
				"name": "prod",
				"dimensions": []any{
					map[string]any{
						"metrics": []any{
							map[string]any{
								"name": "message_count",
								"values": []any{
									map[string]any{"timestamp": int64(1772582400000), "value": "270"},
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
	if rows[0]["date"] != "2026-03-04" {
		t.Fatalf("unexpected derived date: %#v", rows[0]["date"])
	}
	if rows[0]["message_count"] != "270" {
		t.Fatalf("unexpected metric value: %#v", rows[0])
	}
}

func TestWithUsageHumanDates(t *testing.T) {
	input := map[string]any{"timestamp": int64(1772582400000), "date": "20260304"}
	out := withUsageHumanDates(input).(map[string]any)
	if out["date_human"] == "" {
		t.Fatalf("expected human date enrichment: %#v", out)
	}
	if out["timestamp_human"] != "2026-03-04" {
		t.Fatalf("expected ms timestamp conversion: %#v", out)
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
	searchRows := normalizeCPCPayload("search", "", "", "", nil, searchXML)
	if len(searchRows) != 1 {
		t.Fatalf("expected one search row, got %d", len(searchRows))
	}
	if searchRows[0]["symbol"] != "H04L45/00" {
		t.Fatalf("unexpected search row: %#v", searchRows[0])
	}

	mapXML := []byte(`<root><classification-symbol>H04L45/00</classification-symbol><classification-symbol>H04L45/10</classification-symbol></root>`)
	mapRows := normalizeCPCPayload("map", "H04L45/00", "cpc", "ipc", nil, mapXML)
	if len(mapRows) == 0 {
		t.Fatal("expected mapping rows")
	}
	if mapRows[0]["fromScheme"] != "CPC" || mapRows[0]["toScheme"] != "IPC" {
		t.Fatalf("unexpected map row: %#v", mapRows[0])
	}
}

func TestNormalizeCPCPayloadFromParsedSearch(t *testing.T) {
	parsed := map[string]any{
		"world-patent-data": map[string]any{
			"classification-search": map[string]any{
				"search-result": map[string]any{
					"classification-statistics": []any{
						map[string]any{
							"@classification-symbol": "H04L45/00",
							"@percentage":            "14.957265",
							"class-title": map[string]any{
								"title-part": map[string]any{
									"text": map[string]any{"$": "Routing"},
								},
							},
						},
					},
				},
			},
		},
	}
	rows := normalizeCPCPayload("search", "", "", "", parsed, nil)
	if len(rows) != 1 || rows[0]["symbol"] != "H04L45/00" {
		t.Fatalf("unexpected parsed search rows: %#v", rows)
	}
}

func TestCollectProceduralStepLabelsStructured(t *testing.T) {
	input := map[string]any{
		"reg:procedural-data": map[string]any{
			"reg:procedural-step": []any{
				map[string]any{
					"reg:procedural-step-code": map[string]any{"$": "RFEE"},
					"reg:procedural-step-text": map[string]any{
						"$":               "Renewal fee payment",
						"@step-text-type": "STEP_DESCRIPTION",
					},
					"reg:procedural-step-date": map[string]any{
						"reg:date": map[string]any{"$": "20011128"},
					},
				},
			},
		},
	}
	rows := collectProceduralStepLabels(input)
	if len(rows) != 1 {
		t.Fatalf("unexpected procedural rows: %#v", rows)
	}
	if rows[0]["code"] != "RFEE" || rows[0]["description"] != "Renewal fee payment" {
		t.Fatalf("unexpected procedural row: %#v", rows[0])
	}
}

func TestDedupeFlatPublicationRows(t *testing.T) {
	rows := []map[string]any{
		{"reference": "EP4703890A1", "title": "X", "pubDate": "20260304"},
		{"reference": "EP4703890A1", "title": "X", "pubDate": "20260304"},
	}
	got := dedupeFlatPublicationRows(rows)
	if len(got) != 1 {
		t.Fatalf("expected dedupe to keep one row, got %#v", got)
	}
}
