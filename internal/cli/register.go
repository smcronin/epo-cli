package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newRegisterCmd() *cobra.Command {
	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "EP Register service operations",
	}
	registerCmd.AddCommand(newRegisterGetCmd())
	registerCmd.AddCommand(newRegisterSimpleCmd("events", "events", "Fetch EP register events"))
	registerCmd.AddCommand(newRegisterSimpleCmd("procedural-steps", "procedural-steps", "Fetch EP register procedural steps"))
	registerCmd.AddCommand(newRegisterUPPCmd())
	registerCmd.AddCommand(newRegisterSearchCmd())
	return registerCmd
}

func newRegisterGetCmd() *cobra.Command {
	var constituents string

	cmd := &cobra.Command{
		Use:   "get <reference>",
		Short: "Fetch EP register data for an application reference",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runOPSBatch(cmd, "register", references, func(reference string) (api.Request, map[string]any, error) {
				path := fmt.Sprintf("/register/application/epodoc/%s", reference)
				if v := strings.TrimSpace(constituents); v != "" {
					path += "/" + v
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
			}, nil)
		},
	}

	cmd.Flags().StringVar(&constituents, "constituents", "", "Optional constituents: biblio,events,procedural-steps")
	return cmd
}

func newRegisterSimpleCmd(name, endpoint, shortDesc string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <reference>", name),
		Short: shortDesc,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runOPSBatch(cmd, "register", references, func(reference string) (api.Request, map[string]any, error) {
				path := fmt.Sprintf("/register/application/epodoc/%s/%s", reference, endpoint)
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
			}, nil)
		},
	}
	return cmd
}

func newRegisterUPPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upp <reference>",
		Short: "Fetch unitary patent protection (UPP) data for an EP publication reference",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runOPSBatch(cmd, "register", references, func(reference string) (api.Request, map[string]any, error) {
				path := fmt.Sprintf("/register/publication/epodoc/%s/upp", reference)
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
			}, nil)
		},
	}
	return cmd
}

func newRegisterSearchCmd() *cobra.Command {
	var (
		query       string
		rangeHeader string
		usePost     bool
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Run CQL search against EP Register",
		RunE: func(cmd *cobra.Command, args []string) error {
			queries, err := resolveQueryOrStdinInputs(query)
			if err != nil {
				return err
			}
			if flagAll {
				return runRegisterSearchAll(cmd, queries, rangeHeader, usePost)
			}

			return runOPSBatch(cmd, "register", queries, func(inputQuery string) (api.Request, map[string]any, error) {
				request := api.Request{
					Method: http.MethodGet,
					Path:   "/register/search",
					Accept: "application/json",
				}
				if v := strings.TrimSpace(rangeHeader); v != "" {
					request.Headers = map[string]string{
						"Range": v,
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
			}, nil)
		},
	}

	cmd.Flags().StringVar(&query, "q", "", "CQL query (for example: pa=IBM)")
	cmd.Flags().StringVar(&rangeHeader, "range", "", "Result range, for example 1-25")
	cmd.Flags().BoolVar(&usePost, "post", false, "Use POST instead of GET")
	return cmd
}

func runRegisterSearchAll(cmd *cobra.Command, queries []string, rangeHeader string, usePost bool) error {
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

	batchResults := make([]map[string]any, 0, len(queries))
	for _, q := range queries {
		currentStart := start
		currentEnd := end
		pages := 0
		allItems := make([]any, 0)
		combinedWarnings := make([]string, 0)
		var throttleSnapshot any
		var quotaSnapshot any

		for {
			request := api.Request{
				Method: http.MethodGet,
				Path:   "/register/search",
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
			items, _ := extractRegisterSearchItems(parsed)
			allItems = append(allItems, items...)

			if len(items) < pageSize {
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
						"pagesFetched": pages,
						"returned":     len(allItems),
					},
					"throttle": throttleSnapshot,
					"quota":    quotaSnapshot,
					"results":  allItems,
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
				Service:    "register",
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
		Service: "register",
		Results: batchResults,
	})
}

func extractRegisterSearchItems(parsed any) ([]any, bool) {
	root, ok := parsed.(map[string]any)
	if !ok {
		return nil, false
	}

	world := asMap(root["ops:world-patent-data"])
	search := asMap(world["ops:register-search"])
	documents := asMap(search["reg:register-documents"])
	itemsRaw, ok := documents["reg:register-document"]
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
