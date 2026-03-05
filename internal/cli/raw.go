package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newRawCmd() *cobra.Command {
	rawCmd := &cobra.Command{
		Use:   "raw",
		Short: "Raw OPS request escape hatch",
		Long:  "Raw OPS request escape hatch. On Windows Git Bash/MSYS shells, prefix raw path calls with MSYS_NO_PATHCONV=1 to avoid path mangling.",
	}
	rawCmd.AddCommand(newRawGetCmd())
	rawCmd.AddCommand(newRawPostCmd())
	return rawCmd
}

func newRawGetCmd() *cobra.Command {
	var (
		baseURL string
		accept  string
		queryKV []string
	)

	cmd := &cobra.Command{
		Use:   "get <path>",
		Short: "Execute a raw GET against OPS",
		Example: strings.TrimSpace(`
MSYS_NO_PATHCONV=1 epo raw get "/published-data/publication/docdb/EP.1000000.A1/claims" -f json -q
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := strings.TrimSpace(args[0])
			if path == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "path is required",
				}
			}

			query, err := parseQueryPairs(queryKV)
			if err != nil {
				return err
			}
			request := api.Request{
				Method: http.MethodGet,
				Path:   path,
				Query:  query,
				Accept: defaultIfEmptyString(accept, "application/json"),
			}
			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"query":  compactQuery(request.Query),
				"base":   strings.TrimSpace(baseURL),
			}

			resp, err := executeOPSRequestWithBase(cmd.Context(), request, defaultRawBase(baseURL))
			if err != nil {
				return err
			}
			return outputOPSResponse(cmd, "raw", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&baseURL, "base-url", api.DefaultBaseURL, "Base URL for raw requests")
	cmd.Flags().StringVar(&accept, "accept", "application/json", "Accept header")
	cmd.Flags().StringArrayVar(&queryKV, "query", nil, "Query pair key=value (repeatable)")
	return cmd
}

func newRawPostCmd() *cobra.Command {
	var (
		baseURL     string
		accept      string
		contentType string
		queryKV     []string
		body        string
		bodyFile    string
	)

	cmd := &cobra.Command{
		Use:   "post <path>",
		Short: "Execute a raw POST against OPS",
		Example: strings.TrimSpace(`
MSYS_NO_PATHCONV=1 epo raw post "/published-data/search/biblio" --content-type text/plain --body "q=pa%3DIBM" -f json -q
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := strings.TrimSpace(args[0])
			if path == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "path is required",
				}
			}
			if strings.TrimSpace(body) != "" && strings.TrimSpace(bodyFile) != "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "use either --body or --body-file, not both",
				}
			}

			query, err := parseQueryPairs(queryKV)
			if err != nil {
				return err
			}

			requestBody := []byte(strings.TrimSpace(body))
			if strings.TrimSpace(bodyFile) != "" {
				fileBytes, err := os.ReadFile(strings.TrimSpace(bodyFile))
				if err != nil {
					return fmt.Errorf("read body-file: %w", err)
				}
				requestBody = fileBytes
			}

			request := api.Request{
				Method:      http.MethodPost,
				Path:        path,
				Query:       query,
				Body:        requestBody,
				ContentType: defaultIfEmptyString(contentType, "application/json"),
				Accept:      defaultIfEmptyString(accept, "application/json"),
			}
			requestMeta := map[string]any{
				"method":      request.Method,
				"path":        request.Path,
				"query":       compactQuery(request.Query),
				"base":        strings.TrimSpace(baseURL),
				"contentType": request.ContentType,
				"bodyBytes":   len(requestBody),
			}

			resp, err := executeOPSRequestWithBase(cmd.Context(), request, defaultRawBase(baseURL))
			if err != nil {
				return err
			}
			return outputOPSResponse(cmd, "raw", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&baseURL, "base-url", api.DefaultBaseURL, "Base URL for raw requests")
	cmd.Flags().StringVar(&accept, "accept", "application/json", "Accept header")
	cmd.Flags().StringVar(&contentType, "content-type", "application/json", "Content-Type header")
	cmd.Flags().StringArrayVar(&queryKV, "query", nil, "Query pair key=value (repeatable)")
	cmd.Flags().StringVar(&body, "body", "", "Inline request body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Path to request body file")
	return cmd
}

func parseQueryPairs(queryKV []string) (url.Values, error) {
	query := url.Values{}
	for _, pair := range queryKV {
		raw := strings.TrimSpace(pair)
		if raw == "" {
			continue
		}
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, &epoerrors.CLIError{
				Code:    400,
				Type:    "VALIDATION_ERROR",
				Message: fmt.Sprintf("invalid --query pair %q", pair),
				Hint:    "Use key=value",
			}
		}
		query.Add(strings.TrimSpace(parts[0]), parts[1])
	}
	return query, nil
}

func defaultRawBase(baseURL string) string {
	v := strings.TrimSpace(baseURL)
	if v == "" {
		return api.DefaultBaseURL
	}
	return v
}
