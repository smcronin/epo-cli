package cli

import (
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	"github.com/spf13/cobra"
)

type requestBuilder func(input string) (api.Request, map[string]any, error)

func runOPSBatch(cmd *cobra.Command, service string, inputs []string, build requestBuilder, pager paginationParser) error {
	if len(inputs) == 0 {
		return nil
	}

	if len(inputs) == 1 {
		request, requestMeta, err := build(inputs[0])
		if err != nil {
			return err
		}
		resp, err := executeOPSRequest(cmd.Context(), request)
		if err != nil {
			return err
		}
		return outputOPSResponse(cmd, service, requestMeta, resp, pager)
	}

	results := make([]map[string]any, 0, len(inputs))
	for _, input := range inputs {
		request, requestMeta, err := build(input)
		if err != nil {
			results = append(results, map[string]any{
				"input": input,
				"ok":    false,
				"error": mapError(err),
			})
			continue
		}

		resp, err := executeOPSRequest(cmd.Context(), request)
		if err != nil {
			results = append(results, map[string]any{
				"input": input,
				"ok":    false,
				"error": mapError(err),
			})
			continue
		}

		parsed, warnings := parseJSONBody(resp.Body)
		rangeHeader := strings.TrimSpace(resp.Headers.Get("X-OPS-Range"))
		pagination := parsePagination(rangeHeader)
		if pager != nil {
			pagination = mergePagination(pagination, pager(parsed))
		}

		item := map[string]any{
			"input":      input,
			"ok":         true,
			"request":    requestMeta,
			"pagination": pagination,
			"throttle": map[string]any{
				"system":   resp.Metadata.Throttle.System,
				"services": resp.Metadata.Throttle.Services,
			},
			"quota": map[string]int{
				"hourUsed":       resp.Metadata.Quota.IndividualPerHourUsed,
				"weekUsed":       resp.Metadata.Quota.RegisteredPerWeekUsed,
				"payingWeekUsed": resp.Metadata.Quota.RegisteredPayingPerWeekUsed,
			},
			"results":  parsed,
			"warnings": warnings,
		}
		results = append(results, item)
	}

	return outputSuccess(cmd, responsePayload{
		Service: service,
		Results: results,
	})
}
