package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newRegisterCmd() *cobra.Command {
	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "EP Register service operations",
	}
	registerCmd.AddCommand(newRegisterGetCmd())
	registerCmd.AddCommand(newRegisterSimpleCmd("events", "events", "Fetch EP register events"))
	registerCmd.AddCommand(newRegisterSimpleCmd("procedural-steps", "procedural-steps", "Fetch EP register procedural steps"))
	registerCmd.AddCommand(newRegisterUPPCmd())
	registerCmd.AddCommand(newRegisterSearchCmd())
	return registerCmd
}

func newRegisterGetCmd() *cobra.Command {
	var constituents string

	cmd := &cobra.Command{
		Use:   "get <reference>",
		Short: "Fetch EP register data for an application reference",
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

			path := fmt.Sprintf("/register/application/epodoc/%s", reference)
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

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputOPSResponse(cmd, "register", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&constituents, "constituents", "", "Optional constituents: biblio,events,procedural-steps")
	return cmd
}

func newRegisterSimpleCmd(name, endpoint, shortDesc string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <reference>", name),
		Short: shortDesc,
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

			path := fmt.Sprintf("/register/application/epodoc/%s/%s", reference, endpoint)
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
			return outputOPSResponse(cmd, "register", requestMeta, resp, nil)
		},
	}
	return cmd
}

func newRegisterUPPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upp <reference>",
		Short: "Fetch unitary patent protection (UPP) data for an EP publication reference",
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

			path := fmt.Sprintf("/register/publication/epodoc/%s/upp", reference)
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
			return outputOPSResponse(cmd, "register", requestMeta, resp, nil)
		},
	}
	return cmd
}

func newRegisterSearchCmd() *cobra.Command {
	var (
		query       string
		rangeHeader string
		usePost     bool
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Run CQL search against EP Register",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(query) == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "missing CQL query",
					Hint:    "Pass --q \"pa=IBM\"",
				}
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   "/register/search",
				Accept: "application/json",
			}
			if v := strings.TrimSpace(rangeHeader); v != "" {
				request.Headers = map[string]string{
					"Range": v,
				}
			}
			if usePost {
				request.Method = http.MethodPost
				request.ContentType = "text/plain"
				request.Body = []byte("q=" + url.QueryEscape(query))
			} else {
				request.Query = url.Values{}
				request.Query.Set("q", query)
			}

			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"query":  compactQuery(request.Query),
			}
			if len(request.Headers) > 0 {
				requestMeta["headers"] = request.Headers
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputOPSResponse(cmd, "register", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&query, "q", "", "CQL query (for example: pa=IBM)")
	cmd.Flags().StringVar(&rangeHeader, "range", "", "Result range, for example 1-25")
	cmd.Flags().BoolVar(&usePost, "post", false, "Use POST instead of GET")
	return cmd
}
