package cli

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "update golden files")

func TestMethodsCatalogGolden(t *testing.T) {
	t.Parallel()

	got, err := json.MarshalIndent(buildMethodCatalog(), "", "  ")
	if err != nil {
		t.Fatalf("marshal methods catalog: %v", err)
	}
	assertGolden(t, "methods_catalog.golden.json", append(got, '\n'))
}

func TestPubSearchRowsGolden(t *testing.T) {
	t.Parallel()

	assertRowsGolden(t, "pub_search_input.json", "pub_search_rows.golden.json")
}

func TestFamilyRowsGolden(t *testing.T) {
	t.Parallel()
	assertRowsGolden(t, "family_get_input.json", "family_get_rows.golden.json")
}

func TestNumberRowsGolden(t *testing.T) {
	t.Parallel()
	assertRowsGolden(t, "number_convert_input.json", "number_convert_rows.golden.json")
}

func TestRegisterRowsGolden(t *testing.T) {
	t.Parallel()
	assertRowsGolden(t, "register_search_input.json", "register_search_rows.golden.json")
}

func TestUsageRowsGolden(t *testing.T) {
	t.Parallel()
	assertRowsGolden(t, "usage_stats_input.json", "usage_stats_rows.golden.json")
}

func assertRowsGolden(t *testing.T, inputFile, goldenFile string) {
	t.Helper()

	inputPath := filepath.Join("testdata", inputFile)
	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input fixture: %v", err)
	}

	var input map[string]any
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		t.Fatalf("parse input fixture: %v", err)
	}

	rows, ok := normalizeRows(input)
	if !ok {
		t.Fatal("expected extracted rows")
	}
	got, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatalf("marshal rows: %v", err)
	}
	assertGolden(t, goldenFile, append(got, '\n'))
}

func assertGolden(t *testing.T, fileName string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", fileName)
	if *updateGolden {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if string(want) != string(got) {
		t.Fatalf("golden mismatch for %s", fileName)
	}
}
