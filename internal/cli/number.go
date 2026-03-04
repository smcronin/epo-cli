package cli

import (
	"fmt"
	"net/http"
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
	return numberCmd
}

func newNumberConvertCmd() *cobra.Command {
	var (
		refType    string
		fromFormat string
		toFormat   string
	)

	cmd := &cobra.Command{
		Use:   "convert <reference>",
		Short: "Convert patent number formats through OPS number-service",
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
			if !isOneOf(refType, "application", "publication", "priority") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --ref-type %q", refType),
					Hint:    "Use application, publication, or priority",
				}
			}
			if !isOneOf(fromFormat, "original", "docdb", "epodoc") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --from-format %q", fromFormat),
					Hint:    "Use original, docdb, or epodoc",
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
			if fromFormat == toFormat {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "--from-format and --to-format must be different",
				}
			}

			path := fmt.Sprintf("/number-service/%s/%s/%s/%s", refType, fromFormat, reference, toFormat)
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
			return outputOPSResponse(cmd, "number-service", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&refType, "ref-type", "application", "Reference type: application, publication, priority")
	cmd.Flags().StringVar(&fromFormat, "from-format", "docdb", "Input number format: original, docdb, epodoc")
	cmd.Flags().StringVar(&toFormat, "to-format", "epodoc", "Output number format: original, docdb, epodoc")
	return cmd
}
