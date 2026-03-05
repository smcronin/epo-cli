package cli

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestFilterEPSDatesDescWithLimit(t *testing.T) {
	t.Parallel()

	got, err := filterEPSDates(
		[]string{"20240214", "20240221", "20240228", "20240306"},
		"20240221",
		"20240306",
		"desc",
		2,
	)
	if err != nil {
		t.Fatalf("filterEPSDates error: %v", err)
	}
	want := []string{"20240306", "20240228"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected dates: got=%v want=%v", got, want)
	}
}

func TestFilterEPSDatesInvalidOrder(t *testing.T) {
	t.Parallel()

	if _, err := filterEPSDates([]string{"20240214"}, "", "", "sideways", 0); err == nil {
		t.Fatal("expected invalid order error")
	}
}

func TestValidateEPSFormat(t *testing.T) {
	t.Parallel()

	if _, err := validateEPSFormat("tar"); err == nil {
		t.Fatal("expected invalid format error")
	}
	got, err := validateEPSFormat("PDF")
	if err != nil {
		t.Fatalf("validateEPSFormat returned error: %v", err)
	}
	if got != "pdf" {
		t.Fatalf("unexpected format: %s", got)
	}
}

func TestEPSTargetPath(t *testing.T) {
	t.Parallel()

	got := epsTargetPath(".tmp/eps-bulk", "zip", "20240306", "EP12345NWA1")
	want := filepath.Join(".tmp/eps-bulk", "documents", "20240306", "EP12345NWA1.zip")
	if got != want {
		t.Fatalf("unexpected target path: got=%s want=%s", got, want)
	}
}
