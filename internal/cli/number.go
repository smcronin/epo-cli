package cli

import (
	"fmt"
	"net/http"

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

			references, err := resolveSingleOrStdinInputs(args)
			if err != nil {
				return err
			}
			return runOPSBatch(cmd, "number-service", references, func(reference string) (api.Request, map[string]any, error) {
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
				return request, requestMeta, nil
			}, nil)
		},
	}

	cmd.Flags().StringVar(&refType, "ref-type", "application", "Reference type: application, publication, priority")
	cmd.Flags().StringVar(&fromFormat, "from-format", "docdb", "Input number format: original, docdb, epodoc")
	cmd.Flags().StringVar(&toFormat, "to-format", "epodoc", "Output number format: original, docdb, epodoc")
	return cmd
}
