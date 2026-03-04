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
	return familyCmd
}

func newFamilyGetCmd() *cobra.Command {
	var (
		refType      string
		inputFormat  string
		constituents string
	)

	cmd := &cobra.Command{
		Use:   "get <reference>",
		Short: "Fetch INPADOC family data for a reference",
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
	return cmd
}
