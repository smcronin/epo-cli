package cli

import "testing"

func TestProjectByFields(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"alpha": 1,
		"nested": map[string]any{
			"beta": "x",
		},
	}
	got := projectByFields(input, []string{"alpha", "nested.beta"})
	row, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", got)
	}
	if row["alpha"] != 1 {
		t.Fatalf("alpha = %v, want 1", row["alpha"])
	}
	if row["nested.beta"] != "x" {
		t.Fatalf("nested.beta = %v, want x", row["nested.beta"])
	}
}

func TestParseRangeWindow(t *testing.T) {
	t.Parallel()

	start, end, err := parseRangeWindow("10-25")
	if err != nil {
		t.Fatalf("parseRangeWindow error: %v", err)
	}
	if start != 10 || end != 25 {
		t.Fatalf("got %d-%d, want 10-25", start, end)
	}

	_, _, err = parseRangeWindow("bad")
	if err == nil {
		t.Fatal("expected range parse error")
	}
}

func TestExtractPublishedSearchItems(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"ops:world-patent-data": map[string]any{
			"ops:biblio-search": map[string]any{
				"ops:search-result": map[string]any{
					"ops:publication-reference": []any{
						map[string]any{"id": "1"},
						map[string]any{"id": "2"},
					},
				},
			},
		},
	}
	items, ok := extractPublishedSearchItems(input)
	if !ok {
		t.Fatal("expected items")
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}

func TestExtractRegisterSearchItems(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"ops:world-patent-data": map[string]any{
			"ops:register-search": map[string]any{
				"reg:register-documents": map[string]any{
					"reg:register-document": map[string]any{"id": "one"},
				},
			},
		},
	}
	items, ok := extractRegisterSearchItems(input)
	if !ok {
		t.Fatal("expected items")
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}
