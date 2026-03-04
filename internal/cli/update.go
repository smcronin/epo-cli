package cli

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	updateGitHubOwner = "smcronin"
	updateGitHubRepo  = "epo-cli"

	updateArchivePrefix       = "epo"
	updateLegacyArchivePrefix = "epo-cli"
	updateUserAgent           = "epo-update"
)

var (
	updateCheckFlag   bool
	updateForceFlag   bool
	updateVersionFlag string
	updateDryRunFlag  bool
)

type githubRelease struct {
	TagName string               `json:"tag_name"`
	Name    string               `json:"name"`
	Assets  []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"self-update"},
		Short:   "Update epo from GitHub Releases",
		Long: `Update epo from GitHub Releases.

By default, this fetches the latest release for your OS/arch, verifies the
checksum, and replaces the current executable.

Use --check to only show current/latest versions without installing.`,
		RunE: runUpdate,
	}

	cmd.Flags().BoolVar(&updateCheckFlag, "check", false, "Check latest version without installing")
	cmd.Flags().BoolVar(&updateForceFlag, "force", false, "Install even when already on target version")
	cmd.Flags().StringVar(&updateVersionFlag, "version", "", "Target version tag (e.g. v0.1.2); default is latest")
	cmd.Flags().BoolVar(&updateDryRunFlag, "dry-run", false, "Download and verify update without replacing executable")
	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), updateNetworkTimeout())
	defer cancel()

	release, err := fetchGitHubRelease(ctx, updateVersionFlag)
	if err != nil {
		return err
	}

	currentVersion := normalizeVersion(version)
	targetVersion := normalizeVersion(release.TagName)
	upToDate := currentVersion != "" && currentVersion == targetVersion

	assetNames := expectedArchiveNames(release.TagName, runtime.GOOS, runtime.GOARCH)
	archiveAsset, assetName, ok := findReleaseAssetByNames(release.Assets, assetNames)
	if !ok {
		return fmt.Errorf(
			"release %s does not include expected assets (%s) for %s/%s",
			release.TagName,
			strings.Join(assetNames, ", "),
			runtime.GOOS,
			runtime.GOARCH,
		)
	}

	checksumAsset, hasChecksums := findReleaseAssetByName(release.Assets, "checksums.txt")
	execPath, _ := currentExecutablePath()
	installPath := targetExecutablePath(execPath)
	migratedFrom := ""
	if execPath != "" && installPath != execPath {
		migratedFrom = execPath
	}

	if updateCheckFlag || (upToDate && !updateForceFlag && !updateDryRunFlag) {
		result := map[string]any{
			"currentVersion": currentVersionOrUnknown(currentVersion),
			"latestVersion":  targetVersion,
			"upToDate":       upToDate,
			"os":             runtime.GOOS,
			"arch":           runtime.GOARCH,
			"asset":          assetName,
			"executable":     execPath,
			"installPath":    installPath,
		}
		if migratedFrom != "" {
			result["migratedFrom"] = migratedFrom
		}
		return outputSuccess(cmd, result)
	}

	if execPath == "" {
		return fmt.Errorf("could not determine executable path")
	}

	tmpDir, err := os.MkdirTemp("", "epo-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, assetName)
	updateProgress(fmt.Sprintf("Downloading %s...", assetName))
	if err := downloadToFile(ctx, archiveAsset.BrowserDownloadURL, archivePath); err != nil {
		return err
	}

	if hasChecksums {
		updateProgress("Verifying checksum...")
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := downloadToFile(ctx, checksumAsset.BrowserDownloadURL, checksumPath); err != nil {
			return fmt.Errorf("downloading checksums: %w", err)
		}
		if err := verifyFileChecksum(checksumPath, assetName, archivePath); err != nil {
			return err
		}
	}

	updateProgress("Extracting archive...")
	newBinPath, err := extractBinaryFromArchive(archivePath, tmpDir)
	if err != nil {
		return err
	}

	if updateDryRunFlag {
		result := map[string]any{
			"currentVersion":  currentVersionOrUnknown(currentVersion),
			"targetVersion":   targetVersion,
			"asset":           assetName,
			"executable":      execPath,
			"installPath":     installPath,
			"downloadedTo":    archivePath,
			"extractedBinary": newBinPath,
			"dryRun":          true,
		}
		if migratedFrom != "" {
			result["migratedFrom"] = migratedFrom
		}
		return outputSuccess(cmd, result)
	}

	scheduled := false
	legacyRemoved := false
	legacyRemoveWarning := ""
	if runtime.GOOS == "windows" {
		updateProgress("Scheduling Windows binary install...")
		if err := scheduleWindowsBinarySwap(newBinPath, installPath, migratedFrom); err != nil {
			return err
		}
		scheduled = true
	} else {
		updateProgress("Replacing executable...")
		if err := replaceExecutableNow(newBinPath, installPath); err != nil {
			return err
		}
		if migratedFrom != "" {
			if err := os.Remove(migratedFrom); err == nil || os.IsNotExist(err) {
				legacyRemoved = true
			} else {
				legacyRemoveWarning = err.Error()
			}
		}
	}

	result := map[string]any{
		"currentVersion": currentVersionOrUnknown(currentVersion),
		"targetVersion":  targetVersion,
		"asset":          assetName,
		"executable":     execPath,
		"installPath":    installPath,
		"updated":        !scheduled,
		"scheduled":      scheduled,
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
	}
	if migratedFrom != "" {
		result["migratedFrom"] = migratedFrom
	}
	if legacyRemoved {
		result["legacyRemoved"] = true
	}
	if legacyRemoveWarning != "" {
		result["legacyRemoveWarning"] = legacyRemoveWarning
	}
	return outputSuccess(cmd, result)
}

func updateProgress(message string) {
	if flagQuiet {
		return
	}
	fmt.Fprintln(os.Stderr, message)
}

func updateNetworkTimeout() time.Duration {
	seconds := flagTimeout
	if seconds < 120 {
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}

func fetchGitHubRelease(ctx context.Context, requestedTag string) (*githubRelease, error) {
	baseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", updateGitHubOwner, updateGitHubRepo)
	endpoint := baseURL + "/latest"
	if strings.TrimSpace(requestedTag) != "" {
		tag := strings.TrimSpace(requestedTag)
		if !strings.HasPrefix(strings.ToLower(tag), "v") {
			tag = "v" + tag
		}
		endpoint = baseURL + "/tags/" + url.PathEscape(tag)
	}

	body, err := httpGet(ctx, endpoint)
	if err != nil {
		if strings.Contains(err.Error(), "GitHub API error (404)") {
			if strings.TrimSpace(requestedTag) == "" {
				return nil, fmt.Errorf("no GitHub releases found for %s/%s yet", updateGitHubOwner, updateGitHubRepo)
			}
			return nil, fmt.Errorf("release tag %q was not found in %s/%s", requestedTag, updateGitHubOwner, updateGitHubRepo)
		}
		return nil, err
	}

	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, fmt.Errorf("decoding GitHub release response: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("unexpected GitHub release response: missing tag_name")
	}
	return &rel, nil
}

func httpGet(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", updateUserAgent)
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, msg)
	}

	return body, nil
}

func expectedArchiveNames(tag, goos, goarch string) []string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}

	versioned := normalizeVersion(tag)
	candidates := []string{
		fmt.Sprintf("%s_%s_%s%s", updateArchivePrefix, goos, goarch, ext),
		fmt.Sprintf("%s_%s_%s_%s%s", updateArchivePrefix, versioned, goos, goarch, ext),
		fmt.Sprintf("%s_%s_%s%s", updateLegacyArchivePrefix, goos, goarch, ext),
		fmt.Sprintf("%s_%s_%s_%s%s", updateLegacyArchivePrefix, versioned, goos, goarch, ext),
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

func findReleaseAssetByName(assets []githubReleaseAsset, name string) (githubReleaseAsset, bool) {
	for _, asset := range assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return githubReleaseAsset{}, false
}

func findReleaseAssetByNames(assets []githubReleaseAsset, names []string) (githubReleaseAsset, string, bool) {
	for _, name := range names {
		if asset, ok := findReleaseAssetByName(assets, name); ok {
			return asset, name, true
		}
	}
	return githubReleaseAsset{}, "", false
}

func downloadToFile(ctx context.Context, rawURL, outPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", updateUserAgent)
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("download failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	file, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("writing download: %w", err)
	}
	return nil
}

func verifyFileChecksum(checksumsPath, assetName, assetPath string) error {
	checksumData, err := os.ReadFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}

	expected, ok := lookupChecksum(string(checksumData), assetName)
	if !ok {
		return fmt.Errorf("checksums file does not contain %s", assetName)
	}

	got, err := fileSHA256(assetPath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, got) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, got)
	}
	return nil
}

func lookupChecksum(checksums, filename string) (string, bool) {
	for _, line := range strings.Split(checksums, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		hash := fields[0]
		name := strings.TrimPrefix(fields[1], "*")
		if name == filename {
			return hash, true
		}
	}
	return "", false
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file for hash: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hashing file: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func extractBinaryFromArchive(archivePath, outDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractBinaryFromZip(archivePath, outDir)
	}
	return extractBinaryFromTarGz(archivePath, outDir)
}

func extractBinaryFromZip(archivePath, outDir string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("opening zip archive: %w", err)
	}
	defer reader.Close()

	binNames := candidateBinaryNamesForRuntime()
	for _, file := range reader.File {
		if !containsString(binNames, filepath.Base(file.Name)) {
			continue
		}
		in, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("opening zip entry: %w", err)
		}
		defer in.Close()

		outPath := filepath.Join(outDir, binaryNameForRuntime()+".update.bin")
		out, err := os.Create(outPath)
		if err != nil {
			return "", fmt.Errorf("creating extracted binary: %w", err)
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return "", fmt.Errorf("extracting zip binary: %w", err)
		}
		out.Close()
		if err := os.Chmod(outPath, 0o755); err != nil {
			return "", fmt.Errorf("setting executable mode: %w", err)
		}
		return outPath, nil
	}

	return "", fmt.Errorf("binary (%s) not found in %s", strings.Join(binNames, ", "), archivePath)
}

func extractBinaryFromTarGz(archivePath, outDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("opening archive: %w", err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("opening gzip stream: %w", err)
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	binNames := candidateBinaryNamesForRuntime()

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading tar archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if !containsString(binNames, filepath.Base(header.Name)) {
			continue
		}

		outPath := filepath.Join(outDir, binaryNameForRuntime()+".update.bin")
		out, err := os.Create(outPath)
		if err != nil {
			return "", fmt.Errorf("creating extracted binary: %w", err)
		}
		if _, err := io.Copy(out, reader); err != nil {
			out.Close()
			return "", fmt.Errorf("extracting tar binary: %w", err)
		}
		out.Close()
		if err := os.Chmod(outPath, 0o755); err != nil {
			return "", fmt.Errorf("setting executable mode: %w", err)
		}
		return outPath, nil
	}

	return "", fmt.Errorf("binary (%s) not found in %s", strings.Join(binNames, ", "), archivePath)
}

func binaryNameForRuntime() string {
	if runtime.GOOS == "windows" {
		return "epo.exe"
	}
	return "epo"
}

func legacyBinaryNameForRuntime() string {
	if runtime.GOOS == "windows" {
		return "epo-cli.exe"
	}
	return "epo-cli"
}

func candidateBinaryNamesForRuntime() []string {
	primary := binaryNameForRuntime()
	legacy := legacyBinaryNameForRuntime()
	if primary == legacy {
		return []string{primary}
	}
	return []string{primary, legacy}
}

func targetExecutablePath(execPath string) string {
	if execPath == "" {
		return execPath
	}
	if strings.EqualFold(filepath.Base(execPath), legacyBinaryNameForRuntime()) {
		return filepath.Join(filepath.Dir(execPath), binaryNameForRuntime())
	}
	return execPath
}

func replaceExecutableNow(newBinPath, execPath string) error {
	dir := filepath.Dir(execPath)
	tmpDest := filepath.Join(dir, filepath.Base(execPath)+".new")
	if err := copyFile(newBinPath, tmpDest, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpDest, execPath); err != nil {
		return fmt.Errorf("replacing executable: %w", err)
	}
	return nil
}

func scheduleWindowsBinarySwap(newBinPath, installPath, cleanupPath string) error {
	targetNew := installPath + ".new"
	if err := copyFile(newBinPath, targetNew, 0o755); err != nil {
		return err
	}

	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("epo-update-%d.ps1", time.Now().UnixNano()))
	script := windowsSwapScript(scriptPath, targetNew, installPath, cleanupPath, os.Getpid())
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return fmt.Errorf("writing update script: %w", err)
	}

	powershellExe, err := exec.LookPath("powershell")
	if err != nil {
		return fmt.Errorf("powershell not found; cannot self-update on Windows automatically")
	}
	psCommand := exec.Command(powershellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", scriptPath)
	if err := psCommand.Start(); err != nil {
		return fmt.Errorf("launching update script: %w", err)
	}
	return nil
}

func windowsSwapScript(scriptPath, src, dst, cleanup string, pid int) string {
	quote := func(value string) string {
		return strings.ReplaceAll(value, `'`, `''`)
	}
	cleanupLine := ""
	if strings.TrimSpace(cleanup) != "" {
		cleanupLine = fmt.Sprintf("if ($cleanup -ne $dst) { Remove-Item -Path $cleanup -Force -ErrorAction SilentlyContinue }\n")
	}

	return fmt.Sprintf(
		"$pidToWait = %d\n$src = '%s'\n$dst = '%s'\n$cleanup = '%s'\n$self = '%s'\nfor ($i = 0; $i -lt 300; $i++) {\n  if (-not (Get-Process -Id $pidToWait -ErrorAction SilentlyContinue)) { break }\n  Start-Sleep -Milliseconds 200\n}\nfor ($i = 0; $i -lt 50; $i++) {\n  try {\n    Copy-Item -Path $src -Destination $dst -Force\n    break\n  } catch {\n    Start-Sleep -Milliseconds 200\n  }\n}\nRemove-Item -Path $src -Force -ErrorAction SilentlyContinue\n%sRemove-Item -Path $self -Force -ErrorAction SilentlyContinue\n",
		pid, quote(src), quote(dst), quote(cleanup), quote(scriptPath), cleanupLine,
	)
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copying file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("closing destination file: %w", err)
	}
	if err := os.Chmod(dst, mode); err != nil {
		return fmt.Errorf("setting file mode: %w", err)
	}
	return nil
}

func currentExecutablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil && resolved != "" {
		return resolved, nil
	}
	return path, nil
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.ToLower(value), "v")
	return value
}

func currentVersionOrUnknown(value string) string {
	if value == "" || value == "dev" {
		return "dev"
	}
	return value
}
