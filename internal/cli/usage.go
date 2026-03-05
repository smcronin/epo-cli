package cli

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	usageCmd.AddCommand(newUsageTodayCmd())
	usageCmd.AddCommand(newUsageWeekCmd())
	usageCmd.AddCommand(newUsageQuotaCmd())
	return usageCmd
}

func newUsageStatsCmd() *cobra.Command {
	var (
		date       string
		from       string
		to         string
		humanDates bool
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
			results, warnings := parseJSONBody(resp.Body)
			if humanDates {
				results = withUsageHumanDates(results)
			}
			return outputSuccess(cmd, responsePayload{
				Service: "usage",
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
			})
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Single date in dd/mm/yyyy")
	cmd.Flags().StringVar(&from, "from", "", "Start date in dd/mm/yyyy")
	cmd.Flags().StringVar(&to, "to", "", "End date in dd/mm/yyyy")
	cmd.Flags().BoolVar(&humanDates, "human-dates", false, "Add human-readable date fields alongside epoch timestamps")
	return cmd
}

func newUsageTodayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "today",
		Short: "Fetch usage stats for today",
		RunE: func(cmd *cobra.Command, args []string) error {
			today := time.Now().Format("02/01/2006")
			return runUsageStatsShortcut(cmd, today, "", true)
		},
	}
}

func newUsageWeekCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "week",
		Short: "Fetch usage stats for the last 7 days",
		RunE: func(cmd *cobra.Command, args []string) error {
			end := time.Now()
			start := end.AddDate(0, 0, -6)
			return runUsageStatsShortcut(cmd, start.Format("02/01/2006"), end.Format("02/01/2006"), true)
		},
	}
}

func newUsageQuotaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quota",
		Short: "Show current quota/throttle counters without full usage payload",
		RunE: func(cmd *cobra.Command, args []string) error {
			request := api.Request{
				Method: http.MethodGet,
				Path:   "/developers/me/stats/usage",
				Query:  url.Values{"timeRange": []string{time.Now().Format("02/01/2006")}},
				Accept: "application/json",
			}
			resp, err := executeOPSRequestWithBase(cmd.Context(), request, usageBaseURL)
			if err != nil {
				return err
			}
			return outputSuccess(cmd, responsePayload{
				Service: "usage",
				Throttle: map[string]any{
					"system":   resp.Metadata.Throttle.System,
					"services": resp.Metadata.Throttle.Services,
				},
				Quota: map[string]int{
					"hourUsed":       resp.Metadata.Quota.IndividualPerHourUsed,
					"weekUsed":       resp.Metadata.Quota.RegisteredPerWeekUsed,
					"payingWeekUsed": resp.Metadata.Quota.RegisteredPayingPerWeekUsed,
				},
				Results: map[string]any{
					"message": "Quota counters update on OPS cadence and may lag behind message totals.",
				},
			})
		},
	}
}

func runUsageStatsShortcut(cmd *cobra.Command, from, to string, humanDates bool) error {
	timeRange := from
	if strings.TrimSpace(to) != "" {
		timeRange = from + "~" + to
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
	results, warnings := parseJSONBody(resp.Body)
	if humanDates {
		results = withUsageHumanDates(results)
	}
	return outputSuccess(cmd, responsePayload{
		Service: "usage",
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
	})
}

func withUsageHumanDates(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, value := range t {
			out[key] = withUsageHumanDates(value)
			if strings.EqualFold(key, "date") || strings.Contains(strings.ToLower(key), "timestamp") {
				if epoch := toEpochInt(value); epoch > 0 {
					out[key+"_human"] = time.Unix(epoch, 0).UTC().Format("2006-01-02")
				}
			}
		}
		return out
	case []any:
		rows := make([]any, 0, len(t))
		for _, item := range t {
			rows = append(rows, withUsageHumanDates(item))
		}
		return rows
	default:
		return v
	}
}

func toEpochInt(v any) int64 {
	switch t := v.(type) {
	case int64:
		return normalizeEpochSeconds(t)
	case int:
		return normalizeEpochSeconds(int64(t))
	case float64:
		return normalizeEpochSeconds(int64(t))
	case string:
		trimmed := strings.TrimSpace(t)
		if trimmed == "" {
			return 0
		}
		for _, layout := range []string{"2006-01-02", "02/01/2006"} {
			if parsed, err := time.Parse(layout, trimmed); err == nil {
				return parsed.Unix()
			}
		}
		parsed, err := time.Parse("20060102", trimmed)
		if err == nil {
			return parsed.Unix()
		}
		if numeric, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return normalizeEpochSeconds(numeric)
		}
	}
	return 0
}

func normalizeEpochSeconds(epoch int64) int64 {
	if epoch <= 0 {
		return 0
	}
	// OPS usage timestamps are milliseconds since epoch.
	if epoch >= 1_000_000_000_000 {
		return epoch / 1000
	}
	return epoch
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
