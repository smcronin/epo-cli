package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newCPCCmd() *cobra.Command {
	cpcCmd := &cobra.Command{
		Use:   "cpc",
		Short: "Classification (CPC) service operations",
	}
	cpcCmd.AddCommand(newCPCGetCmd())
	cpcCmd.AddCommand(newCPCSearchCmd())
	cpcCmd.AddCommand(newCPCMediaCmd())
	cpcCmd.AddCommand(newCPCMapCmd())
	return cpcCmd
}

func newCPCGetCmd() *cobra.Command {
	var (
		depth      string
		navigation bool
		ancestors  bool
		accept     string
		normalize  bool
	)

	cmd := &cobra.Command{
		Use:   "get <symbol>",
		Short: "Retrieve CPC symbol details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol := strings.TrimSpace(args[0])
			if symbol == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "symbol is required",
				}
			}

			path := "/classification/cpc/" + symbol
			query := url.Values{}
			if v := strings.TrimSpace(depth); v != "" {
				query.Set("depth", v)
			}
			if navigation {
				query.Set("navigation", "true")
			}
			if ancestors {
				query.Set("ancestors", "true")
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   path,
				Query:  query,
				Accept: defaultIfEmptyString(accept, "application/cpc+xml"),
			}
			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"query":  compactQuery(request.Query),
				"accept": request.Accept,
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputCPCResponse(cmd, requestMeta, resp, normalize, "get", symbol, "", "")
		},
	}

	cmd.Flags().StringVar(&depth, "depth", "", "Optional depth (for example: 1 or all)")
	cmd.Flags().BoolVar(&navigation, "navigation", false, "Include previous/next navigation nodes")
	cmd.Flags().BoolVar(&ancestors, "ancestors", false, "Include ancestor nodes")
	cmd.Flags().StringVar(&accept, "accept", "application/cpc+xml", "Accept header")
	cmd.Flags().BoolVar(&normalize, "normalize", false, "Parse XML and return structured symbol/title fields")
	cmd.Flags().BoolVar(&normalize, "parsed", false, "Alias for --normalize")
	return cmd
}

func newCPCSearchCmd() *cobra.Command {
	var (
		query       string
		rangeHeader string
		accept      string
		normalize   bool
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search CPC classes by keyword",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(query) == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "missing search query",
					Hint:    "Pass --q \"chemistry\"",
				}
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   "/classification/cpc/search/",
				Query:  url.Values{"q": []string{query}},
				Accept: defaultIfEmptyString(accept, "application/cpc+xml"),
			}
			if v := strings.TrimSpace(rangeHeader); v != "" {
				request.Query.Set("Range", v)
			}

			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"query":  compactQuery(request.Query),
				"accept": request.Accept,
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputCPCResponse(cmd, requestMeta, resp, normalize, "search", "", "", "")
		},
	}

	cmd.Flags().StringVar(&query, "q", "", "Search keyword")
	cmd.Flags().StringVar(&rangeHeader, "range", "", "Range window (for example 1-20)")
	cmd.Flags().StringVar(&accept, "accept", "application/cpc+xml", "Accept header")
	cmd.Flags().BoolVar(&normalize, "normalize", false, "Parse XML and return structured rows")
	cmd.Flags().BoolVar(&normalize, "parsed", false, "Alias for --normalize")
	return cmd
}

func newCPCMediaCmd() *cobra.Command {
	var (
		accept      string
		outputPath  string
		includeBody bool
	)

	cmd := &cobra.Command{
		Use:   "media <media-id>",
		Short: "Fetch CPC media asset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mediaID := strings.TrimSpace(args[0])
			if mediaID == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "media-id is required",
				}
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   "/classification/cpc/media/" + mediaID,
				Accept: defaultIfEmptyString(accept, "image/gif"),
			}
			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"accept": request.Accept,
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}

			results := map[string]any{
				"contentType": resp.Headers.Get("Content-Type"),
				"bytes":       len(resp.Body),
				"sha256":      sha256Hex(resp.Body),
			}
			warnings := []string{}

			if strings.TrimSpace(outputPath) != "" {
				cleanPath := filepath.Clean(outputPath)
				if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
					return fmt.Errorf("create output directory: %w", err)
				}
				if err := os.WriteFile(cleanPath, resp.Body, 0o644); err != nil {
					return fmt.Errorf("write media file: %w", err)
				}
				results["saved"] = true
				results["outputPath"] = cleanPath
			}

			if includeBody {
				results["base64"] = base64.StdEncoding.EncodeToString(resp.Body)
			} else if strings.TrimSpace(outputPath) == "" {
				warnings = append(warnings, "Body omitted; use --include-body or --out to retain content.")
			}

			payload := responsePayload{
				Service: "classification/cpc",
				Request: requestMeta,
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
		},
	}

	cmd.Flags().StringVar(&accept, "accept", "image/gif", "Accept header for media request")
	cmd.Flags().StringVar(&outputPath, "out", "", "Optional output file path for binary content")
	cmd.Flags().BoolVar(&includeBody, "include-body", false, "Include base64 body in JSON output")
	return cmd
}

func newCPCMapCmd() *cobra.Command {
	var (
		from       string
		to         string
		additional bool
		accept     string
		normalize  bool
	)

	cmd := &cobra.Command{
		Use:   "map <symbol>",
		Short: "Map CPC/ECLA/IPC classification symbols",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol := strings.TrimSpace(args[0])
			if symbol == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "symbol is required",
				}
			}
			if !isOneOf(from, "ecla", "cpc") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --from %q", from),
					Hint:    "Use ecla or cpc",
				}
			}
			if !isOneOf(to, "cpc", "ecla", "ipc") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --to %q", to),
					Hint:    "Use cpc, ecla, or ipc",
				}
			}

			path := fmt.Sprintf("/classification/map/%s/%s/%s", from, symbol, to)
			query := url.Values{}
			if additional {
				query.Set("additional", "true")
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   path,
				Query:  query,
				Accept: defaultIfEmptyString(accept, "application/cpc+xml"),
			}
			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"query":  compactQuery(request.Query),
				"accept": request.Accept,
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputCPCResponse(cmd, requestMeta, resp, normalize, "map", symbol, from, to)
		},
	}

	cmd.Flags().StringVar(&from, "from", "cpc", "Input scheme: ecla or cpc")
	cmd.Flags().StringVar(&to, "to", "ecla", "Output scheme: cpc, ecla, or ipc")
	cmd.Flags().BoolVar(&additional, "additional", false, "Include additional mapping context when supported (some symbols may return no additional differences)")
	cmd.Flags().StringVar(&accept, "accept", "application/cpc+xml", "Accept header")
	cmd.Flags().BoolVar(&normalize, "normalize", false, "Parse XML and return {from,fromScheme,to,toScheme} mappings")
	cmd.Flags().BoolVar(&normalize, "parsed", false, "Alias for --normalize")
	return cmd
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func defaultIfEmptyString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func outputCPCResponse(cmd *cobra.Command, requestMeta map[string]any, resp api.Response, normalize bool, mode, symbol, from, to string) error {
	if !normalize {
		return outputOPSResponse(cmd, "classification/cpc", requestMeta, resp, nil)
	}

	parsed, warnings := parseJSONBody(resp.Body)
	normalized := normalizeCPCPayload(mode, symbol, from, to, parsed, resp.Body)
	warningsOut := append([]string{}, warnings...)
	if len(normalized) == 0 {
		normalized = []map[string]any{
			{"warning": "No structured CPC rows extracted from response"},
		}
		warningsOut = append(warningsOut, "Use --format json without --normalize to inspect full payload.")
	}
	return outputSuccess(cmd, responsePayload{
		Service: "classification/cpc",
		Request: requestMeta,
		Throttle: map[string]any{
			"system":   resp.Metadata.Throttle.System,
			"services": resp.Metadata.Throttle.Services,
		},
		Quota: map[string]int{
			"hourUsed":       resp.Metadata.Quota.IndividualPerHourUsed,
			"weekUsed":       resp.Metadata.Quota.RegisteredPerWeekUsed,
			"payingWeekUsed": resp.Metadata.Quota.RegisteredPayingPerWeekUsed,
		},
		Results:  normalized,
		Warnings: warningsOut,
	})
}

func normalizeCPCPayload(mode, symbol, from, to string, parsed any, body []byte) []map[string]any {
	var rows []map[string]any
	switch mode {
	case "map":
		rows = extractCPCMapRowsFromParsed(parsed, symbol, from, to)
	case "search":
		rows = extractCPCSearchRowsFromParsed(parsed)
	default:
		rows = extractCPCGetRowsFromParsed(parsed)
	}
	if len(rows) > 0 {
		return rows
	}

	// Fallback to direct XML extraction for minimal structures.
	symbols := extractXMLValuesByLocalName(body, "classification-symbol")
	if len(symbols) == 0 {
		symbols = extractXMLValuesByLocalName(body, "symbol")
	}
	titles := extractXMLValuesByLocalName(body, "class-title")
	if len(titles) == 0 {
		titles = extractXMLValuesByLocalName(body, "title")
	}
	percentages := extractXMLValuesByLocalName(body, "score")
	if len(percentages) == 0 {
		percentages = extractXMLValuesByLocalName(body, "relevance")
	}

	switch mode {
	case "map":
		return buildCPCMapRows(symbol, from, to, symbols)
	case "search":
		return buildCPCSearchRows(symbols, titles, percentages)
	default:
		return buildCPCGetRows(symbols, titles)
	}
}

func extractCPCWorldPatentData(parsed any) map[string]any {
	root := asAnyMap(parsed)
	if len(root) == 0 {
		return map[string]any{}
	}
	for _, key := range []string{"world-patent-data", "ops:world-patent-data"} {
		if world := asAnyMap(root[key]); len(world) > 0 {
			return world
		}
	}
	return map[string]any{}
}

func asAnySliceOrSingleton(v any) []any {
	if items, ok := asAnySlice(v); ok {
		return items
	}
	if m := asAnyMap(v); len(m) > 0 {
		return []any{m}
	}
	return nil
}

func cpcTextFromNode(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case map[string]any:
		if direct := textValue(t["$"]); direct != "" {
			return direct
		}
		if text := cpcTextFromNode(t["text"]); text != "" {
			return text
		}
		if text := cpcTextFromNode(t["comment"]); text != "" {
			return text
		}
		if text := cpcTextFromNode(t["title-part"]); text != "" {
			return text
		}
		for _, child := range t {
			if text := cpcTextFromNode(child); text != "" {
				return text
			}
		}
	case []any:
		for _, child := range t {
			if text := cpcTextFromNode(child); text != "" {
				return text
			}
		}
	}
	return ""
}

func extractCPCSearchRowsFromParsed(parsed any) []map[string]any {
	world := extractCPCWorldPatentData(parsed)
	if len(world) == 0 {
		return nil
	}
	search := asAnyMap(world["classification-search"])
	if len(search) == 0 {
		search = asAnyMap(world["ops:classification-search"])
	}
	if len(search) == 0 {
		return nil
	}
	searchResult := asAnyMap(search["search-result"])
	if len(searchResult) == 0 {
		searchResult = asAnyMap(search["ops:search-result"])
	}
	stats := asAnySliceOrSingleton(searchResult["classification-statistics"])
	if len(stats) == 0 {
		stats = asAnySliceOrSingleton(searchResult["ops:classification-statistics"])
	}
	if len(stats) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(stats))
	for _, raw := range stats {
		stat := asAnyMap(raw)
		if len(stat) == 0 {
			continue
		}
		row := map[string]any{}
		if symbol := firstNonEmpty(textValue(stat["@classification-symbol"]), textValue(stat["classification-symbol"])); symbol != "" {
			row["symbol"] = symbol
		}
		if title := cpcTextFromNode(stat["class-title"]); title != "" {
			row["title"] = title
		}
		percentageRaw := firstNonEmpty(textValue(stat["@percentage"]), textValue(stat["score"]), textValue(stat["relevance"]))
		if percentageRaw != "" {
			if f, err := strconv.ParseFloat(strings.TrimSpace(percentageRaw), 64); err == nil {
				row["percentage"] = f
			} else {
				row["percentage"] = percentageRaw
			}
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func extractCPCGetRowsFromParsed(parsed any) []map[string]any {
	world := extractCPCWorldPatentData(parsed)
	if len(world) == 0 {
		return nil
	}
	classificationScheme := asAnyMap(world["classification-scheme"])
	if len(classificationScheme) == 0 {
		classificationScheme = asAnyMap(world["ops:classification-scheme"])
	}
	if len(classificationScheme) == 0 {
		return nil
	}
	cpc := asAnyMap(classificationScheme["cpc"])
	classScheme := asAnyMap(cpc["class-scheme"])
	rootItems := classScheme["classification-item"]
	if len(asAnySliceOrSingleton(rootItems)) == 0 {
		rootItems = classificationScheme["classification-item"]
	}
	items := asAnySliceOrSingleton(rootItems)
	if len(items) == 0 {
		return nil
	}

	rows := []map[string]any{}
	seen := map[string]struct{}{}
	var walk func(any)
	walk = func(node any) {
		for _, raw := range asAnySliceOrSingleton(node) {
			item := asAnyMap(raw)
			if len(item) == 0 {
				continue
			}
			symbol := firstNonEmpty(
				textValue(asAnyMap(item["classification-symbol"])["$"]),
				textValue(item["@classification-symbol"]),
				textValue(item["@sort-key"]),
			)
			title := cpcTextFromNode(item["class-title"])
			if symbol != "" || title != "" {
				key := strings.ToUpper(symbol) + "|" + title
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					row := map[string]any{}
					if symbol != "" {
						row["symbol"] = symbol
					}
					if title != "" {
						row["title"] = title
					}
					rows = append(rows, row)
				}
			}
			if child := item["classification-item"]; child != nil {
				walk(child)
			}
		}
	}
	walk(items)
	return rows
}

func extractCPCMapRowsFromParsed(parsed any, sourceSymbol, from, to string) []map[string]any {
	world := extractCPCWorldPatentData(parsed)
	if len(world) == 0 {
		return nil
	}
	classificationScheme := asAnyMap(world["classification-scheme"])
	if len(classificationScheme) == 0 {
		classificationScheme = asAnyMap(world["ops:classification-scheme"])
	}
	mappings := asAnyMap(classificationScheme["mappings"])
	if len(mappings) == 0 {
		return nil
	}
	mappingItems := asAnySliceOrSingleton(mappings["mapping"])
	if len(mappingItems) == 0 {
		return nil
	}

	inScheme := firstNonEmpty(textValue(mappings["@inputSchema"]), strings.ToUpper(strings.TrimSpace(from)))
	outScheme := firstNonEmpty(textValue(mappings["@outputSchema"]), strings.ToUpper(strings.TrimSpace(to)))
	fromKey := strings.ToLower(strings.TrimSpace(from))
	if fromKey == "" {
		fromKey = strings.ToLower(inScheme)
	}
	toKey := strings.ToLower(strings.TrimSpace(to))
	if toKey == "" {
		toKey = strings.ToLower(outScheme)
	}

	rows := make([]map[string]any, 0, len(mappingItems))
	seen := map[string]struct{}{}
	for _, raw := range mappingItems {
		item := asAnyMap(raw)
		if len(item) == 0 {
			continue
		}
		fromSymbol := firstNonEmpty(textValue(asAnyMap(item[fromKey])["$"]), strings.ToUpper(strings.TrimSpace(sourceSymbol)))
		toSymbol := textValue(asAnyMap(item[toKey])["$"])
		if toSymbol == "" {
			toSymbol = firstNonEmpty(
				textValue(item["@classification-symbol"]),
				textValue(item["classification-symbol"]),
			)
		}
		toSymbol = strings.TrimSpace(toSymbol)
		if toSymbol == "" {
			continue
		}
		key := strings.ToUpper(fromSymbol) + "|" + strings.ToUpper(toSymbol) + "|" + strings.ToUpper(inScheme) + "|" + strings.ToUpper(outScheme)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		rows = append(rows, map[string]any{
			"from":       strings.ToUpper(strings.TrimSpace(fromSymbol)),
			"fromScheme": strings.ToUpper(strings.TrimSpace(inScheme)),
			"to":         strings.ToUpper(strings.TrimSpace(toSymbol)),
			"toScheme":   strings.ToUpper(strings.TrimSpace(outScheme)),
		})
	}
	return rows
}

func extractXMLValuesByLocalName(body []byte, localName string) []string {
	localName = strings.ToLower(strings.TrimSpace(localName))
	if localName == "" {
		return nil
	}

	decoder := xml.NewDecoder(bytes.NewReader(body))
	values := []string{}
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if strings.ToLower(start.Name.Local) != localName {
			continue
		}
		var value string
		if decodeErr := decoder.DecodeElement(&value, &start); decodeErr != nil {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func buildCPCGetRows(symbols, titles []string) []map[string]any {
	max := len(symbols)
	if len(titles) > max {
		max = len(titles)
	}
	if max == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, max)
	for i := 0; i < max; i++ {
		row := map[string]any{}
		if i < len(symbols) {
			row["symbol"] = symbols[i]
		}
		if i < len(titles) {
			row["title"] = titles[i]
		}
		rows = append(rows, row)
	}
	return rows
}

func buildCPCSearchRows(symbols, titles, percentages []string) []map[string]any {
	max := len(symbols)
	if len(titles) > max {
		max = len(titles)
	}
	if len(percentages) > max {
		max = len(percentages)
	}
	if max == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, max)
	for i := 0; i < max; i++ {
		row := map[string]any{}
		if i < len(symbols) {
			row["symbol"] = symbols[i]
		}
		if i < len(titles) {
			row["title"] = titles[i]
		}
		if i < len(percentages) {
			if f, err := strconv.ParseFloat(strings.TrimSpace(percentages[i]), 64); err == nil {
				row["percentage"] = f
			} else {
				row["percentage"] = percentages[i]
			}
		}
		rows = append(rows, row)
	}
	return rows
}

var nonSymbolChars = regexp.MustCompile(`[^A-Z0-9/]+`)

func buildCPCMapRows(sourceSymbol, from, to string, symbols []string) []map[string]any {
	sourceSymbol = strings.ToUpper(strings.TrimSpace(sourceSymbol))
	seen := map[string]struct{}{}
	rows := []map[string]any{}
	for _, candidate := range symbols {
		candidate = strings.ToUpper(strings.TrimSpace(candidate))
		candidate = nonSymbolChars.ReplaceAllString(candidate, "")
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		rows = append(rows, map[string]any{
			"from":       sourceSymbol,
			"fromScheme": strings.ToUpper(from),
			"to":         candidate,
			"toScheme":   strings.ToUpper(to),
		})
	}
	return rows
}
