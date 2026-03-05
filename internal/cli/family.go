package cli

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newFamilyCmd() *cobra.Command {
	familyCmd := &cobra.Command{
		Use:   "family",
		Short: "INPADOC family service operations",
	}
	familyCmd.AddCommand(newFamilyGetCmd())
	familyCmd.AddCommand(newFamilySummaryCmd())
	return familyCmd
}

func newFamilyGetCmd() *cobra.Command {
	var (
		refType      string
		inputFormat  string
		constituents string
		flatMode     bool
		tableMode    bool
	)

	cmd := &cobra.Command{
		Use:   "get <reference>",
		Short: "Fetch INPADOC family data for a reference",
		Long:  "Fetch INPADOC family data. Some family members may only include publication-reference data without exchange-document nodes.",
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
			if strings.TrimSpace(constituents) != "" && !isOneOf(constituents, "biblio", "legal", "biblio,legal") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --constituents %q", constituents),
					Hint:    "Use biblio, legal, or biblio,legal",
				}
			}

			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			if tableMode {
				flagFormat = "table"
				flatMode = true
			}
			if flatMode {
				return runFamilyFlat(cmd, references, refType, inputFormat, constituents)
			}
			return runOPSBatch(cmd, "family", references, func(reference string) (api.Request, map[string]any, error) {
				path := fmt.Sprintf("/family/%s/%s/%s", refType, inputFormat, reference)
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

	cmd.Flags().StringVar(&refType, "ref-type", "publication", "Reference type: publication, application, priority")
	cmd.Flags().StringVar(&inputFormat, "input-format", "docdb", "Input format: docdb or epodoc")
	cmd.Flags().StringVar(&constituents, "constituents", "", "Optional family constituents: biblio, legal, biblio,legal")
	cmd.Flags().BoolVar(&flatMode, "flat", false, "Return normalized family rows")
	cmd.Flags().BoolVar(&tableMode, "table", false, "Shortcut for --format table --flat")
	return cmd
}

func runFamilyFlat(cmd *cobra.Command, references []string, refType, inputFormat, constituents string) error {
	results := make([]map[string]any, 0, len(references))
	for _, reference := range references {
		path := fmt.Sprintf("/family/%s/%s/%s", refType, inputFormat, reference)
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
		rows, _ := extractFamilyRows(asAnyMap(parsed))
		results = append(results, map[string]any{
			"input":    reference,
			"ok":       true,
			"results":  rows,
			"warnings": warnings,
		})
	}

	if len(results) == 1 {
		single := results[0]
		if ok, _ := single["ok"].(bool); ok {
			return outputSuccess(cmd, responsePayload{
				Service: "family",
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
		Service: "family",
		Results: results,
	})
}

func newFamilySummaryCmd() *cobra.Command {
	var (
		refType     string
		inputFormat string
	)

	cmd := &cobra.Command{
		Use:   "summary <reference>",
		Short: "Return condensed family summary with country counts",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runFamilySummary(cmd, references, refType, inputFormat)
		},
	}

	cmd.Flags().StringVar(&refType, "ref-type", "publication", "Reference type: publication, application, priority")
	cmd.Flags().StringVar(&inputFormat, "input-format", "docdb", "Input format: docdb or epodoc")
	return cmd
}

func runFamilySummary(cmd *cobra.Command, references []string, refType, inputFormat string) error {
	summaries := make([]map[string]any, 0, len(references))
	for _, reference := range references {
		request := api.Request{
			Method: http.MethodGet,
			Path:   fmt.Sprintf("/family/%s/%s/%s", refType, inputFormat, reference),
			Accept: "application/json",
		}
		resp, err := executeOPSRequest(cmd.Context(), request)
		if err != nil {
			summaries = append(summaries, map[string]any{
				"input": reference,
				"ok":    false,
				"error": mapError(err),
			})
			continue
		}

		parsed, _ := parseJSONBody(resp.Body)
		rows, _ := extractFamilyRows(asAnyMap(parsed))
		countryCounts := map[string]int{}
		for _, row := range rows {
			country := textValue(row["pubCountry"])
			if country == "" {
				country = textValue(row["appCountry"])
			}
			if country == "" {
				continue
			}
			countryCounts[country]++
		}

		summaries = append(summaries, map[string]any{
			"input":        reference,
			"ok":           true,
			"memberCount":  len(rows),
			"countryCount": countryCounts,
		})
	}

	if len(summaries) == 1 {
		single := summaries[0]
		if ok, _ := single["ok"].(bool); ok {
			return outputSuccess(cmd, responsePayload{
				Service: "family",
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
		Service: "family",
		Results: summaries,
	})
}
