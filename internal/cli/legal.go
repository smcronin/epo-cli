package cli

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
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
			if !isOneOf(refType, "publication", "application", "priority") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --ref-type %q", refType),
					Hint:    "Use publication, application, or priority",
				}
			}
			if !isOneOf(inputFormat, "docdb", "epodoc") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --input-format %q", inputFormat),
					Hint:    "Use docdb or epodoc",
				}
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

	if len(results) == 1 {
		single := results[0]
		if ok, _ := single["ok"].(bool); ok {
			return outputSuccess(cmd, responsePayload{
				Service: "legal",
				Results: single["results"],
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
		Service: "legal",
		Results: results,
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
	hasCodeShape := false
	for key := range m {
		if legalCodePattern.MatchString(strings.TrimSpace(key)) {
			hasCodeShape = true
			break
		}
	}
	if !hasCodeShape {
		return nil
	}

	row := map[string]any{
		"date":        firstNonEmpty(textValue(m["date"]), textValue(m["L007EP"]), textValue(m["L500EP"]), textValue(m["L515EP"])),
		"code":        firstNonEmpty(textValue(m["code"]), textValue(m["L001EP"]), textValue(m["L501EP"])),
		"description": firstNonEmpty(textValue(m["description"]), textValue(m["desc"]), textValue(m["L002EP"]), textValue(m["L502EP"])),
		"country":     firstNonEmpty(textValue(m["country"]), textValue(m["L003EP"])),
		"influence":   firstNonEmpty(textValue(m["influence"]), textValue(m["L004EP"])),
	}
	return row
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
