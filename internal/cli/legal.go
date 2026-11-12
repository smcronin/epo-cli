package cli

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	"github.com/spf13/cobra"
)

func newLegalCmd() *cobra.Command {
	legalCmd := &cobra.Command{
		Use:   "legal",
		Short: "Legal status service operations",
	}
	legalCmd.AddCommand(newLegalGetCmd())
	return legalCmd
}

func newLegalGetCmd() *cobra.Command {
	var (
		refType     string
		inputFormat string
		flatMode    bool
		summaryMode bool
	)

	cmd := &cobra.Command{
		Use:   "get <reference>",
		Short: "Fetch legal status events for a reference",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateRefType(refType); err != nil {
				return err
			}
			if err := validateInputFormat(inputFormat, "Use docdb or epodoc", "docdb", "epodoc"); err != nil {
				return err
			}

			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			if summaryMode {
				flatMode = true
			}
			if flatMode {
				return runLegalFlat(cmd, references, refType, inputFormat, summaryMode)
			}
			return runOPSBatch(cmd, "legal", references, func(reference string) (api.Request, map[string]any, error) {
				path := fmt.Sprintf("/legal/%s/%s/%s", refType, inputFormat, reference)
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

	cmd.Flags().StringVar(&refType, "ref-type", "publication", "Reference type: publication, application, priority")
	cmd.Flags().StringVar(&inputFormat, "input-format", "docdb", "Input format: docdb or epodoc")
	cmd.Flags().BoolVar(&flatMode, "flat", false, "Return simplified legal event rows {date,code,description,country,influence}")
	cmd.Flags().BoolVar(&summaryMode, "summary", false, "Return compact legal summary and counts")
	return cmd
}

var legalCodePattern = regexp.MustCompile(`^L[0-9]{3}EP$`)

func runLegalFlat(cmd *cobra.Command, references []string, refType, inputFormat string, summaryMode bool) error {
	results := make([]map[string]any, 0, len(references))
	for _, reference := range references {
		request := api.Request{
			Method: http.MethodGet,
			Path:   fmt.Sprintf("/legal/%s/%s/%s", refType, inputFormat, reference),
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
		events := flattenLegalEvents(parsed)
		payload := any(events)
		if summaryMode {
			payload = map[string]any{
				"eventCount": len(events),
				"events":     events,
			}
		}
		results = append(results, map[string]any{
			"input":    reference,
			"ok":       true,
			"results":  payload,
			"warnings": warnings,
		})
	}

	return finishBatch(cmd, "legal", results, func(single map[string]any) responsePayload {
		return responsePayload{
			Service:  "legal",
			Results:  single["results"],
			Warnings: toStringSlice(single["warnings"]),
		}
	})
}

func flattenLegalEvents(v any) []map[string]any {
	events := []map[string]any{}
	collectLegalEvents(v, &events)
	return events
}

func collectLegalEvents(v any, out *[]map[string]any) {
	switch t := v.(type) {
	case map[string]any:
		if event := legalEventFromMap(t); len(event) > 0 {
			*out = append(*out, event)
		}
		for _, child := range t {
			collectLegalEvents(child, out)
		}
	case []any:
		for _, child := range t {
			collectLegalEvents(child, out)
		}
	}
}

func legalEventFromMap(m map[string]any) map[string]any {
	if !looksLikeLegalEventMap(m) {
		return nil
	}

	date := firstNonEmpty(
		textValue(m["@date"]),
		textValue(m["date"]),
		textValue(legalValueByLocalKey(m, "L007EP")),
		textValue(legalValueByLocalKey(m, "L525EP")),
	)
	code := firstNonEmpty(
		textValue(m["@code"]),
		textValue(m["code"]),
		textValue(legalValueByLocalKey(m, "L008EP")),
		textValue(legalValueByLocalKey(m, "L501EP")),
	)
	description := firstNonEmpty(
		textValue(m["@desc"]),
		textValue(m["description"]),
		textValue(m["desc"]),
		textValue(legalValueByLocalKey(m, "L002EP")),
		textValue(legalValueByLocalKey(m, "L502EP")),
	)
	country := firstNonEmpty(
		textValue(m["country"]),
		textValue(legalValueByLocalKey(m, "L001EP")),
		textValue(legalValueByLocalKey(m, "L501EP")),
	)
	influence := firstNonEmpty(
		textValue(m["@infl"]),
		textValue(m["influence"]),
	)

	row := map[string]any{
		"date":        date,
		"code":        strings.TrimSpace(code),
		"description": strings.TrimSpace(description),
		"country":     strings.TrimSpace(country),
		"influence":   strings.TrimSpace(influence),
	}
	if text := textValue(legalValueByLocalKey(m, "L003EP")); text != "" {
		row["docNumber"] = text
	}
	if text := textValue(legalValueByLocalKey(m, "L004EP")); text != "" {
		row["kind"] = text
	}
	if text := textValue(legalValueByLocalKey(m, "L510EP")); text != "" {
		row["detail"] = text
	}
	return row
}

func looksLikeLegalEventMap(m map[string]any) bool {
	if strings.TrimSpace(textValue(m["@code"])) != "" {
		return true
	}
	hasL001 := false
	hasL007 := false
	hasL008 := false
	hasAnyCode := false
	for key := range m {
		local := localXMLKey(key)
		if legalCodePattern.MatchString(local) {
			hasAnyCode = true
		}
		if local == "L001EP" {
			hasL001 = true
		}
		if local == "L007EP" {
			hasL007 = true
		}
		if local == "L008EP" {
			hasL008 = true
		}
	}
	return hasAnyCode && hasL001 && (hasL007 || hasL008)
}

func legalValueByLocalKey(v any, local string) any {
	switch t := v.(type) {
	case map[string]any:
		for key, child := range t {
			if strings.EqualFold(localXMLKey(key), local) {
				return child
			}
			if nested := legalValueByLocalKey(child, local); nested != nil {
				return nested
			}
		}
	case []any:
		for _, child := range t {
			if nested := legalValueByLocalKey(child, local); nested != nil {
				return nested
			}
		}
	}
	return nil
}

func localXMLKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if idx := strings.LastIndex(key, ":"); idx >= 0 && idx+1 < len(key) {
		key = key[idx+1:]
	}
	return strings.TrimPrefix(key, "@")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
