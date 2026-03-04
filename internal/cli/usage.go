package cli

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

const usageBaseURL = "https://ops.epo.org/3.2"

var usageDatePattern = regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)

func newUsageCmd() *cobra.Command {
	usageCmd := &cobra.Command{
		Use:   "usage",
		Short: "OPS usage statistics operations",
	}
	usageCmd.AddCommand(newUsageStatsCmd())
	return usageCmd
}

func newUsageStatsCmd() *cobra.Command {
	var (
		date string
		from string
		to   string
	)

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Fetch usage stats for a date or date range (dd/mm/yyyy)",
		RunE: func(cmd *cobra.Command, args []string) error {
			timeRange, err := resolveUsageTimeRange(date, from, to)
			if err != nil {
				return err
			}

			request := api.Request{
				Method: http.MethodGet,
				Path:   "/developers/me/stats/usage",
				Query:  url.Values{"timeRange": []string{timeRange}},
				Accept: "application/json",
			}
			requestMeta := map[string]any{
				"method": request.Method,
				"path":   request.Path,
				"query":  compactQuery(request.Query),
			}

			resp, err := executeOPSRequestWithBase(cmd.Context(), request, usageBaseURL)
			if err != nil {
				return err
			}
			return outputOPSResponse(cmd, "usage", requestMeta, resp, nil)
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Single date in dd/mm/yyyy")
	cmd.Flags().StringVar(&from, "from", "", "Start date in dd/mm/yyyy")
	cmd.Flags().StringVar(&to, "to", "", "End date in dd/mm/yyyy")
	return cmd
}

func resolveUsageTimeRange(date, from, to string) (string, error) {
	date = strings.TrimSpace(date)
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)

	if date != "" && (from != "" || to != "") {
		return "", &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "use either --date or --from/--to",
		}
	}

	if date != "" {
		if !usageDatePattern.MatchString(date) {
			return "", &epoerrors.CLIError{
				Code:    400,
				Type:    "VALIDATION_ERROR",
				Message: "invalid --date format",
				Hint:    "Use dd/mm/yyyy",
			}
		}
		return date, nil
	}

	if from == "" && to == "" {
		return "", &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "missing usage range",
			Hint:    "Use --date dd/mm/yyyy or --from dd/mm/yyyy --to dd/mm/yyyy",
		}
	}
	if from == "" || to == "" {
		return "", &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "both --from and --to are required for range queries",
		}
	}
	if !usageDatePattern.MatchString(from) || !usageDatePattern.MatchString(to) {
		return "", &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "invalid date format for range query",
			Hint:    "Use dd/mm/yyyy",
		}
	}

	return from + "~" + to, nil
}
