package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
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
	var (
		constituents string
		summaryMode  bool
	)

	cmd := &cobra.Command{
		Use:   "get <reference>",
		Short: "Fetch EP register data for an application reference",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			if summaryMode {
				return runRegisterGetSummary(cmd, references, constituents)
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
	cmd.Flags().BoolVar(&summaryMode, "summary", false, "Return compact prosecution summary")
	return cmd
}

func newRegisterSimpleCmd(name, endpoint, shortDesc string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <reference>", name),
		Short: shortDesc + " (expects application reference in epodoc format, e.g. EP99203729)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runOPSBatch(cmd, "register", references, func(reference string) (api.Request, map[string]any, error) {
				if isLikelyPublicationRef(reference) {
					return api.Request{}, nil, &epoerrors.CLIError{
						Code:    400,
						Type:    "VALIDATION_ERROR",
						Message: "register events/procedural-steps require an application reference in epodoc format",
						Hint:    "Use an application number like EP99203729 (publication references such as EP.1000000.A1 are not accepted by this endpoint)",
					}
				}
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
				if looksApplicationEpRef(reference) {
					path = fmt.Sprintf("/register/application/epodoc/%s/upp", reference)
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
	return cmd
}

var (
	publicationDocdbPattern  = regexp.MustCompile(`^[A-Z]{2}\.[0-9]+\.[A-Z][0-9]?$`)
	publicationEpodocPattern = regexp.MustCompile(`^[A-Z]{2}[0-9]+[A-Z][0-9]?$`)
	applicationEpodocPattern = regexp.MustCompile(`^[A-Z]{2}[0-9]{6,}$`)
)

func isLikelyPublicationRef(reference string) bool {
	ref := strings.ToUpper(strings.TrimSpace(reference))
	return publicationDocdbPattern.MatchString(ref) || publicationEpodocPattern.MatchString(ref)
}

func looksApplicationEpRef(reference string) bool {
	ref := strings.ToUpper(strings.TrimSpace(reference))
	return applicationEpodocPattern.MatchString(ref)
}

func runRegisterGetSummary(cmd *cobra.Command, references []string, constituents string) error {
	results := make([]map[string]any, 0, len(references))
	for _, reference := range references {
		path := fmt.Sprintf("/register/application/epodoc/%s", reference)
		if v := strings.TrimSpace(constituents); v != "" {
			path += "/" + v
		}
		request := api.Request{
			Method: http.MethodGet,
			Path:   path,
			Accept: "application/json",
		}
		resp, err := executeOPSRequest(cmd.Context(), request)
		if err != nil {
			results = append(results, map[string]any{
				"input": reference,
				"ok":    false,
				"error": mapError(err),
			})
			continue
		}
		parsed, warnings := parseJSONBody(resp.Body)
		results = append(results, map[string]any{
			"input":    reference,
			"ok":       true,
			"summary":  summarizeRegisterPayload(parsed),
			"warnings": warnings,
		})
	}

	if len(results) == 1 {
		single := results[0]
		if ok, _ := single["ok"].(bool); ok {
			return outputSuccess(cmd, responsePayload{
				Service: "register",
				Results: single["summary"],
				Warnings: func() []string {
					return toStringSlice(single["warnings"])
				}(),
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
		Results: results,
	})
}

func summarizeRegisterPayload(v any) map[string]any {
	summary := map[string]any{
		"status":           "",
		"application":      "",
		"publication":      "",
		"designatedStates": []string{},
		"lapseList":        []string{},
		"keyDates":         map[string]string{},
	}

	registerDoc := firstRegisterDocument(v)
	if len(registerDoc) == 0 {
		return summary
	}

	status := firstNonEmpty(
		textValue(registerDoc["@status"]),
		textValue(asAnyMap(asAnyMap(registerDoc["reg:ep-patent-statuses"])["reg:ep-patent-status"])["$"]),
	)
	if status != "" {
		summary["status"] = status
	}

	biblio := asAnyMap(registerDoc["reg:bibliographic-data"])
	keyDates := summary["keyDates"].(map[string]string)

	appDoc := asAnyMap(asAnyMap(biblio["reg:application-reference"])["reg:document-id"])
	appCountry := textValue(appDoc["reg:country"])
	appDocNumber := textValue(appDoc["reg:doc-number"])
	if appCountry != "" || appDocNumber != "" {
		summary["application"] = strings.ToUpper(strings.TrimSpace(appCountry + appDocNumber))
	}
	if filingDate := textValue(appDoc["reg:date"]); filingDate != "" {
		keyDates["filingDate"] = filingDate
	}

	pubDoc := registerPublicationDocumentID(biblio)
	pubCountry := textValue(pubDoc["reg:country"])
	pubDocNumber := textValue(pubDoc["reg:doc-number"])
	pubKind := textValue(pubDoc["reg:kind"])
	if pubCountry != "" || pubDocNumber != "" || pubKind != "" {
		summary["publication"] = strings.ToUpper(strings.TrimSpace(pubCountry + pubDocNumber + pubKind))
	}
	if pubDate := textValue(pubDoc["reg:date"]); pubDate != "" {
		keyDates["publicationDate"] = pubDate
	}

	rightsEffective := asAnyMap(biblio["reg:dates-rights-effective"])
	if firstExam := textValue(asAnyMap(rightsEffective["reg:first-examination-report-despatched"])["reg:date"]); firstExam != "" {
		keyDates["firstExaminationReportDate"] = firstExam
	}
	if reqExam := textValue(asAnyMap(rightsEffective["reg:request-for-examination"])["reg:date"]); reqExam != "" {
		keyDates["requestForExaminationDate"] = reqExam
	}
	if oppositionNotFiled := textValue(asAnyMap(asAnyMap(biblio["reg:opposition-data"])["reg:opposition-not-filed"])["reg:date"]); oppositionNotFiled != "" {
		keyDates["oppositionNotFiledDate"] = oppositionNotFiled
	}

	designatedStates := extractRegisterDesignatedStates(biblio)
	if len(designatedStates) > 0 {
		summary["designatedStates"] = designatedStates
	}

	lapseList := extractRegisterLapseCountries(biblio)
	if len(lapseList) > 0 {
		summary["lapseList"] = lapseList
	}

	return summary
}

func firstRegisterDocument(v any) map[string]any {
	world := asAnyMap(asAnyMap(v)["ops:world-patent-data"])
	if len(world) == 0 {
		return map[string]any{}
	}
	search := asAnyMap(world["ops:register-search"])
	if len(search) == 0 {
		return map[string]any{}
	}
	docs := asAnyMap(search["reg:register-documents"])
	if len(docs) == 0 {
		return map[string]any{}
	}
	if doc := asAnyMap(docs["reg:register-document"]); len(doc) > 0 {
		return doc
	}
	if docList, ok := asAnySlice(docs["reg:register-document"]); ok {
		for _, item := range docList {
			if doc := asAnyMap(item); len(doc) > 0 {
				return doc
			}
		}
	}
	return map[string]any{}
}

func registerPublicationDocumentID(biblio map[string]any) map[string]any {
	refs := asAnySliceOrSingleton(biblio["reg:publication-reference"])
	if len(refs) == 0 {
		return map[string]any{}
	}
	best := map[string]any{}
	for _, raw := range refs {
		doc := asAnyMap(asAnyMap(raw)["reg:document-id"])
		if len(doc) == 0 {
			continue
		}
		if best == nil || len(best) == 0 {
			best = doc
		}
		if strings.EqualFold(textValue(doc["reg:country"]), "EP") {
			return doc
		}
	}
	return best
}

func extractRegisterDesignatedStates(biblio map[string]any) []string {
	designation := asAnyMap(biblio["reg:designation-of-states"])
	if len(designation) == 0 {
		return nil
	}
	states := map[string]struct{}{}
	collectCountryCodes(designation, states)
	delete(states, "EP")
	return sortedSet(states)
}

func extractRegisterLapseCountries(biblio map[string]any) []string {
	termOfGrant := asAnySliceOrSingleton(biblio["reg:term-of-grant"])
	if len(termOfGrant) == 0 {
		return nil
	}
	out := map[string]struct{}{}
	for _, rawTerm := range termOfGrant {
		term := asAnyMap(rawTerm)
		lapses := asAnySliceOrSingleton(term["reg:lapsed-in-country"])
		for _, rawLapse := range lapses {
			lapse := asAnyMap(rawLapse)
			country := strings.ToUpper(strings.TrimSpace(textValue(lapse["reg:country"])))
			if isLikelyCountryCode(country) {
				out[country] = struct{}{}
			}
		}
	}
	return sortedSet(out)
}

func collectCountryCodes(v any, out map[string]struct{}) {
	switch t := v.(type) {
	case map[string]any:
		for key, child := range t {
			if localXMLKey(key) == "country" {
				collectCountryCodeValues(child, out)
			}
			collectCountryCodes(child, out)
		}
	case []any:
		for _, child := range t {
			collectCountryCodes(child, out)
		}
	}
}

func collectCountryCodeValues(v any, out map[string]struct{}) {
	switch t := v.(type) {
	case string:
		country := strings.ToUpper(strings.TrimSpace(t))
		if isLikelyCountryCode(country) {
			out[country] = struct{}{}
		}
	case map[string]any:
		if raw, ok := t["$"]; ok {
			collectCountryCodeValues(raw, out)
		}
		for _, child := range t {
			collectCountryCodeValues(child, out)
		}
	case []any:
		for _, child := range t {
			collectCountryCodeValues(child, out)
		}
	}
}

func isLikelyCountryCode(value string) bool {
	if len(value) != 2 {
		return false
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func sortedSet(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func collectStringValuesByKey(v any, key string, out map[string]struct{}) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if k == key {
				value := textValue(child)
				if value != "" {
					out[value] = struct{}{}
				}
			}
			collectStringValuesByKey(child, key, out)
		}
	case []any:
		for _, child := range t {
			collectStringValuesByKey(child, key, out)
		}
	}
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
