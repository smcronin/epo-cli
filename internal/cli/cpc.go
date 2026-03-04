package cli

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newCPCCmd() *cobra.Command {
	cpcCmd := &cobra.Command{
		Use:   "cpc",
		Short: "Classification (CPC) service operations",
	}
	cpcCmd.AddCommand(newCPCGetCmd())
	cpcCmd.AddCommand(newCPCSearchCmd())
	cpcCmd.AddCommand(newCPCMediaCmd())
	cpcCmd.AddCommand(newCPCMapCmd())
	return cpcCmd
}

func newCPCGetCmd() *cobra.Command {
	var (
		depth      string
		navigation bool
		ancestors  bool
		accept     string
	)

	cmd := &cobra.Command{
		Use:   "get <symbol>",
		Short: "Retrieve CPC symbol details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol := strings.TrimSpace(args[0])
			if symbol == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "symbol is required",
				}
			}

			path := "/classification/cpc/" + symbol
			query := url.Values{}
			if v := strings.TrimSpace(depth); v != "" {
				query.Set("depth", v)
			}
			if navigation {
				query.Set("navigation", "true")
			}
			if ancestors {
				query.Set("ancestors", "true")
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   path,
				Query:  query,
				Accept: defaultIfEmptyString(accept, "application/cpc+xml"),
			}
			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"query":  compactQuery(request.Query),
				"accept": request.Accept,
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputOPSResponse(cmd, "classification/cpc", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&depth, "depth", "", "Optional depth (for example: 1 or all)")
	cmd.Flags().BoolVar(&navigation, "navigation", false, "Include previous/next navigation nodes")
	cmd.Flags().BoolVar(&ancestors, "ancestors", false, "Include ancestor nodes")
	cmd.Flags().StringVar(&accept, "accept", "application/cpc+xml", "Accept header")
	return cmd
}

func newCPCSearchCmd() *cobra.Command {
	var (
		query       string
		rangeHeader string
		accept      string
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search CPC classes by keyword",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(query) == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "missing search query",
					Hint:    "Pass --q \"chemistry\"",
				}
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   "/classification/cpc/search/",
				Query:  url.Values{"q": []string{query}},
				Accept: defaultIfEmptyString(accept, "application/cpc+xml"),
			}
			if v := strings.TrimSpace(rangeHeader); v != "" {
				request.Query.Set("Range", v)
			}

			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"query":  compactQuery(request.Query),
				"accept": request.Accept,
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputOPSResponse(cmd, "classification/cpc", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&query, "q", "", "Search keyword")
	cmd.Flags().StringVar(&rangeHeader, "range", "", "Range window (for example 1-20)")
	cmd.Flags().StringVar(&accept, "accept", "application/cpc+xml", "Accept header")
	return cmd
}

func newCPCMediaCmd() *cobra.Command {
	var (
		accept      string
		outputPath  string
		includeBody bool
	)

	cmd := &cobra.Command{
		Use:   "media <media-id>",
		Short: "Fetch CPC media asset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mediaID := strings.TrimSpace(args[0])
			if mediaID == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "media-id is required",
				}
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   "/classification/cpc/media/" + mediaID,
				Accept: defaultIfEmptyString(accept, "image/gif"),
			}
			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"accept": request.Accept,
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}

			results := map[string]any{
				"contentType": resp.Headers.Get("Content-Type"),
				"bytes":       len(resp.Body),
				"sha256":      sha256Hex(resp.Body),
			}
			warnings := []string{}

			if strings.TrimSpace(outputPath) != "" {
				cleanPath := filepath.Clean(outputPath)
				if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
					return fmt.Errorf("create output directory: %w", err)
				}
				if err := os.WriteFile(cleanPath, resp.Body, 0o644); err != nil {
					return fmt.Errorf("write media file: %w", err)
				}
				results["saved"] = true
				results["outputPath"] = cleanPath
			}

			if includeBody {
				results["base64"] = base64.StdEncoding.EncodeToString(resp.Body)
			} else if strings.TrimSpace(outputPath) == "" {
				warnings = append(warnings, "Body omitted; use --include-body or --out to retain content.")
			}

			payload := responsePayload{
				Service: "classification/cpc",
				Request: requestMeta,
				Throttle: map[string]any{
					"system":   resp.Metadata.Throttle.System,
					"services": resp.Metadata.Throttle.Services,
				},
				Quota: map[string]int{
					"hourUsed":       resp.Metadata.Quota.IndividualPerHourUsed,
					"weekUsed":       resp.Metadata.Quota.RegisteredPerWeekUsed,
					"payingWeekUsed": resp.Metadata.Quota.RegisteredPayingPerWeekUsed,
				},
				Results:  results,
				Warnings: warnings,
			}
			return outputSuccess(cmd, payload)
		},
	}

	cmd.Flags().StringVar(&accept, "accept", "image/gif", "Accept header for media request")
	cmd.Flags().StringVar(&outputPath, "out", "", "Optional output file path for binary content")
	cmd.Flags().BoolVar(&includeBody, "include-body", false, "Include base64 body in JSON output")
	return cmd
}

func newCPCMapCmd() *cobra.Command {
	var (
		from       string
		to         string
		additional bool
		accept     string
	)

	cmd := &cobra.Command{
		Use:   "map <symbol>",
		Short: "Map CPC/ECLA/IPC classification symbols",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol := strings.TrimSpace(args[0])
			if symbol == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "symbol is required",
				}
			}
			if !isOneOf(from, "ecla", "cpc") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --from %q", from),
					Hint:    "Use ecla or cpc",
				}
			}
			if !isOneOf(to, "cpc", "ecla", "ipc") {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("unsupported --to %q", to),
					Hint:    "Use cpc, ecla, or ipc",
				}
			}

			path := fmt.Sprintf("/classification/map/%s/%s/%s", from, symbol, to)
			query := url.Values{}
			if additional {
				query.Set("additional", "true")
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   path,
				Query:  query,
				Accept: defaultIfEmptyString(accept, "application/cpc+xml"),
			}
			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"query":  compactQuery(request.Query),
				"accept": request.Accept,
			}

			resp, err := executeOPSRequest(cmd.Context(), request)
			if err != nil {
				return err
			}
			return outputOPSResponse(cmd, "classification/cpc", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&from, "from", "cpc", "Input scheme: ecla or cpc")
	cmd.Flags().StringVar(&to, "to", "ecla", "Output scheme: cpc, ecla, or ipc")
	cmd.Flags().BoolVar(&additional, "additional", false, "Include additional mapping context when supported")
	cmd.Flags().StringVar(&accept, "accept", "application/cpc+xml", "Accept header")
	return cmd
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func defaultIfEmptyString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
