package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type scenario struct {
	Name          string
	Args          []string
	ExpectSuccess bool
	ExpectJSON    bool
	Contains      []string
}

type scenarioResult struct {
	Name       string   `json:"name"`
	Args       []string `json:"args"`
	ExitCode   int      `json:"exitCode"`
	Pass       bool     `json:"pass"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	Failure    string   `json:"failure,omitempty"`
	JSONParsed bool     `json:"jsonParsed"`
}

func main() {
	var jsonOut string
	var timeoutSeconds int
	flag.StringVar(&jsonOut, "json-out", "", "Optional JSON report output path")
	flag.IntVar(&timeoutSeconds, "timeout", 45, "Per-scenario timeout in seconds")
	flag.Parse()

	rootDir, err := os.Getwd()
	if err != nil {
		fail("resolve working directory: %v", err)
	}

	binPath, cleanup, err := buildBinary(rootDir)
	if err != nil {
		fail("build epo binary for evaluation: %v", err)
	}
	defer cleanup()

	scenarios := []scenario{
		{
			Name:          "root-help-has-agent-flags",
			Args:          []string{"--help"},
			ExpectSuccess: true,
			ExpectJSON:    false,
			Contains:      []string{"--all", "--pick", "--stdin"},
		},
		{
			Name:          "methods-json",
			Args:          []string{"methods", "-f", "json", "-q"},
			ExpectSuccess: true,
			ExpectJSON:    true,
		},
		{
			Name:          "pub-search-help-has-all-stdin",
			Args:          []string{"pub", "search", "--help"},
			ExpectSuccess: true,
			ExpectJSON:    false,
			Contains:      []string{"--all", "--stdin"},
		},
		{
			Name:          "config-show-json",
			Args:          []string{"config", "show", "-f", "json", "-q"},
			ExpectSuccess: true,
			ExpectJSON:    true,
		},
		{
			Name:          "stdin-validation",
			Args:          []string{"pub", "search", "--stdin", "-f", "json", "-q"},
			ExpectSuccess: false,
			ExpectJSON:    true,
		},
		{
			Name:          "updater-check",
			Args:          []string{"update", "--check", "-f", "json", "-q"},
			ExpectSuccess: false,
			ExpectJSON:    true,
		},
	}

	results := make([]scenarioResult, 0, len(scenarios))
	for _, sc := range scenarios {
		results = append(results, runScenario(binPath, sc, time.Duration(timeoutSeconds)*time.Second))
	}

	passed := 0
	for _, result := range results {
		if result.Pass {
			passed++
		}
	}

	recommendations := suggestImprovements(results)

	fmt.Printf("EPO Eval Runner\n")
	fmt.Printf("Binary: %s\n", binPath)
	fmt.Printf("Scenarios: %d passed / %d total\n\n", passed, len(results))
	for _, result := range results {
		status := "PASS"
		if !result.Pass {
			status = "FAIL"
		}
		fmt.Printf("- [%s] %s (exit=%d)\n", status, result.Name, result.ExitCode)
		if result.Failure != "" {
			fmt.Printf("  reason: %s\n", result.Failure)
		}
	}

	fmt.Printf("\nSuggested Improvements:\n")
	if len(recommendations) == 0 {
		fmt.Println("- No blocking issues detected by this evaluator.")
	} else {
		for _, rec := range recommendations {
			fmt.Printf("- %s\n", rec)
		}
	}

	if jsonOut != "" {
		report := map[string]any{
			"generatedAt":       time.Now().UTC().Format(time.RFC3339),
			"binaryPath":        binPath,
			"passedScenarios":   passed,
			"totalScenarios":    len(results),
			"scenarios":         results,
			"recommendations":   recommendations,
			"runnerVersionHint": "v1",
		}
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fail("marshal JSON report: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(jsonOut), 0o755); err != nil {
			fail("create output directory: %v", err)
		}
		if err := os.WriteFile(jsonOut, append(body, '\n'), 0o644); err != nil {
			fail("write JSON report: %v", err)
		}
		fmt.Printf("\nJSON report written: %s\n", jsonOut)
	}
}

func buildBinary(rootDir string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "epo-eval-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	binName := "epo"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(tmpDir, binName)

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/epo")
	cmd.Dir = rootDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", cleanup, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return binPath, cleanup, nil
}

func runScenario(binPath string, sc scenario, timeout time.Duration) scenarioResult {
	cmd := exec.Command(binPath, sc.Args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	timer := time.AfterFunc(timeout, func() {
		_ = cmd.Process.Kill()
	})
	defer timer.Stop()

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	result := scenarioResult{
		Name:     sc.Name,
		Args:     sc.Args,
		ExitCode: exitCode,
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
	}

	if sc.ExpectSuccess && exitCode != 0 {
		result.Failure = "expected success exit code"
	}
	if !sc.ExpectSuccess && exitCode == 0 {
		result.Failure = "expected non-zero exit code"
	}

	if sc.ExpectJSON {
		var payload map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
			result.Failure = appendReason(result.Failure, "expected JSON output envelope")
		} else {
			result.JSONParsed = true
		}
	}

	for _, marker := range sc.Contains {
		if !strings.Contains(result.Stdout, marker) && !strings.Contains(result.Stderr, marker) {
			result.Failure = appendReason(result.Failure, fmt.Sprintf("missing expected text %q", marker))
		}
	}

	result.Pass = result.Failure == ""
	return result
}

func appendReason(existing, reason string) string {
	if existing == "" {
		return reason
	}
	return existing + "; " + reason
}

func suggestImprovements(results []scenarioResult) []string {
	suggestions := []string{}
	for _, result := range results {
		if result.Name == "updater-check" && strings.Contains(result.Stdout, "no GitHub releases found") {
			suggestions = append(suggestions, "Publish the first GitHub release tag so `epo update` can install real artifacts.")
		}
		if result.Name == "root-help-has-agent-flags" && !result.Pass {
			suggestions = append(suggestions, "Ensure root help keeps `--all`, `--pick`, and `--stdin` exposed for agent workflows.")
		}
		if result.Name == "pub-search-help-has-all-stdin" && !result.Pass {
			suggestions = append(suggestions, "Ensure `epo pub search --help` documents both `--all` and `--stdin` options.")
		}
	}
	return dedupeStrings(suggestions)
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
