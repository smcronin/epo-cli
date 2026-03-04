package cli

import (
	"fmt"
	"net/http"
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
	)

	cmd := &cobra.Command{
		Use:   "get <reference>",
		Short: "Fetch legal status events for a reference",
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

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputOPSResponse(cmd, "legal", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&refType, "ref-type", "publication", "Reference type: publication, application, priority")
	cmd.Flags().StringVar(&inputFormat, "input-format", "docdb", "Input format: docdb or epodoc")
	return cmd
}
