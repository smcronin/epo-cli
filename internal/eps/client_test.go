package eps

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
)

func TestDoMarksForbiddenAsEPSError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<html>Forbidden</html>`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	client.SetBaseURL(srv.URL)

	_, err := client.Do(context.Background(), http.MethodGet, "/publication-dates", nil, "text/html")

	var apiErr *epoerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected an API error, got %v", err)
	}
	if apiErr.StatusCode != http.StatusForbidden || apiErr.Service != epoerrors.ServiceEPS {
		t.Fatalf("expected a 403 marked as EPS, got %d / %q", apiErr.StatusCode, apiErr.Service)
	}
}

func TestParsePublicationDates(t *testing.T) {
	t.Parallel()

	body := []byte(`
<html>
<body>
<a href="https://data.epo.org/publication-server/rest/v1.2/publication-dates/20130904/patents">2013/09/04</a>
<a href="https://data.epo.org/publication-server/rest/v1.2/publication-dates/20130911/patents">2013/09/11</a>
<a href="https://data.epo.org/publication-server/rest/v1.2/publication-dates/20130904/patents">dup</a>
</body>
</html>
`)

	got := ParsePublicationDates(body)
	want := []string{"20130904", "20130911"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected dates: got=%v want=%v", got, want)
	}
}

func TestParsePatentIDs(t *testing.T) {
	t.Parallel()

	body := []byte(`
<html>
<body>
<a href="https://data.epo.org/publication-server/rest/v1.2/publication-dates/20130904/patents/EP1004359NWB1">EP1004359NWB1</a>
<a href="https://data.epo.org/publication-server/rest/v1.2/publication-dates/20130904/patents/EP1026129NWB1">EP1026129NWB1</a>
<a href="https://data.epo.org/publication-server/rest/v1.2/publication-dates/20130904/patents/EP1026129NWB1">dup</a>
</body>
</html>
`)

	got := ParsePatentIDs(body)
	want := []string{"EP1004359NWB1", "EP1026129NWB1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected patents: got=%v want=%v", got, want)
	}
}

func TestParseDocumentFormats(t *testing.T) {
	t.Parallel()

	body := []byte(`
<html>
<body>
<a href="https://data.epo.org/publication-server/rest/v1.2/patents/EP1004359NWB1/document.xml">XML</a>
<a href="https://data.epo.org/publication-server/rest/v1.2/patents/EP1004359NWB1/document.html">HTML</a>
<a href="https://data.epo.org/publication-server/rest/v1.2/patents/EP1004359NWB1/document.pdf">PDF</a>
<a href="https://data.epo.org/publication-server/rest/v1.2/patents/EP1004359NWB1/document.zip">ZIP</a>
</body>
</html>
`)

	got := ParseDocumentFormats(body)
	want := []DocumentFormatLink{
		{Format: "html", URL: "https://data.epo.org/publication-server/rest/v1.2/patents/EP1004359NWB1/document.html"},
		{Format: "pdf", URL: "https://data.epo.org/publication-server/rest/v1.2/patents/EP1004359NWB1/document.pdf"},
		{Format: "xml", URL: "https://data.epo.org/publication-server/rest/v1.2/patents/EP1004359NWB1/document.xml"},
		{Format: "zip", URL: "https://data.epo.org/publication-server/rest/v1.2/patents/EP1004359NWB1/document.zip"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected formats: got=%v want=%v", got, want)
	}
}
