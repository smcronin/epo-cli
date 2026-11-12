package cli

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExpectedArchiveNames(t *testing.T) {
	got := expectedArchiveNames("v1.2.3", "windows", "amd64")
	if len(got) < 2 {
		t.Fatalf("expected multiple archive names, got %d", len(got))
	}
	if got[0] != "epo_windows_amd64.zip" {
		t.Fatalf("unexpected first archive candidate: %s", got[0])
	}
	if got[1] != "epo_1.2.3_windows_amd64.zip" {
		t.Fatalf("unexpected second archive candidate: %s", got[1])
	}
}

func TestFindReleaseAssetByName(t *testing.T) {
	assets := []githubReleaseAsset{
		{Name: "checksums.txt", BrowserDownloadURL: "https://example/checksums.txt"},
		{Name: "epo_windows_amd64.zip", BrowserDownloadURL: "https://example/win.zip"},
	}

	got, ok := findReleaseAssetByName(assets, "checksums.txt")
	if !ok {
		t.Fatal("expected checksums asset to be found")
	}
	if got.BrowserDownloadURL != "https://example/checksums.txt" {
		t.Fatalf("unexpected URL: %s", got.BrowserDownloadURL)
	}

	_, ok = findReleaseAssetByName(assets, "missing")
	if ok {
		t.Fatal("did not expect missing asset to be found")
	}
}

func TestFindReleaseAssetByNames(t *testing.T) {
	assets := []githubReleaseAsset{
		{Name: "epo-cli_1.2.3_windows_amd64.zip", BrowserDownloadURL: "https://example/win.zip"},
	}
	got, name, ok := findReleaseAssetByNames(assets, []string{
		"epo_windows_amd64.zip",
		"epo-cli_1.2.3_windows_amd64.zip",
	})
	if !ok {
		t.Fatal("expected legacy asset fallback")
	}
	if name != "epo-cli_1.2.3_windows_amd64.zip" {
		t.Fatalf("unexpected matched name: %s", name)
	}
	if got.BrowserDownloadURL != "https://example/win.zip" {
		t.Fatalf("unexpected URL: %s", got.BrowserDownloadURL)
	}
}

func TestLookupChecksum(t *testing.T) {
	checksums := `
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  epo_linux_amd64.tar.gz
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb *epo_windows_amd64.zip
`
	got, ok := lookupChecksum(checksums, "epo_windows_amd64.zip")
	if !ok {
		t.Fatal("expected checksum to be found")
	}
	if got != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("unexpected checksum: %s", got)
	}

	_, ok = lookupChecksum(checksums, "missing")
	if ok {
		t.Fatal("did not expect missing checksum entry")
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "v0.1.2", want: "0.1.2"},
		{in: "0.1.2", want: "0.1.2"},
		{in: " V1.2.3 ", want: "1.2.3"},
		{in: "dev", want: "dev"},
	}

	for _, tc := range tests {
		got := normalizeVersion(tc.in)
		if got != tc.want {
			t.Fatalf("normalizeVersion(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTargetExecutablePath(t *testing.T) {
	dir := t.TempDir()
	legacyName := "epo-cli"
	newName := "epo"
	if runtime.GOOS == "windows" {
		legacyName += ".exe"
		newName += ".exe"
	}
	legacyPath := filepath.Join(dir, legacyName)
	newPath := filepath.Join(dir, newName)

	got := targetExecutablePath(legacyPath)
	if got != newPath {
		t.Fatalf("targetExecutablePath(%q)=%q, want %q", legacyPath, got, newPath)
	}

	got = targetExecutablePath(newPath)
	if got != newPath {
		t.Fatalf("targetExecutablePath(%q)=%q, want %q", newPath, got, newPath)
	}
}

// writeTestArchive builds a release-style archive (zip or tar.gz, chosen by
// the file name) holding a single entry.
func writeTestArchive(t *testing.T, archivePath, entryName string, payload []byte) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer file.Close()

	if strings.HasSuffix(archivePath, ".zip") {
		zw := zip.NewWriter(file)
		entry, err := zw.Create(entryName)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := entry.Write(payload); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("close zip: %v", err)
		}
		return
	}

	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: entryName, Mode: 0o755, Size: int64(len(payload)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatalf("write tar entry: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
}

func TestExtractBinaryFromReleaseArchive(t *testing.T) {
	t.Parallel()

	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}
	// .github/workflows/release.yml archives the binary as epo_<os>_<arch>.
	releaseEntry := fmt.Sprintf("epo_%s_%s%s", runtime.GOOS, runtime.GOARCH, exe)
	payload := []byte("fake epo binary")

	cases := []struct {
		name, archive, entry string
	}{
		{"release zip", "epo_test.zip", releaseEntry},
		{"release tar.gz", "epo_test.tar.gz", releaseEntry},
		{"plain name zip", "epo_plain.zip", "epo" + exe},
		{"plain name tar.gz", "epo_plain.tar.gz", "epo" + exe},
		{"legacy name tar.gz", "epo_legacy.tar.gz", "epo-cli" + exe},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			archivePath := filepath.Join(dir, tc.archive)
			writeTestArchive(t, archivePath, tc.entry, payload)

			binPath, err := extractBinaryFromArchive(archivePath, dir)
			if err != nil {
				t.Fatalf("extract %s from %s: %v", tc.entry, tc.archive, err)
			}
			got, err := os.ReadFile(binPath)
			if err != nil {
				t.Fatalf("read extracted binary: %v", err)
			}
			if string(got) != string(payload) {
				t.Fatalf("extracted content mismatch: %q", got)
			}
		})
	}

	t.Run("unrelated entry is rejected", func(t *testing.T) {
		dir := t.TempDir()
		archivePath := filepath.Join(dir, "other.zip")
		writeTestArchive(t, archivePath, "README.md", payload)
		if _, err := extractBinaryFromArchive(archivePath, dir); err == nil {
			t.Fatal("expected an error for an archive without the epo binary")
		}
	})
}
