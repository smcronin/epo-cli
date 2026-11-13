package cli

import (
	"fmt"
	"strings"
)

// --pick projection rules:
//
//   - The output shape follows the input shape. A list in gives a list out,
//     also when it holds a single hit; a single object in gives an object out.
//   - Envelope-level projection happens only when every pick field is an
//     explicit envelope path (results.x, pagination.x, quota.x, throttle.x,
//     request.x or warnings). Bare names resolve against the result rows, so
//     a row field called "command" or "ok" is not shadowed by the envelope.
//   - A pick field that matches nothing adds a warning instead of vanishing.
//   - Failed rows of a batch keep input/query/ok/error.

// pickEnvelopeSections are the envelope sections a pick path can reach into.
var pickEnvelopeSections = []string{"results", "pagination", "quota", "throttle", "request"}

// pickEnvelopeKeys are bare envelope names. They are printed in the envelope
// regardless of --pick, so as pick fields they resolve against rows.
var pickEnvelopeKeys = []string{"ok", "command", "service", "version", "results", "pagination", "quota", "throttle", "request"}

func isEnvelopePickPath(field string) bool {
	field = strings.TrimSpace(field)
	if field == "warnings" || strings.HasPrefix(field, "warnings[") || strings.HasPrefix(field, "warnings.") {
		return true
	}
	for _, section := range pickEnvelopeSections {
		if strings.HasPrefix(field, section+".") || strings.HasPrefix(field, section+"[") {
			return true
		}
	}
	return false
}

func allEnvelopePickPaths(fields []string) bool {
	if len(fields) == 0 {
		return false
	}
	for _, field := range fields {
		if !isEnvelopePickPath(field) {
			return false
		}
	}
	return true
}

// pickResults applies --pick to a success envelope. It returns the projected
// results and one warning per pick field that matched nothing.
func pickResults(env successEnvelope) (any, []string) {
	fields := parsePickFields(flagPick)
	if len(fields) == 0 {
		return env.Results, nil
	}

	envelopeOnly := allEnvelopePickPaths(fields)
	if envelopeOnly {
		if projected, warnings, ok := projectEnvelopePaths(env, fields); ok {
			return projected, warnings
		}
	}

	projector := newPickProjector(fields)
	results := projector.project(env.Results)
	return results, projector.warnings(envelopeOnly)
}

// projectEnvelopeIfRequested returns the envelope-level projection when every
// --pick field is an explicit envelope path and at least one of them resolves.
func projectEnvelopeIfRequested(env successEnvelope) (any, bool) {
	projected, _, ok := projectEnvelopePaths(env, parsePickFields(flagPick))
	return projected, ok
}

func projectEnvelopePaths(env successEnvelope, fields []string) (map[string]any, []string, bool) {
	if !allEnvelopePickPaths(fields) {
		return nil, nil, false
	}

	envelopeMap := map[string]any{
		"request":    env.Request,
		"pagination": env.Pagination,
		"throttle":   env.Throttle,
		"quota":      env.Quota,
		"results":    env.Results,
		"warnings":   env.Warnings,
	}
	projected := map[string]any{}
	warnings := []string{}
	for _, field := range fields {
		if value, ok := lookupProjectionValue(envelopeMap, field); ok {
			projected[field] = value
			continue
		}
		warnings = append(warnings, fmt.Sprintf("--pick path %q matched nothing in the envelope", field))
	}
	if len(projected) == 0 {
		return nil, nil, false
	}
	return projected, warnings, true
}

// pickProjector projects rows and remembers which pick fields ever matched.
type pickProjector struct {
	fields  []string
	matched map[string]bool
	rows    int
}

func newPickProjector(fields []string) *pickProjector {
	return &pickProjector{fields: fields, matched: map[string]bool{}}
}

// project applies the pick fields to a results value of any shape.
func (p *pickProjector) project(value any) any {
	if projected, ok := p.projectBatch(value); ok {
		return projected
	}
	return p.projectValue(value)
}

// projectValue projects a non-batch value: a list stays a list, a single
// object stays an object.
func (p *pickProjector) projectValue(value any) any {
	if summary, ok := value.(map[string]any); ok && isSummaryObject(summary) {
		return p.projectSummary(summary)
	}

	rows, isList, ok := normalizeRowsShape(value)
	if !ok {
		return value
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, p.projectListRow(row))
	}
	if !isList && len(out) == 1 {
		return out[0]
	}
	return out
}

func (p *pickProjector) projectListRow(row map[string]any) map[string]any {
	switch {
	case isFailedBatchRow(row):
		out := batchRowIdentity(row)
		for _, field := range p.fields {
			if value, ok := lookupProjectionValue(row, field); ok {
				out[field] = value
			}
		}
		return out
	case isSummaryObject(row):
		out := p.projectSummary(row)
		for key, value := range batchRowIdentity(row) {
			if _, exists := out[key]; !exists {
				out[key] = value
			}
		}
		return out
	default:
		return p.projectRow(row, p.fields)
	}
}

func (p *pickProjector) projectRow(row map[string]any, fields []string) map[string]any {
	p.rows++
	projected := map[string]any{}
	for _, field := range fields {
		if value, ok := lookupProjectionValue(row, field); ok {
			projected[field] = value
			p.matched[field] = true
		}
	}
	return projected
}

// projectSummary handles a --summary object ({query,total,...,topResults}).
// Fields that name summary keys are read from the summary itself; any other
// field is applied to the topResults rows, and the summary keys are kept so
// the counts stay next to the rows they describe.
func (p *pickProjector) projectSummary(summary map[string]any) map[string]any {
	rowFields := make([]string, 0, len(p.fields))
	for _, field := range p.fields {
		if _, ok := lookupProjectionValue(summary, field); !ok {
			rowFields = append(rowFields, field)
		}
	}
	if len(rowFields) == 0 {
		return p.projectRow(summary, p.fields)
	}

	out := make(map[string]any, len(summary))
	for key, value := range summary {
		if key != "topResults" {
			out[key] = value
		}
	}
	for _, field := range p.fields {
		if _, ok := lookupProjectionValue(summary, field); ok {
			p.matched[field] = true
		}
	}

	rows, _ := normalizeRows(summary["topResults"])
	projected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		projected = append(projected, p.projectRow(row, rowFields))
	}
	out["topResults"] = projected
	return out
}

// projectBatch handles per-input batch rows ({input|query, ok, results, ...}).
// Each row keeps its identity keys and has its results projected; failed rows
// pass through with their error.
func (p *pickProjector) projectBatch(value any) ([]map[string]any, bool) {
	items, ok := asAnySlice(value)
	if !ok || len(items) == 0 {
		return nil, false
	}

	rows := make([]map[string]any, 0, len(items))
	hasResultsRow := false
	for _, item := range items {
		row := asAnyMap(item)
		if len(row) == 0 {
			return nil, false
		}
		if !isFailedBatchRow(row) {
			_, hasResults := row["results"]
			_, hasOK := row["ok"]
			_, hasInput := row["input"]
			if !(hasResults && (hasOK || hasInput)) {
				return nil, false
			}
			hasResultsRow = true
		}
		rows = append(rows, row)
	}
	if !hasResultsRow {
		return nil, false
	}

	projected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out := batchRowIdentity(row)
		if results, exists := row["results"]; exists {
			out["results"] = p.projectValue(results)
		}
		projected = append(projected, out)
	}
	return projected, true
}

func batchRowIdentity(row map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"input", "query", "ok", "error"} {
		if value, exists := row[key]; exists {
			out[key] = value
		}
	}
	return out
}

func isFailedBatchRow(row map[string]any) bool {
	ok, isBool := row["ok"].(bool)
	_, hasError := row["error"]
	return isBool && !ok && hasError
}

func isSummaryObject(row map[string]any) bool {
	_, ok := asAnySlice(row["topResults"])
	return ok
}

// warnings reports the pick fields that matched no row. Nothing is reported
// when there were no rows to match against.
func (p *pickProjector) warnings(envelopeOnly bool) []string {
	if p.rows == 0 {
		return nil
	}
	out := []string{}
	for _, field := range p.fields {
		if p.matched[field] {
			continue
		}
		out = append(out, pickUnmatchedWarning(field, envelopeOnly))
	}
	return out
}

func pickUnmatchedWarning(field string, envelopeOnly bool) string {
	if envelopeOnly {
		return fmt.Sprintf("--pick path %q matched nothing in the envelope or the result rows", field)
	}
	message := fmt.Sprintf("--pick field %q matched nothing in the result rows", field)
	switch {
	case isEnvelopePickPath(field):
		message += "; envelope paths (results.x, pagination.x, quota.x, throttle.x, request.x, warnings) apply only when every --pick field is one"
	case isOneOf(field, pickEnvelopeSections...):
		message += fmt.Sprintf("; %q is already printed in the envelope, use a path such as %s.<field> to project inside it", field, field)
	case isOneOf(field, pickEnvelopeKeys...):
		message += fmt.Sprintf("; %q is already printed in the envelope", field)
	}
	return message
}
