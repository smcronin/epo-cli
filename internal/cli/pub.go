package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/smcronin/epo-cli/internal/api"
	"github.com/smcronin/epo-cli/internal/auth"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newPubCmd() *cobra.Command {
	pubCmd := &cobra.Command{
		Use:   "pub",
		Short: "Published-data service operations",
	}

	pubCmd.AddCommand(newPubBiblioCmd())
	pubCmd.AddCommand(newPubSimpleEndpointCmd("abstract", "abstract", "Fetch abstract text/metadata for a published reference"))
	pubCmd.AddCommand(newPubSimpleEndpointCmd("claims", "claims", "Fetch claims for a published reference"))
	pubCmd.AddCommand(newPubSimpleEndpointCmd("description", "description", "Fetch description fulltext for a published reference"))
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reference := strings.TrimSpace(args[0])
			if reference == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "reference is required",
				}
			}
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
			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputPublishedResponse(cmd, requestMeta, resp)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reference := strings.TrimSpace(args[0])
			if reference == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "reference is required",
				}
			}
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

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputPublishedResponse(cmd, requestMeta, resp)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reference := strings.TrimSpace(args[0])
			if reference == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "reference is required",
				}
			}
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

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputPublishedResponse(cmd, requestMeta, resp)
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
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Run a CQL search against published-data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(query) == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "missing CQL query",
					Hint:    "Pass --q \"applicant=IBM\"",
				}
			}

			path := "/published-data/search"
			if v := strings.TrimSpace(constituents); v != "" {
				path += "/" + v
			}

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
				request.Body = []byte("q=" + url.QueryEscape(query))
			} else {
				request.Query = url.Values{}
				request.Query.Set("q", query)
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
				return err
			}
			return outputPublishedResponse(cmd, requestMeta, resp)
		},
	}

	cmd.Flags().StringVar(&query, "q", "", "CQL query (for example: applicant=IBM)")
	cmd.Flags().StringVar(&constituents, "constituents", "", "Search constituents (for example: biblio,abstract,full-cycle)")
	cmd.Flags().StringVar(&rangeHeader, "range", "", "Result range, for example 1-25")
	cmd.Flags().BoolVar(&usePost, "post", false, "Use POST instead of GET")
	return cmd
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
