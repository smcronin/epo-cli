package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAttachKindCodeChecksSuffixNotSubstring(t *testing.T) {
	t.Parallel()

	cases := []struct {
		reference, inputFormat, kind, want string
	}{
		// Country codes that contain the kind letter must still get the kind.
		{"CA2000000", "epodoc", "C", "CA2000000C"},
		{"AT123456", "epodoc", "T", "AT123456T"},
		{"AU2020200001", "epodoc", "A", "AU2020200001A"},
		{"EP.1000000", "docdb", "B1", "EP.1000000.B1"},
		// Already carries the kind: leave it alone.
		{"EP1000000A1", "epodoc", "A1", "EP1000000A1"},
		{"EP.1000000.A1", "docdb", "a1", "EP.1000000.A1"},
		{"EP1000000", "epodoc", "", "EP1000000"},
	}
	for _, tc := range cases {
		if got := attachKindCode(tc.reference, tc.inputFormat, tc.kind); got != tc.want {
			t.Fatalf("attachKindCode(%q, %q, %q) = %q, want %q", tc.reference, tc.inputFormat, tc.kind, got, tc.want)
		}
	}
}

func TestLooksApplicationEpRef(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"EP99203729":    true,  // application number
		"ep19990203729": true,  // long-form application number
		"EP1000000":     false, // kindless publication number
		"EP0536425":     false,
		"EP1000000B1":   false,
		"EP.1000000.A1": false,
	}
	for reference, want := range tests {
		if got := looksApplicationEpRef(reference); got != want {
			t.Fatalf("looksApplicationEpRef(%q) = %v, want %v", reference, got, want)
		}
	}
}

func TestRegisterUPPRoutesKindlessPublicationNumber(t *testing.T) {
	tests := map[string]string{
		"EP1000000":   "/rest-services/register/publication/epodoc/EP1000000/upp",
		"EP1000000B1": "/rest-services/register/publication/epodoc/EP1000000B1/upp",
		"EP99203729":  "/rest-services/register/application/epodoc/EP99203729/upp",
	}
	for reference, wantPath := range tests {
		stub := installOPSStub(t, func(req *http.Request) (int, http.Header, string) {
			return http.StatusOK, nil, `{}`
		})
		if _, err := runForTest(t, newRegisterCmd(), "upp", reference); err != nil {
			t.Fatalf("register upp %s: %v", reference, err)
		}
		requests := stub.dataRequests()
		if len(requests) != 1 || requests[0].URL.Path != wantPath {
			t.Fatalf("register upp %s requested %v, want %s", reference, requests, wantPath)
		}
	}
}

func TestStatusDoesNotUsePublicationNumberAsRegisterRef(t *testing.T) {
	stub := installOPSStub(t, func(req *http.Request) (int, http.Header, string) {
		return http.StatusOK, nil, `{}`
	})

	out, err := runForTest(t, newStatusCmd(), "EP1000000")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, req := range stub.dataRequests() {
		if strings.Contains(req.URL.Path, "/register/") {
			t.Fatalf("publication number was sent to the register application endpoint: %s", req.URL.Path)
		}
	}
	var env struct {
		Results struct {
			RegisterRef string `json:"registerRef"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if env.Results.RegisterRef != "" {
		t.Fatalf("expected an unresolved register reference, got %q", env.Results.RegisterRef)
	}
}

// registerDocumentWithArrays mirrors the register sample in the OPS reference
// guide: a granted EP patent carries one reg:publication-reference per
// publication (A1 and B1) and a list of status changes.
func registerDocumentWithArrays() map[string]any {
	publication := func(kind, date string) map[string]any {
		return map[string]any{
			"reg:document-id": map[string]any{
				"reg:country":    map[string]any{"$": "EP"},
				"reg:doc-number": map[string]any{"$": "1000000"},
				"reg:kind":       map[string]any{"$": kind},
				"reg:date":       map[string]any{"$": date},
			},
		}
	}
	return map[string]any{
		"ops:world-patent-data": map[string]any{
			"ops:register-search": map[string]any{
				"reg:register-documents": map[string]any{
					"reg:register-document": map[string]any{
						"reg:bibliographic-data": map[string]any{
							"reg:application-reference": map[string]any{
								"reg:document-id": map[string]any{
									"reg:country":    map[string]any{"$": "EP"},
									"reg:doc-number": map[string]any{"$": "99203729"},
								},
							},
							"reg:publication-reference": []any{
								publication("A1", "20000517"),
								publication("B1", "20030212"),
							},
						},
						"reg:ep-patent-statuses": map[string]any{
							"reg:ep-patent-status": []any{
								map[string]any{"$": "The patent has been granted", "@change-date": "20030103"},
								map[string]any{"$": "No opposition filed within time limit", "@change-date": "20031219"},
							},
						},
					},
				},
			},
		},
	}
}

func TestRegisterRowsReadArrayShapedFields(t *testing.T) {
	t.Parallel()

	rows, ok := normalizeRows(registerDocumentWithArrays())
	if !ok || len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row["pubCountry"] != "EP" || row["pubDocNumber"] != "1000000" || row["pubDate"] != "20000517" {
		t.Fatalf("publication fields dropped for array-shaped reg:publication-reference: %#v", row)
	}
	if row["epPatentStatus"] != "No opposition filed within time limit" {
		t.Fatalf("status dropped for array-shaped reg:ep-patent-status: %#v", row["epPatentStatus"])
	}
}

func TestSummarizeRegisterPayloadReadsStatusArray(t *testing.T) {
	t.Parallel()

	summary := summarizeRegisterPayload(registerDocumentWithArrays())
	if summary["status"] != "No opposition filed within time limit" {
		t.Fatalf("unexpected status: %#v", summary["status"])
	}
	if summary["publication"] != "EP1000000A1" {
		t.Fatalf("unexpected publication: %#v", summary["publication"])
	}
}
