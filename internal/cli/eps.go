package cli

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/smcronin/epo-cli/internal/eps"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

const epsDefaultOutDir = ".tmp/eps-bulk"

var (
	epsDatePattern   = regexp.MustCompile(`^\d{8}$`)
	epsPatentPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)
)

func newEPSCmd() *cobra.Command {
	epsCmd := &cobra.Command{
		Use:   "eps",
		Short: "European Publication Server (EPS) operations",
		Long:  "Access EPS endpoints for publication dates, weekly patent lists, document formats, and bulk document download.",
	}
	epsCmd.AddCommand(newEPSDatesCmd())
	epsCmd.AddCommand(newEPSPatentsCmd())
	epsCmd.AddCommand(newEPSFormatsCmd())
	epsCmd.AddCommand(newEPSFetchCmd())
	epsCmd.AddCommand(newEPSBulkCmd())
	return epsCmd
}

func newEPSDatesCmd() *cobra.Command {
	var (
		fromDate string
		toDate   string
		order    string
		limit    int
	)

	cmd := &cobra.Command{
		Use:   "dates",
		Short: "List available EPS publication dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newEPSClient()
			dates, err := client.ListPublicationDates(cmd.Context())
			if err != nil {
				return err
			}
			selected, err := filterEPSDates(dates, fromDate, toDate, order, limit)
			if err != nil {
				return err
			}
			if len(selected) == 0 {
				return &epoerrors.CLIError{
					Code:    404,
					Type:    "NOT_FOUND",
					Message: "no publication dates matched the filters",
				}
			}

			requestMeta := map[string]any{
				"fromDate": strings.TrimSpace(fromDate),
				"toDate":   strings.TrimSpace(toDate),
				"order":    strings.ToLower(strings.TrimSpace(order)),
				"limit":    limit,
			}

			return outputSuccess(cmd, responsePayload{
				Service: "eps",
				Request: requestMeta,
				Results: map[string]any{
					"count": len(selected),
					"dates": selected,
				},
			})
		},
	}

	cmd.Flags().StringVar(&fromDate, "from-date", "", "Inclusive lower bound (YYYYMMDD)")
	cmd.Flags().StringVar(&toDate, "to-date", "", "Inclusive upper bound (YYYYMMDD)")
	cmd.Flags().StringVar(&order, "order", "desc", "Sort order: asc or desc")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of dates to return (0 = all)")
	return cmd
}

func newEPSPatentsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "patents <publication-date>",
		Short: "List EPS patents for a publication date",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputs, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			client := newEPSClient()

			if len(inputs) == 1 {
				date := strings.TrimSpace(inputs[0])
				if err := validateEPSDate(date); err != nil {
					return err
				}
				patents, err := client.ListPatents(cmd.Context(), date)
				if err != nil {
					return err
				}
				patents = applyEPSLimit(patents, limit)
				return outputSuccess(cmd, responsePayload{
					Service: "eps",
					Request: map[string]any{
						"publicationDate": date,
						"limit":           limit,
					},
					Results: map[string]any{
						"publicationDate": date,
						"count":           len(patents),
						"patents":         patents,
					},
				})
			}

			batch := make([]map[string]any, 0, len(inputs))
			for _, raw := range inputs {
				date := strings.TrimSpace(raw)
				if err := validateEPSDate(date); err != nil {
					batch = append(batch, map[string]any{
						"input": date,
						"ok":    false,
						"error": mapError(err),
					})
					continue
				}
				patents, err := client.ListPatents(cmd.Context(), date)
				if err != nil {
					batch = append(batch, map[string]any{
						"input": date,
						"ok":    false,
						"error": mapError(err),
					})
					continue
				}
				patents = applyEPSLimit(patents, limit)
				batch = append(batch, map[string]any{
					"input":           date,
					"ok":              true,
					"publicationDate": date,
					"count":           len(patents),
					"results": map[string]any{
						"patents": patents,
					},
				})
			}

			return outputSuccess(cmd, responsePayload{
				Service: "eps",
				Request: map[string]any{
					"limit": limit,
				},
				Results: batch,
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of patents to return per input date (0 = all)")
	return cmd
}

func newEPSFormatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "formats <patent-number>",
		Short: "List available document formats for an EPS patent number",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputs, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			client := newEPSClient()

			if len(inputs) == 1 {
				patent := normalizeEPSPatent(inputs[0])
				if err := validateEPSPatent(patent); err != nil {
					return err
				}
				formats, err := client.ListDocumentFormats(cmd.Context(), patent)
				if err != nil {
					return err
				}
				formatNames := make([]string, 0, len(formats))
				for _, item := range formats {
					formatNames = append(formatNames, item.Format)
				}
				return outputSuccess(cmd, responsePayload{
					Service: "eps",
					Request: map[string]any{"patentNumber": patent},
					Results: map[string]any{
						"patentNumber": patent,
						"count":        len(formats),
						"formats":      formatNames,
						"links":        formats,
					},
				})
			}

			batch := make([]map[string]any, 0, len(inputs))
			for _, raw := range inputs {
				patent := normalizeEPSPatent(raw)
				if err := validateEPSPatent(patent); err != nil {
					batch = append(batch, map[string]any{
						"input": patent,
						"ok":    false,
						"error": mapError(err),
					})
					continue
				}
				formats, err := client.ListDocumentFormats(cmd.Context(), patent)
				if err != nil {
					batch = append(batch, map[string]any{
						"input": patent,
						"ok":    false,
						"error": mapError(err),
					})
					continue
				}
				formatNames := make([]string, 0, len(formats))
				for _, item := range formats {
					formatNames = append(formatNames, item.Format)
				}
				batch = append(batch, map[string]any{
					"input": patent,
					"ok":    true,
					"results": map[string]any{
						"patentNumber": patent,
						"count":        len(formats),
						"formats":      formatNames,
						"links":        formats,
					},
				})
			}

			return outputSuccess(cmd, responsePayload{
				Service: "eps",
				Results: batch,
			})
		},
	}

	return cmd
}

func newEPSFetchCmd() *cobra.Command {
	var (
		format      string
		outPath     string
		overwrite   bool
		includeBody bool
	)

	cmd := &cobra.Command{
		Use:   "fetch <patent-number>",
		Short: "Download EPS document content in xml/html/pdf/zip",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			patent := normalizeEPSPatent(args[0])
			if err := validateEPSPatent(patent); err != nil {
				return err
			}
			normalizedFormat, err := validateEPSFormat(format)
			if err != nil {
				return err
			}

			client := newEPSClient()
			resp, err := client.FetchDocument(cmd.Context(), patent, normalizedFormat)
			if err != nil {
				return err
			}

			writtenPath := ""
			if strings.TrimSpace(outPath) != "" {
				target := strings.TrimSpace(outPath)
				if !overwrite {
					if _, statErr := os.Stat(target); statErr == nil {
						return &epoerrors.CLIError{
							Code:    400,
							Type:    "VALIDATION_ERROR",
							Message: fmt.Sprintf("output file already exists: %s", target),
							Hint:    "Pass --overwrite to replace it",
						}
					}
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return fmt.Errorf("create output directory: %w", err)
				}
				if err := os.WriteFile(target, resp.Body, 0o644); err != nil {
					return fmt.Errorf("write output file: %w", err)
				}
				writtenPath = target
			}

			hash := sha256.Sum256(resp.Body)
			result := map[string]any{
				"patentNumber": patent,
				"format":       normalizedFormat,
				"contentType":  strings.TrimSpace(resp.Headers.Get("Content-Type")),
				"bytes":        len(resp.Body),
				"sha256":       hex.EncodeToString(hash[:]),
			}
			if writtenPath != "" {
				result["out"] = writtenPath
			}
			if includeBody {
				result["bodyBase64"] = base64.StdEncoding.EncodeToString(resp.Body)
			}

			return outputSuccess(cmd, responsePayload{
				Service: "eps",
				Request: map[string]any{
					"patentNumber": patent,
					"format":       normalizedFormat,
					"out":          strings.TrimSpace(outPath),
					"includeBody":  includeBody,
				},
				Results: result,
			})
		},
	}

	cmd.Flags().StringVar(&format, "format", "xml", "Document format: xml, html, pdf, zip")
	cmd.Flags().StringVar(&outPath, "out", "", "Write output bytes to file path")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite output file if it exists")
	cmd.Flags().BoolVar(&includeBody, "include-body", false, "Include base64-encoded body in JSON output")
	return cmd
}

func newEPSBulkCmd() *cobra.Command {
	var (
		fromDate     string
		toDate       string
		singleDate   string
		order        string
		limitDates   int
		maxPatents   int
		format       string
		outDir       string
		concurrency  int
		skipExisting bool
		indexOnly    bool
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:   "bulk",
		Short: "Bulk index/download EPS data by publication date",
		Long:  "Build publication-date indexes and optionally download raw EPS documents (xml/html/pdf/zip) into a local data-lake folder.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(singleDate) != "" {
				if strings.TrimSpace(fromDate) != "" || strings.TrimSpace(toDate) != "" {
					return &epoerrors.CLIError{
						Code:    400,
						Type:    "VALIDATION_ERROR",
						Message: "use either --date or --from-date/--to-date",
					}
				}
				fromDate = strings.TrimSpace(singleDate)
				toDate = strings.TrimSpace(singleDate)
			}
			if strings.TrimSpace(fromDate) != "" {
				if err := validateEPSDate(fromDate); err != nil {
					return err
				}
			}
			if strings.TrimSpace(toDate) != "" {
				if err := validateEPSDate(toDate); err != nil {
					return err
				}
			}
			if concurrency <= 0 {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "--concurrency must be >= 1",
				}
			}
			normalizedFormat, err := validateEPSFormat(format)
			if err != nil {
				return err
			}
			if maxPatents < 0 {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "--max-patents must be >= 0",
				}
			}
			if limitDates < 0 {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "--max-dates must be >= 0",
				}
			}

			outputDir := strings.TrimSpace(outDir)
			if outputDir == "" {
				outputDir = epsDefaultOutDir
			}
			indexDir := filepath.Join(outputDir, "indexes")
			if err := os.MkdirAll(indexDir, 0o755); err != nil {
				return fmt.Errorf("create index directory: %w", err)
			}

			client := newEPSClient()
			allDates, err := client.ListPublicationDates(cmd.Context())
			if err != nil {
				return err
			}
			selectedDates, err := filterEPSDates(allDates, fromDate, toDate, order, limitDates)
			if err != nil {
				return err
			}
			if len(selectedDates) == 0 {
				return &epoerrors.CLIError{
					Code:    404,
					Type:    "NOT_FOUND",
					Message: "no publication dates matched the bulk filters",
				}
			}

			if err := writeStringLines(filepath.Join(indexDir, "publication-dates.txt"), selectedDates); err != nil {
				return err
			}

			jobs := make([]epsDownloadJob, 0, 4096)
			indexedPatents := 0
			perDateCounts := map[string]int{}
			errorsOut := []string{}

			datePatentDir := filepath.Join(indexDir, "patents")
			if err := os.MkdirAll(datePatentDir, 0o755); err != nil {
				return fmt.Errorf("create patents index directory: %w", err)
			}

			for _, date := range selectedDates {
				patents, listErr := client.ListPatents(cmd.Context(), date)
				if listErr != nil {
					errorsOut = append(errorsOut, fmt.Sprintf("list patents %s: %v", date, listErr))
					continue
				}
				if maxPatents > 0 {
					remaining := maxPatents - indexedPatents
					if remaining <= 0 {
						perDateCounts[date] = 0
						break
					}
					if len(patents) > remaining {
						patents = patents[:remaining]
					}
				}

				perDateCounts[date] = len(patents)
				indexedPatents += len(patents)

				if err := writeStringLines(filepath.Join(datePatentDir, date+".txt"), patents); err != nil {
					return err
				}

				if !indexOnly {
					for _, patent := range patents {
						jobs = append(jobs, epsDownloadJob{
							Date:   date,
							Patent: patent,
						})
					}
				}

				if maxPatents > 0 && indexedPatents >= maxPatents {
					break
				}
			}

			result := map[string]any{
				"outDir":                outputDir,
				"format":                normalizedFormat,
				"selectedDates":         selectedDates,
				"dateCount":             len(selectedDates),
				"indexedPatentCount":    indexedPatents,
				"downloadQueueCount":    len(jobs),
				"indexOnly":             indexOnly,
				"dryRun":                dryRun,
				"skipExisting":          skipExisting,
				"concurrency":           concurrency,
				"publicationDatesIndex": filepath.Join(indexDir, "publication-dates.txt"),
				"datePatentIndexDir":    datePatentDir,
			}

			var downloaded int64
			var skipped int64
			var failed int64
			var bytesDownloaded int64
			if !indexOnly && !dryRun && len(jobs) > 0 {
				downloadErrs := runEPSDownloadWorkers(cmd.Context(), client, outputDir, normalizedFormat, concurrency, skipExisting, jobs, &downloaded, &skipped, &failed, &bytesDownloaded)
				if len(downloadErrs) > 0 {
					errorsOut = append(errorsOut, downloadErrs...)
				}
			}

			result["downloadedCount"] = downloaded
			result["skippedExistingCount"] = skipped
			result["failedCount"] = failed
			result["bytesDownloaded"] = bytesDownloaded
			result["datePatentCounts"] = perDateCounts

			if len(errorsOut) > 0 {
				if len(errorsOut) > 50 {
					errorsOut = append(errorsOut[:50], fmt.Sprintf("...truncated %d more errors", len(errorsOut)-50))
				}
				result["errors"] = errorsOut
			}

			manifestPath := filepath.Join(outputDir, "manifest.json")
			manifest := map[string]any{
				"generatedAt":      time.Now().UTC().Format(time.RFC3339),
				"baseURL":          eps.DefaultBaseURL,
				"fromDate":         strings.TrimSpace(fromDate),
				"toDate":           strings.TrimSpace(toDate),
				"singleDate":       strings.TrimSpace(singleDate),
				"order":            strings.ToLower(strings.TrimSpace(order)),
				"maxDates":         limitDates,
				"maxPatents":       maxPatents,
				"format":           normalizedFormat,
				"outDir":           outputDir,
				"concurrency":      concurrency,
				"skipExisting":     skipExisting,
				"indexOnly":        indexOnly,
				"dryRun":           dryRun,
				"summary":          result,
				"selectedDates":    selectedDates,
				"datePatentCounts": perDateCounts,
			}
			if err := writeJSONFile(manifestPath, manifest); err != nil {
				return err
			}
			result["manifest"] = manifestPath

			return outputSuccess(cmd, responsePayload{
				Service: "eps",
				Request: map[string]any{
					"fromDate":     strings.TrimSpace(fromDate),
					"toDate":       strings.TrimSpace(toDate),
					"singleDate":   strings.TrimSpace(singleDate),
					"order":        strings.ToLower(strings.TrimSpace(order)),
					"maxDates":     limitDates,
					"maxPatents":   maxPatents,
					"format":       normalizedFormat,
					"outDir":       outputDir,
					"concurrency":  concurrency,
					"skipExisting": skipExisting,
					"indexOnly":    indexOnly,
					"dryRun":       dryRun,
				},
				Results: result,
			})
		},
	}

	cmd.Flags().StringVar(&fromDate, "from-date", "", "Inclusive lower date bound YYYYMMDD")
	cmd.Flags().StringVar(&toDate, "to-date", "", "Inclusive upper date bound YYYYMMDD")
	cmd.Flags().StringVar(&singleDate, "date", "", "Single publication date YYYYMMDD")
	cmd.Flags().StringVar(&order, "order", "desc", "Date iteration order: asc or desc")
	cmd.Flags().IntVar(&limitDates, "max-dates", 0, "Maximum publication dates to process (0 = all in range)")
	cmd.Flags().IntVar(&maxPatents, "max-patents", 0, "Maximum patents to index/download (0 = all)")
	cmd.Flags().StringVar(&format, "format", "zip", "Document format to download: xml, html, pdf, zip")
	cmd.Flags().StringVar(&outDir, "out-dir", epsDefaultOutDir, "Output root directory for indexes and downloaded files")
	cmd.Flags().IntVar(&concurrency, "concurrency", 4, "Number of parallel download workers")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", true, "Skip downloads when target file already exists")
	cmd.Flags().BoolVar(&indexOnly, "index-only", false, "Only build date/patent index files without downloading documents")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Build indexes and queue summary without downloading files")
	return cmd
}

type epsDownloadJob struct {
	Date   string
	Patent string
}

func newEPSClient() *eps.Client {
	httpClient := &http.Client{Timeout: time.Duration(flagTimeout) * time.Second}
	return eps.NewClient(httpClient)
}

func validateEPSDate(date string) error {
	date = strings.TrimSpace(date)
	if !epsDatePattern.MatchString(date) {
		return &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: fmt.Sprintf("invalid EPS publication date: %q", date),
			Hint:    "Use YYYYMMDD (example: 20240131)",
		}
	}
	if _, err := time.Parse("20060102", date); err != nil {
		return &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: fmt.Sprintf("invalid calendar date: %q", date),
			Hint:    "Use a real date in YYYYMMDD format",
		}
	}
	return nil
}

func normalizeEPSPatent(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func validateEPSPatent(patent string) error {
	patent = normalizeEPSPatent(patent)
	if patent == "" {
		return &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "patent number is required",
		}
	}
	if !epsPatentPattern.MatchString(patent) {
		return &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: fmt.Sprintf("invalid EPS patent number: %q", patent),
			Hint:    "Use country+number+correction+kind (example: EP1004359NWB1)",
		}
	}
	return nil
}

func validateEPSFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "xml", "html", "pdf", "zip":
		return strings.ToLower(strings.TrimSpace(format)), nil
	default:
		return "", &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: fmt.Sprintf("unsupported EPS format %q", format),
			Hint:    "Use xml, html, pdf, or zip",
		}
	}
}

func filterEPSDates(dates []string, fromDate, toDate, order string, limit int) ([]string, error) {
	fromDate = strings.TrimSpace(fromDate)
	toDate = strings.TrimSpace(toDate)
	order = strings.ToLower(strings.TrimSpace(order))

	if fromDate != "" {
		if err := validateEPSDate(fromDate); err != nil {
			return nil, err
		}
	}
	if toDate != "" {
		if err := validateEPSDate(toDate); err != nil {
			return nil, err
		}
	}
	if fromDate != "" && toDate != "" && fromDate > toDate {
		return nil, &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "--from-date cannot be greater than --to-date",
		}
	}
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		return nil, &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: fmt.Sprintf("invalid order %q", order),
			Hint:    "Use asc or desc",
		}
	}
	if limit < 0 {
		return nil, &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "limit must be >= 0",
		}
	}

	items := make([]string, 0, len(dates))
	for _, raw := range dates {
		date := strings.TrimSpace(raw)
		if !epsDatePattern.MatchString(date) {
			continue
		}
		if fromDate != "" && date < fromDate {
			continue
		}
		if toDate != "" && date > toDate {
			continue
		}
		items = append(items, date)
	}
	sort.Strings(items)
	if order == "desc" {
		reverseStringSlice(items)
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func reverseStringSlice(items []string) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func applyEPSLimit(items []string, limit int) []string {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}

func writeStringLines(path string, lines []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest JSON: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func epsTargetPath(root, format, date, patent string) string {
	fileName := patent + "." + format
	return filepath.Join(root, "documents", date, fileName)
}

func runEPSDownloadWorkers(
	ctx context.Context,
	client *eps.Client,
	rootDir string,
	format string,
	concurrency int,
	skipExisting bool,
	jobs []epsDownloadJob,
	downloaded *int64,
	skipped *int64,
	failed *int64,
	bytesDownloaded *int64,
) []string {
	if len(jobs) == 0 {
		return nil
	}
	if concurrency < 1 {
		concurrency = 1
	}

	var mu sync.Mutex
	errorsOut := []string{}

	jobCh := make(chan epsDownloadJob)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				select {
				case <-ctx.Done():
					atomic.AddInt64(failed, 1)
					mu.Lock()
					errorsOut = append(errorsOut, fmt.Sprintf("%s/%s: %v", job.Date, job.Patent, ctx.Err()))
					mu.Unlock()
					continue
				default:
				}

				target := epsTargetPath(rootDir, format, job.Date, job.Patent)
				if skipExisting {
					if info, err := os.Stat(target); err == nil && !info.IsDir() {
						atomic.AddInt64(skipped, 1)
						continue
					}
				}

				resp, err := client.FetchDocument(ctx, job.Patent, format)
				if err != nil {
					atomic.AddInt64(failed, 1)
					mu.Lock()
					errorsOut = append(errorsOut, fmt.Sprintf("%s/%s: %v", job.Date, job.Patent, err))
					mu.Unlock()
					continue
				}

				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					atomic.AddInt64(failed, 1)
					mu.Lock()
					errorsOut = append(errorsOut, fmt.Sprintf("%s/%s: mkdir: %v", job.Date, job.Patent, err))
					mu.Unlock()
					continue
				}

				temp := target + ".part"
				if err := os.WriteFile(temp, resp.Body, 0o644); err != nil {
					atomic.AddInt64(failed, 1)
					mu.Lock()
					errorsOut = append(errorsOut, fmt.Sprintf("%s/%s: write temp: %v", job.Date, job.Patent, err))
					mu.Unlock()
					continue
				}
				if err := os.Rename(temp, target); err != nil {
					_ = os.Remove(temp)
					atomic.AddInt64(failed, 1)
					mu.Lock()
					errorsOut = append(errorsOut, fmt.Sprintf("%s/%s: rename: %v", job.Date, job.Patent, err))
					mu.Unlock()
					continue
				}

				atomic.AddInt64(downloaded, 1)
				atomic.AddInt64(bytesDownloaded, int64(len(resp.Body)))
			}
		}()
	}

	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)
	wg.Wait()

	return errorsOut
}
