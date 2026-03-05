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

func newNumberCmd() *cobra.Command {
	numberCmd := &cobra.Command{
		Use:   "number",
		Short: "Number-service conversions",
	}
	numberCmd.AddCommand(newNumberConvertCmd())
	numberCmd.AddCommand(newNumberNormalizeCmd())
	return numberCmd
}

func newNumberConvertCmd() *cobra.Command {
	var (
		refType    string
		fromFormat string
		toFormat   string
		guessMode  bool
		flatMode   bool
	)

	cmd := &cobra.Command{
		Use:   "convert <reference>",
		Short: "Convert patent number formats through OPS number-service",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isOneOf(refType, "application", "publication", "priority") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --ref-type %q", refType),
					Hint:    "Use application, publication, or priority",
				}
			}
			if !isOneOf(fromFormat, "original", "docdb", "epodoc", "auto") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --from-format %q", fromFormat),
					Hint:    "Use auto, original, docdb, or epodoc",
				}
			}
			if !isOneOf(toFormat, "original", "docdb", "epodoc") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --to-format %q", toFormat),
					Hint:    "Use original, docdb, or epodoc",
				}
			}
			if fromFormat != "auto" && fromFormat == toFormat {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "--from-format and --to-format must be different",
				}
			}

			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			if flatMode {
				return runNumberConvertFlat(cmd, references, refType, fromFormat, toFormat, guessMode)
			}
			return runOPSBatch(cmd, "number-service", references, func(reference string) (api.Request, map[string]any, error) {
				effectiveFrom := resolveNumberFromFormat(reference, fromFormat, guessMode)
				path := fmt.Sprintf("/number-service/%s/%s/%s/%s", refType, effectiveFrom, reference, toFormat)
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

	cmd.Flags().StringVar(&refType, "ref-type", "application", "Reference type: application, publication, priority")
	cmd.Flags().StringVar(&fromFormat, "from-format", "auto", "Input number format: auto, original, docdb, epodoc")
	cmd.Flags().StringVar(&toFormat, "to-format", "epodoc", "Output number format: original, docdb, epodoc")
	cmd.Flags().BoolVar(&guessMode, "guess-format", true, "Auto-detect input format when --from-format=auto")
	cmd.Flags().BoolVar(&guessMode, "auto-detect", true, "Alias for --guess-format")
	cmd.Flags().BoolVar(&flatMode, "normalize", false, "Return flattened conversion fields (input/output country, doc-number, kind)")
	return cmd
}

func newNumberNormalizeCmd() *cobra.Command {
	var refType string

	cmd := &cobra.Command{
		Use:   "normalize <reference>",
		Short: "Auto-detect reference format and normalize to docdb",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runNumberConvertFlat(cmd, references, refType, "auto", "docdb", true)
		},
	}
	cmd.Flags().StringVar(&refType, "ref-type", "publication", "Reference type: application, publication, priority")
	return cmd
}

var (
	numberDocdbPattern   = regexp.MustCompile(`^[A-Z]{2}\.[0-9]+(?:\.[A-Z][0-9]?)?(?:\.[0-9]{8})?$`)
	numberEpodocPattern  = regexp.MustCompile(`^[A-Z]{2}[0-9]+(?:[A-Z][0-9]?)?$`)
	numberOriginalMarker = regexp.MustCompile(`[()/,-]`)
)

func resolveNumberFromFormat(reference, fromFormat string, guessMode bool) string {
	fromFormat = strings.TrimSpace(strings.ToLower(fromFormat))
	if fromFormat != "auto" {
		return fromFormat
	}
	if !guessMode {
		return "docdb"
	}
	return detectNumberFormat(reference)
}

func detectNumberFormat(reference string) string {
	ref := strings.ToUpper(strings.TrimSpace(reference))
	switch {
	case numberDocdbPattern.MatchString(ref):
		return "docdb"
	case numberEpodocPattern.MatchString(ref):
		return "epodoc"
	case numberOriginalMarker.MatchString(ref):
		return "original"
	default:
		return "original"
	}
}

func runNumberConvertFlat(cmd *cobra.Command, references []string, refType, fromFormat, toFormat string, guessMode bool) error {
	rows := make([]map[string]any, 0, len(references))
	for _, reference := range references {
		effectiveFrom := resolveNumberFromFormat(reference, fromFormat, guessMode)
		request := api.Request{
			Method: http.MethodGet,
			Path:   fmt.Sprintf("/number-service/%s/%s/%s/%s", refType, effectiveFrom, reference, toFormat),
			Accept: "application/json",
		}
		resp, err := executeOPSRequest(cmd.Context(), request)
		if err != nil {
			rows = append(rows, map[string]any{
				"input": reference,
				"ok":    false,
				"error": mapError(err),
			})
			continue
		}

		parsed, _ := parseJSONBody(resp.Body)
		flatRows, _ := extractNumberRows(asAnyMap(parsed))
		if len(flatRows) == 0 {
			rows = append(rows, map[string]any{
				"input": reference,
				"ok":    true,
				"from":  effectiveFrom,
				"to":    toFormat,
			})
			continue
		}
		row := flatRows[0]
		row["input"] = reference
		row["ok"] = true
		row["from"] = effectiveFrom
		row["to"] = toFormat
		if isSuspiciousPatentReference(reference) {
			row["warning"] = "input does not strongly resemble a known patent number pattern"
		}
		rows = append(rows, row)
	}

	if len(rows) == 1 {
		single := rows[0]
		if ok, _ := single["ok"].(bool); ok {
			return outputSuccess(cmd, responsePayload{
				Service: "number-service",
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
		Service: "number-service",
		Results: rows,
	})
}

func isSuspiciousPatentReference(reference string) bool {
	ref := strings.ToUpper(strings.TrimSpace(reference))
	return !numberDocdbPattern.MatchString(ref) && !numberEpodocPattern.MatchString(ref) && !numberOriginalMarker.MatchString(ref)
}
