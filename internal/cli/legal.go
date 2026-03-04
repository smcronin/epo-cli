package cli

import (
	"fmt"
	"net/http"

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
	return cmd
}
