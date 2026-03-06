package eps

import (
	"reflect"
	"testing"
)

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
