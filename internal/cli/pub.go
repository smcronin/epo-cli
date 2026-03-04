package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/smcronin/epo-cli/internal/api"
	"github.com/smcronin/epo-cli/internal/auth"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

const (
	defaultPubSearchTablePick = "country,docNumber,kind,pubDate,title,familyId"
	pubSearchSortNone         = "none"
	pubSearchSortDateAsc      = "pub-date-asc"
	pubSearchSortDateDesc     = "pub-date-desc"
)

func newPubCmd() *cobra.Command {
	pubCmd := &cobra.Command{
		Use:   "pub",
		Short: "Published-data service operations",
	}

	pubCmd.AddCommand(newPubBiblioCmd())
	pubCmd.AddCommand(newPubSimpleEndpointCmd("abstract", "abstract", "Fetch abstract text/metadata for a published reference"))
	pubCmd.AddCommand(newPubSimpleEndpointCmd("fulltext", "fulltext", "Fetch fulltext availability inquiry for a published reference"))
	pubCmd.AddCommand(newPubSimpleEndpointCmd("claims", "claims", "Fetch claims for a published reference"))
	pubCmd.AddCommand(newPubSimpleEndpointCmd("description", "description", "Fetch description fulltext for a published reference"))
	pubCmd.AddCommand(newPubImagesCmd())
	pubCmd.AddCommand(newPubEquivalentsCmd())
	pubCmd.AddCommand(newPubSearchCmd())
	return pubCmd
}

func newPubBiblioCmd() *cobra.Command {
	var (
		refType      string
		inputFormat  string
		constituents string
	)

	cmd := &cobra.Command{
		Use:   "biblio <reference>",
		Short: "Fetch published-data bibliographic record",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isOneOf(refType, "publication", "application", "priority") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --ref-type %q", refType),
					Hint:    "Use publication, application, or priority",
				}
			}
			if !isOneOf(inputFormat, "epodoc", "docdb") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --input-format %q", inputFormat),
					Hint:    "Use epodoc or docdb",
				}
			}

			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runOPSBatch(cmd, "published-data", references, func(reference string) (api.Request, map[string]any, error) {
				path := fmt.Sprintf("/published-data/%s/%s/%s/%s", refType, inputFormat, reference, constituents)
				request := api.Request{
					Method: http.MethodGet,
					Path:   path,
					Accept: "application/json",
				}
				requestMeta := map[string]any{
					"method": request.Method,
					"path":   request.Path,
				}
				return request, requestMeta, nil
			}, parseSearchPagination)
		},
	}

	cmd.Flags().StringVar(&refType, "ref-type", "publication", "Reference type: publication, application, priority")
	cmd.Flags().StringVar(&inputFormat, "input-format", "epodoc", "Input format: epodoc or docdb")
	cmd.Flags().StringVar(&constituents, "constituents", "biblio", "Response constituents (for example: biblio or biblio,full-cycle)")
	return cmd
}

func newPubSimpleEndpointCmd(name, endpoint, shortDesc string) *cobra.Command {
	var (
		refType     string
		inputFormat string
	)

	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <reference>", name),
		Short: shortDesc,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isOneOf(refType, "publication", "application", "priority") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --ref-type %q", refType),
					Hint:    "Use publication, application, or priority",
				}
			}
			if !isOneOf(inputFormat, "epodoc", "docdb") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --input-format %q", inputFormat),
					Hint:    "Use epodoc or docdb",
				}
			}

			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runOPSBatch(cmd, "published-data", references, func(reference string) (api.Request, map[string]any, error) {
				path := fmt.Sprintf("/published-data/%s/%s/%s/%s", refType, inputFormat, reference, endpoint)
				request := api.Request{
					Method: http.MethodGet,
					Path:   path,
					Accept: "application/json",
				}
				requestMeta := map[string]any{
					"method": request.Method,
					"path":   request.Path,
				}
				return request, requestMeta, nil
			}, parseSearchPagination)
		},
	}

	cmd.Flags().StringVar(&refType, "ref-type", "publication", "Reference type: publication, application, priority")
	cmd.Flags().StringVar(&inputFormat, "input-format", "epodoc", "Input format: epodoc or docdb")
	return cmd
}

func newPubEquivalentsCmd() *cobra.Command {
	var (
		refType      string
		inputFormat  string
		constituents string
	)

	cmd := &cobra.Command{
		Use:   "equivalents <reference>",
		Short: "Fetch simple-family equivalents for a published reference",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isOneOf(refType, "publication", "application", "priority") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --ref-type %q", refType),
					Hint:    "Use publication, application, or priority",
				}
			}
			if !isOneOf(inputFormat, "epodoc", "docdb") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --input-format %q", inputFormat),
					Hint:    "Use epodoc or docdb",
				}
			}

			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runOPSBatch(cmd, "published-data", references, func(reference string) (api.Request, map[string]any, error) {
				basePath := fmt.Sprintf("/published-data/%s/%s/%s/equivalents", refType, inputFormat, reference)
				path := basePath
				if v := strings.TrimSpace(constituents); v != "" {
					path = basePath + "/" + v
				}
				request := api.Request{
					Method: http.MethodGet,
					Path:   path,
					Accept: "application/json",
				}
				requestMeta := map[string]any{
					"method": request.Method,
					"path":   request.Path,
				}
				return request, requestMeta, nil
			}, parseSearchPagination)
		},
	}

	cmd.Flags().StringVar(&refType, "ref-type", "publication", "Reference type: publication, application, priority")
	cmd.Flags().StringVar(&inputFormat, "input-format", "epodoc", "Input format: epodoc or docdb")
	cmd.Flags().StringVar(&constituents, "constituents", "", "Equivalent constituents: abstract, biblio, biblio,full-cycle, images")
	return cmd
}

func newPubSearchCmd() *cobra.Command {
	var (
		query        string
		constituents string
		rangeHeader  string
		usePost      bool
		sortModeRaw  string
		flatMode     bool
		tableMode    bool
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Run a CQL search against published-data",
		Example: strings.TrimSpace(`
epo pub search --query "applicant=IBM" --range 1-25 -f json -q
epo pub search --query "applicant=\"SAP SE\" and pd>=2024" --all --sort pub-date-desc --flat -f json -q
epo pub search --query "applicant=IBM" --all --table
echo "applicant=IBM" | epo pub search --stdin --all --table
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			queries, err := resolveQueryOrStdinInputs(query)
			if err != nil {
				return err
			}

			if tableMode {
				flagFormat = "table"
				flatMode = true
				if strings.TrimSpace(flagPick) == "" {
					flagPick = defaultPubSearchTablePick
				}
			}

			sortMode, err := normalizePubSearchSort(sortModeRaw)
			if err != nil {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: err.Error(),
					Hint:    "Use --sort none, pub-date-asc, or pub-date-desc",
				}
			}
			if sortMode != pubSearchSortNone && !flagAll && !flatMode {
				flatMode = true
			}

			if flagAll {
				return runPublishedSearchAll(cmd, queries, constituents, rangeHeader, usePost, sortMode, flatMode)
			}

			if flatMode {
				return runPublishedSearchFlat(cmd, queries, constituents, rangeHeader, usePost, sortMode)
			}

			path := "/published-data/search"
			if v := strings.TrimSpace(constituents); v != "" {
				path += "/" + v
			}

			return runOPSBatch(cmd, "published-data", queries, func(inputQuery string) (api.Request, map[string]any, error) {
				request := api.Request{
					Method: http.MethodGet,
					Path:   path,
					Accept: "application/json",
				}
				if strings.TrimSpace(rangeHeader) != "" {
					request.Headers = map[string]string{
						"Range": strings.TrimSpace(rangeHeader),
					}
				}

				if usePost {
					request.Method = http.MethodPost
					request.ContentType = "text/plain"
					request.Body = []byte("q=" + url.QueryEscape(inputQuery))
				} else {
					request.Query = url.Values{}
					request.Query.Set("q", inputQuery)
				}

				requestMeta := map[string]any{
					"method": request.Method,
					"path":   request.Path,
					"query":  compactQuery(request.Query),
				}
				if len(request.Headers) > 0 {
					requestMeta["headers"] = request.Headers
				}
				return request, requestMeta, nil
			}, parseSearchPagination)
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "CQL query (for example: applicant=IBM)")
	cmd.Flags().StringVar(&query, "cql", "", "Alias for --query")
	cmd.Flags().StringVar(&query, "q", "", "Deprecated alias for --query")
	_ = cmd.Flags().MarkDeprecated("q", "use --query (or --cql)")
	cmd.Flags().StringVar(&constituents, "constituents", "", "Search constituents (for example: biblio,abstract,full-cycle)")
	cmd.Flags().StringVar(&rangeHeader, "range", "", "Result range, for example 1-25")
	cmd.Flags().BoolVar(&usePost, "post", false, "Use POST instead of GET")
	cmd.Flags().StringVar(&sortModeRaw, "sort", "none", "Client sort mode: none, pub-date-asc, pub-date-desc")
	cmd.Flags().BoolVar(&flatMode, "flat", false, "Return flattened search rows (country/docNumber/kind/pubDate/title)")
	cmd.Flags().BoolVar(&tableMode, "table", false, "Shortcut for --format table --flat and default --pick columns")
	return cmd
}

func newPubImagesCmd() *cobra.Command {
	imagesCmd := &cobra.Command{
		Use:   "images",
		Short: "Published-data images inquiry and retrieval",
	}
	imagesCmd.AddCommand(newPubImagesInquiryCmd())
	imagesCmd.AddCommand(newPubImagesFetchCmd())
	return imagesCmd
}

func newPubImagesInquiryCmd() *cobra.Command {
	var (
		refType     string
		inputFormat string
	)

	cmd := &cobra.Command{
		Use:   "inquiry <reference>",
		Short: "List available images/documents for a published reference",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isOneOf(refType, "publication", "application", "priority") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --ref-type %q", refType),
					Hint:    "Use publication, application, or priority",
				}
			}
			if !isOneOf(inputFormat, "epodoc", "docdb") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --input-format %q", inputFormat),
					Hint:    "Use epodoc or docdb",
				}
			}
			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}

			return runOPSBatch(cmd, "published-data", references, func(reference string) (api.Request, map[string]any, error) {
				request := api.Request{
					Method: http.MethodGet,
					Path:   fmt.Sprintf("/published-data/%s/%s/%s/images", refType, inputFormat, reference),
					Accept: "application/json",
				}
				requestMeta := map[string]any{
					"method": request.Method,
					"path":   request.Path,
				}
				return request, requestMeta, nil
			}, nil)
		},
	}

	cmd.Flags().StringVar(&refType, "ref-type", "publication", "Reference type: publication, application, priority")
	cmd.Flags().StringVar(&inputFormat, "input-format", "epodoc", "Input format: epodoc or docdb")
	return cmd
}

func newPubImagesFetchCmd() *cobra.Command {
	var (
		accept      string
		rangeQuery  string
		fromSystem  string
		outPath     string
		includeBody bool
	)

	cmd := &cobra.Command{
		Use:   "fetch <link-path>",
		Short: "Fetch an image/document by link from images inquiry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			links, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}

			results := make([]map[string]any, 0, len(links))
			for i, linkPath := range links {
				linkPath = strings.TrimPrefix(strings.TrimSpace(linkPath), "/")
				if linkPath == "" {
					results = append(results, map[string]any{
						"input": linkPath,
						"ok":    false,
						"error": &epoerrors.CLIError{
							Code:    400,
							Type:    "VALIDATION_ERROR",
							Message: "link-path is required",
						},
					})
					continue
				}

				query := url.Values{}
				if v := strings.TrimSpace(rangeQuery); v != "" {
					query.Set("Range", v)
				}
				if v := strings.TrimSpace(fromSystem); v != "" {
					query.Set("From", v)
				}

				request := api.Request{
					Method: http.MethodGet,
					Path:   "/published-data/images/" + linkPath,
					Query:  query,
					Accept: defaultIfEmptyString(accept, "application/pdf"),
				}
				requestMeta := map[string]any{
					"method": request.Method,
					"path":   request.Path,
					"query":  compactQuery(request.Query),
					"accept": request.Accept,
				}

				resp, err := executeOPSRequest(cmd.Context(), request)
				if err != nil {
					results = append(results, map[string]any{
						"input": linkPath,
						"ok":    false,
						"error": mapError(err),
					})
					continue
				}

				result := map[string]any{
					"input":       linkPath,
					"ok":          true,
					"request":     requestMeta,
					"contentType": resp.Headers.Get("Content-Type"),
					"bytes":       len(resp.Body),
					"sha256":      sha256Hex(resp.Body),
				}

				if strings.TrimSpace(outPath) != "" {
					savePath := outPath
					if len(links) > 1 {
						savePath = fmt.Sprintf("%s.%d", outPath, i+1)
					}
					if mkErr := os.MkdirAll(filepath.Dir(savePath), 0o755); mkErr != nil {
						result["ok"] = false
						result["error"] = mapError(fmt.Errorf("create output directory for %q: %w", savePath, mkErr))
					} else if writeErr := os.WriteFile(savePath, resp.Body, 0o644); writeErr != nil {
						result["ok"] = false
						result["error"] = mapError(fmt.Errorf("write output file %q: %w", savePath, writeErr))
					} else {
						result["saved"] = true
						result["outputPath"] = savePath
					}
				}
				if includeBody {
					result["base64"] = encodeBase64(resp.Body)
				}

				results = append(results, result)
			}

			if len(results) == 1 {
				single := results[0]
				if ok, _ := single["ok"].(bool); ok {
					return outputSuccess(cmd, responsePayload{
						Service: "published-data",
						Results: single,
					})
				}
				return &epoerrors.CLIError{
					Code:    1,
					Type:    "GENERAL_ERROR",
					Message: fmt.Sprintf("%v", single["error"]),
				}
			}

			return outputSuccess(cmd, responsePayload{
				Service: "published-data",
				Results: results,
			})
		},
	}

	cmd.Flags().StringVar(&accept, "accept", "application/pdf", "Accept header (for example application/pdf, image/png, application/tiff)")
	cmd.Flags().StringVar(&rangeQuery, "range", "", "Optional page selector (single number for fullimage/tiff)")
	cmd.Flags().StringVar(&fromSystem, "from", "", "Optional source system (From header/query equivalent)")
	cmd.Flags().StringVar(&outPath, "out", "", "Output file path (for batch mode, files are suffixed with .N)")
	cmd.Flags().BoolVar(&includeBody, "include-body", false, "Include base64 body in JSON output")
	return cmd
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func runPublishedSearchAll(cmd *cobra.Command, queries []string, constituents, rangeHeader string, usePost bool, sortMode string, flatMode bool) error {
	start, end, err := parseRangeWindow(rangeHeader)
	if err != nil {
		return &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: err.Error(),
			Hint:    "Use --range start-end (for example 1-25)",
		}
	}
	pageSize := end - start + 1

	path := "/published-data/search"
	if v := strings.TrimSpace(constituents); v != "" {
		path += "/" + v
	}

	batchResults := make([]map[string]any, 0, len(queries))
	for _, q := range queries {
		currentStart := start
		currentEnd := end
		pages := 0
		total := 0
		allItems := make([]any, 0)
		combinedWarnings := make([]string, 0)
		var throttleSnapshot any
		var quotaSnapshot any

		for {
			request := api.Request{
				Method: http.MethodGet,
				Path:   path,
				Accept: "application/json",
				Headers: map[string]string{
					"Range": fmt.Sprintf("%d-%d", currentStart, currentEnd),
				},
			}
			if usePost {
				request.Method = http.MethodPost
				request.ContentType = "text/plain"
				request.Body = []byte("q=" + url.QueryEscape(q))
			} else {
				request.Query = url.Values{}
				request.Query.Set("q", q)
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				batchResults = append(batchResults, map[string]any{
					"query": q,
					"ok":    false,
					"error": mapError(err),
				})
				break
			}

			pages++
			throttleSnapshot = map[string]any{
				"system":   resp.Metadata.Throttle.System,
				"services": resp.Metadata.Throttle.Services,
			}
			quotaSnapshot = map[string]int{
				"hourUsed":       resp.Metadata.Quota.IndividualPerHourUsed,
				"weekUsed":       resp.Metadata.Quota.RegisteredPerWeekUsed,
				"payingWeekUsed": resp.Metadata.Quota.RegisteredPayingPerWeekUsed,
			}

			parsed, warnings := parseJSONBody(resp.Body)
			combinedWarnings = append(combinedWarnings, warnings...)
			pagination := parseSearchPagination(parsed)

			items, _ := extractPublishedSearchItems(parsed)
			allItems = append(allItems, items...)

			if value, ok := pagination["total"]; ok {
				if totalInt, ok := value.(int); ok {
					total = totalInt
				}
			}

			hasMore := false
			if value, ok := pagination["hasMore"]; ok {
				if boolValue, ok := value.(bool); ok {
					hasMore = boolValue
				}
			}
			if !hasMore || len(items) == 0 {
				if sortMode != pubSearchSortNone {
					sortPublishedSearchItems(allItems, sortMode)
				}
				var resultPayload any = allItems
				if flatMode {
					resultPayload = flattenPublishedSearchItems(allItems)
				}
				batchResults = append(batchResults, map[string]any{
					"query": q,
					"ok":    true,
					"request": map[string]any{
						"method": request.Method,
						"path":   request.Path,
						"query":  compactQuery(request.Query),
						"all":    true,
					},
					"pagination": map[string]any{
						"offset":       start,
						"limit":        pageSize,
						"total":        total,
						"pagesFetched": pages,
						"returned":     len(allItems),
					},
					"throttle": throttleSnapshot,
					"quota":    quotaSnapshot,
					"results":  resultPayload,
					"warnings": joinAndSortUnique(combinedWarnings),
				})
				break
			}

			currentStart = currentEnd + 1
			currentEnd = currentStart + pageSize - 1
		}
	}

	if len(batchResults) == 1 {
		single := batchResults[0]
		if ok, _ := single["ok"].(bool); ok {
			return outputSuccess(cmd, responsePayload{
				Service:    "published-data",
				Request:    single["request"],
				Pagination: single["pagination"],
				Throttle:   single["throttle"],
				Quota:      single["quota"],
				Results:    single["results"],
				Warnings:   toStringSlice(single["warnings"]),
			})
		}
		return &epoerrors.CLIError{
			Code:    1,
			Type:    "GENERAL_ERROR",
			Message: fmt.Sprintf("%v", single["error"]),
		}
	}

	return outputSuccess(cmd, responsePayload{
		Service: "published-data",
		Results: batchResults,
	})
}

func runPublishedSearchFlat(cmd *cobra.Command, queries []string, constituents, rangeHeader string, usePost bool, sortMode string) error {
	path := "/published-data/search"
	if v := strings.TrimSpace(constituents); v != "" {
		path += "/" + v
	}

	batchResults := make([]map[string]any, 0, len(queries))
	for _, q := range queries {
		request := api.Request{
			Method: http.MethodGet,
			Path:   path,
			Accept: "application/json",
		}
		if strings.TrimSpace(rangeHeader) != "" {
			request.Headers = map[string]string{
				"Range": strings.TrimSpace(rangeHeader),
			}
		}
		if usePost {
			request.Method = http.MethodPost
			request.ContentType = "text/plain"
			request.Body = []byte("q=" + url.QueryEscape(q))
		} else {
			request.Query = url.Values{}
			request.Query.Set("q", q)
		}

		requestMeta := map[string]any{
			"method": request.Method,
			"path":   request.Path,
			"query":  compactQuery(request.Query),
		}
		if len(request.Headers) > 0 {
			requestMeta["headers"] = request.Headers
		}

		resp, err := executeOPSRequest(cmd.Context(), request)
		if err != nil {
			batchResults = append(batchResults, map[string]any{
				"query": q,
				"ok":    false,
				"error": mapError(err),
			})
			continue
		}

		parsed, warnings := parseJSONBody(resp.Body)
		pagination := parsePagination(strings.TrimSpace(resp.Headers.Get("X-OPS-Range")))
		pagination = mergePagination(pagination, parseSearchPagination(parsed))
		items, _ := extractPublishedSearchItems(parsed)
		if sortMode != pubSearchSortNone {
			sortPublishedSearchItems(items, sortMode)
		}
		flatRows := flattenPublishedSearchItems(items)

		batchResults = append(batchResults, map[string]any{
			"query":      q,
			"ok":         true,
			"request":    requestMeta,
			"pagination": pagination,
			"throttle": map[string]any{
				"system":   resp.Metadata.Throttle.System,
				"services": resp.Metadata.Throttle.Services,
			},
			"quota": map[string]int{
				"hourUsed":       resp.Metadata.Quota.IndividualPerHourUsed,
				"weekUsed":       resp.Metadata.Quota.RegisteredPerWeekUsed,
				"payingWeekUsed": resp.Metadata.Quota.RegisteredPayingPerWeekUsed,
			},
			"results":  flatRows,
			"warnings": warnings,
		})
	}

	if len(batchResults) == 1 {
		single := batchResults[0]
		if ok, _ := single["ok"].(bool); ok {
			return outputSuccess(cmd, responsePayload{
				Service:    "published-data",
				Request:    single["request"],
				Pagination: single["pagination"],
				Throttle:   single["throttle"],
				Quota:      single["quota"],
				Results:    single["results"],
				Warnings:   toStringSlice(single["warnings"]),
			})
		}
		return &epoerrors.CLIError{
			Code:    1,
			Type:    "GENERAL_ERROR",
			Message: fmt.Sprintf("%v", single["error"]),
		}
	}

	return outputSuccess(cmd, responsePayload{
		Service: "published-data",
		Results: batchResults,
	})
}

func extractPublishedSearchItems(parsed any) ([]any, bool) {
	root, ok := parsed.(map[string]any)
	if !ok {
		return nil, false
	}

	world := asMap(root["ops:world-patent-data"])
	search := asMap(world["ops:biblio-search"])
	result := asMap(search["ops:search-result"])

	if exchangeDocsRaw, ok := result["exchange-documents"]; ok {
		if docs, ok := asAnySlice(exchangeDocsRaw); ok {
			return docs, len(docs) > 0
		}
		if one, ok := exchangeDocsRaw.(map[string]any); ok {
			return []any{one}, true
		}
	}
	if exchangeDocRaw, ok := result["exchange-document"]; ok {
		if docs, ok := asAnySlice(exchangeDocRaw); ok {
			return docs, len(docs) > 0
		}
		if one, ok := exchangeDocRaw.(map[string]any); ok {
			return []any{one}, true
		}
	}

	itemsRaw, ok := result["ops:publication-reference"]
	if !ok {
		return nil, false
	}
	if items, ok := asAnySlice(itemsRaw); ok {
		return items, true
	}
	if one, ok := itemsRaw.(map[string]any); ok {
		return []any{one}, true
	}
	return nil, false
}

func normalizePubSearchSort(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "", "none":
		return pubSearchSortNone, nil
	case "pub-date-asc", "date-asc", "asc":
		return pubSearchSortDateAsc, nil
	case "pub-date-desc", "date-desc", "desc":
		return pubSearchSortDateDesc, nil
	default:
		return "", fmt.Errorf("unsupported sort mode %q", raw)
	}
}

func sortPublishedSearchItems(items []any, sortMode string) {
	sort.SliceStable(items, func(i, j int) bool {
		leftDate := publishedSearchDateKey(items[i])
		rightDate := publishedSearchDateKey(items[j])
		leftRef := publishedSearchReferenceKey(items[i])
		rightRef := publishedSearchReferenceKey(items[j])

		if leftDate == "" && rightDate == "" {
			return leftRef < rightRef
		}
		if leftDate == "" {
			return false
		}
		if rightDate == "" {
			return true
		}

		if sortMode == pubSearchSortDateAsc {
			if leftDate == rightDate {
				return leftRef < rightRef
			}
			return leftDate < rightDate
		}

		if leftDate == rightDate {
			return leftRef < rightRef
		}
		return leftDate > rightDate
	})
}

func flattenPublishedSearchItems(items []any) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, flattenPublishedSearchItem(item))
	}
	return rows
}

func flattenPublishedSearchItem(item any) map[string]any {
	row := map[string]any{
		"country":   "",
		"docNumber": "",
		"kind":      "",
		"pubDate":   "",
		"title":     "",
		"familyId":  "",
		"system":    "",
	}

	top := asAnyMap(item)
	exchange := asAnyMap(top["exchange-document"])
	if len(exchange) > 0 {
		top = exchange
	}

	if familyID := textValue(top["@family-id"]); familyID != "" {
		row["familyId"] = familyID
	}
	if system := textValue(top["@system"]); system != "" {
		row["system"] = system
	}

	docID := firstDocumentID(top["document-id"])
	if len(docID) == 0 {
		pubRef := asAnyMap(asAnyMap(top["bibliographic-data"])["publication-reference"])
		docID = firstDocumentID(pubRef["document-id"])
	}

	if country := textValue(asAnyMap(docID["country"])["$"]); country != "" {
		row["country"] = country
	}
	if docNumber := textValue(asAnyMap(docID["doc-number"])["$"]); docNumber != "" {
		row["docNumber"] = docNumber
	}
	if kind := textValue(asAnyMap(docID["kind"])["$"]); kind != "" {
		row["kind"] = kind
	}
	if pubDate := textValue(asAnyMap(docID["date"])["$"]); pubDate != "" {
		row["pubDate"] = pubDate
	}

	if row["country"] == "" {
		row["country"] = textValue(top["@country"])
	}
	if row["docNumber"] == "" {
		row["docNumber"] = textValue(top["@doc-number"])
	}
	if row["kind"] == "" {
		row["kind"] = textValue(top["@kind"])
	}
	if row["pubDate"] == "" {
		row["pubDate"] = textValue(top["@date"])
	}

	if title := firstTitleText(top["invention-title"]); title != "" {
		row["title"] = title
	} else if title := firstTitleText(asAnyMap(top["bibliographic-data"])["invention-title"]); title != "" {
		row["title"] = title
	}

	row["reference"] = fmt.Sprintf("%s%s%s", textValue(row["country"]), textValue(row["docNumber"]), textValue(row["kind"]))
	return row
}

func publishedSearchDateKey(item any) string {
	return textValue(flattenPublishedSearchItem(item)["pubDate"])
}

func publishedSearchReferenceKey(item any) string {
	return textValue(flattenPublishedSearchItem(item)["reference"])
}

func toStringSlice(v any) []string {
	items, ok := v.([]string)
	if ok {
		return items
	}
	return nil
}

func executeOPSRequest(ctx context.Context, request api.Request) (api.Response, error) {
	return executeOPSRequestWithBase(ctx, request, "")
}

func executeOPSRequestWithBase(ctx context.Context, request api.Request, baseURL string) (api.Response, error) {
	clientID, clientSecret, _, _, err := resolveRuntimeCredentials()
	if err != nil {
		return api.Response{}, err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(flagTimeout)*time.Second)
	defer cancel()

	httpClient := &http.Client{Timeout: time.Duration(flagTimeout) * time.Second}
	tokenManager := auth.NewTokenManager(httpClient, clientID, clientSecret, 2*time.Minute)
	opsClient := api.NewClient(httpClient, tokenManager)
	if strings.TrimSpace(baseURL) != "" {
		opsClient.SetBaseURL(baseURL)
	}
	return opsClient.Do(ctx, request)
}

func outputPublishedResponse(cmd *cobra.Command, requestMeta map[string]any, resp api.Response) error {
	return outputOPSResponse(cmd, "published-data", requestMeta, resp, parseSearchPagination)
}

func isOneOf(value string, options ...string) bool {
	normalized := strings.TrimSpace(value)
	for _, option := range options {
		if normalized == option {
			return true
		}
	}
	return false
}

func parseJSONBody(body []byte) (any, []string) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return map[string]any{}, nil
	}

	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return map[string]any{
			"raw": trimmed,
		}, []string{"response was not valid JSON; returned raw body"}
	}
	return parsed, nil
}

func parsePagination(rangeHeader string) map[string]any {
	rangeHeader = strings.TrimSpace(rangeHeader)
	if rangeHeader == "" {
		return nil
	}

	pieces := strings.SplitN(rangeHeader, "-", 2)
	if len(pieces) != 2 {
		return map[string]any{"range": rangeHeader}
	}

	start, errStart := strconv.Atoi(strings.TrimSpace(pieces[0]))
	end, errEnd := strconv.Atoi(strings.TrimSpace(pieces[1]))
	if errStart != nil || errEnd != nil || end < start {
		return map[string]any{"range": rangeHeader}
	}

	return map[string]any{
		"offset": start,
		"limit":  end - start + 1,
	}
}

func compactQuery(query url.Values) map[string]any {
	if len(query) == 0 {
		return nil
	}
	out := make(map[string]any, len(query))
	for key, values := range query {
		if len(values) == 1 {
			out[key] = values[0]
			continue
		}
		if len(values) > 1 {
			out[key] = values
		}
	}
	return out
}

type paginationParser func(results any) map[string]any

func outputOPSResponse(cmd *cobra.Command, service string, requestMeta map[string]any, resp api.Response, pager paginationParser) error {
	results, warnings := parseJSONBody(resp.Body)
	rangeHeader := strings.TrimSpace(resp.Headers.Get("X-OPS-Range"))
	if rangeHeader != "" {
		requestMeta["range"] = rangeHeader
	}

	pagination := parsePagination(rangeHeader)
	if pager != nil {
		pagination = mergePagination(pagination, pager(results))
	}

	payload := responsePayload{
		Service:    service,
		Request:    requestMeta,
		Pagination: pagination,
		Throttle: map[string]any{
			"system":   resp.Metadata.Throttle.System,
			"services": resp.Metadata.Throttle.Services,
		},
		Quota: map[string]int{
			"hourUsed":       resp.Metadata.Quota.IndividualPerHourUsed,
			"weekUsed":       resp.Metadata.Quota.RegisteredPerWeekUsed,
			"payingWeekUsed": resp.Metadata.Quota.RegisteredPayingPerWeekUsed,
		},
		Results:  results,
		Warnings: warnings,
	}
	return outputSuccess(cmd, payload)
}

func parseSearchPagination(results any) map[string]any {
	root, ok := results.(map[string]any)
	if !ok {
		return nil
	}
	worldPatentData := asMap(root["ops:world-patent-data"])
	biblioSearch := asMap(worldPatentData["ops:biblio-search"])
	if len(biblioSearch) == 0 {
		return nil
	}

	out := map[string]any{}
	if total := atoiAny(biblioSearch["@total-result-count"]); total > 0 {
		out["total"] = total
	}

	rangeMap := asMap(biblioSearch["ops:range"])
	begin := atoiAny(rangeMap["@begin"])
	end := atoiAny(rangeMap["@end"])
	if begin > 0 && end >= begin {
		out["offset"] = begin
		out["limit"] = end - begin + 1
		if total, ok := out["total"].(int); ok && total > 0 {
			out["hasMore"] = end < total
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func mergePagination(primary, secondary map[string]any) map[string]any {
	if len(primary) == 0 {
		return secondary
	}
	if len(secondary) == 0 {
		return primary
	}
	out := map[string]any{}
	for key, value := range primary {
		out[key] = value
	}
	for key, value := range secondary {
		out[key] = value
	}
	return out
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func atoiAny(v any) int {
	switch t := v.(type) {
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int(t)
	case int:
		return t
	default:
		return 0
	}
}
