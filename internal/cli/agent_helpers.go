package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
)

func applyPickProjection(value any) any {
	fields := parsePickFields(flagPick)
	if len(fields) == 0 {
		return value
	}
	return projectByFields(value, fields)
}

func parsePickFields(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		field := strings.TrimSpace(part)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	return out
}

func projectByFields(value any, fields []string) any {
	rows, ok := normalizeRows(value)
	if !ok {
		return value
	}

	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		projected := map[string]any{}
		for _, field := range fields {
			if fieldValue, ok := lookupProjectionValue(row, field); ok {
				projected[field] = fieldValue
			}
		}
		out = append(out, projected)
	}

	if len(out) == 1 {
		return out[0]
	}
	return out
}

func lookupProjectionValue(v any, path string) (any, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}
	if m, ok := v.(map[string]any); ok {
		if direct, ok := m[path]; ok {
			return direct, true
		}
	}

	current := v
	for _, segment := range strings.Split(path, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, false
		}
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[segment]
			if !ok {
				return nil, false
			}
			current = next
		case []any:
			i, err := strconv.Atoi(segment)
			if err != nil || i < 0 || i >= len(typed) {
				return nil, false
			}
			current = typed[i]
		default:
			return nil, false
		}
	}
	return current, true
}

func resolveSingleOrStdinInputs(args []string) ([]string, error) {
	if flagStdin {
		if len(args) > 0 {
			return nil, &epoerrors.CLIError{
				Code:    400,
				Type:    "VALIDATION_ERROR",
				Message: "do not pass positional inputs when using --stdin",
				Hint:    "Provide newline-separated inputs through stdin",
			}
		}
		return readStdinInputs()
	}

	if len(args) != 1 {
		return nil, &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "exactly one input is required",
		}
	}
	value := strings.TrimSpace(args[0])
	if value == "" {
		return nil, &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "input is required",
		}
	}
	return []string{value}, nil
}

func resolveQueryOrStdinInputs(query string) ([]string, error) {
	query = strings.TrimSpace(query)
	if flagStdin {
		if query != "" {
			return nil, &epoerrors.CLIError{
				Code:    400,
				Type:    "VALIDATION_ERROR",
				Message: "cannot combine query flags and --stdin",
				Hint:    "Pass one query per stdin line when using --stdin",
			}
		}
		return readStdinInputs()
	}
	if query == "" {
		return nil, &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "missing query input",
			Hint:    "Pass --query \"...\" (or --cql / legacy --q) or provide newline-separated queries with --stdin",
		}
	}
	return []string{query}, nil
}

func readStdinInputs() ([]string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("read stdin metadata: %w", err)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "stdin is empty",
			Hint:    "Pipe newline-separated values into the command",
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	values := []string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		values = append(values, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	if len(values) == 0 {
		return nil, &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "stdin did not contain any non-empty inputs",
		}
	}
	return values, nil
}

func parseRangeWindow(raw string) (start int, end int, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 1, 25, nil
	}

	parts := strings.SplitN(raw, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range %q: expected start-end", raw)
	}
	start, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || start <= 0 {
		return 0, 0, fmt.Errorf("invalid range start in %q", raw)
	}
	end, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || end < start {
		return 0, 0, fmt.Errorf("invalid range end in %q", raw)
	}
	return start, end, nil
}

func writeTableEnvelope(env successEnvelope) error {
	rows, ok := normalizeRows(env.Results)
	if !ok || len(rows) == 0 {
		rows = []map[string]any{
			{
				"ok":      env.OK,
				"command": env.Command,
				"service": env.Service,
				"version": env.Version,
			},
		}
	}
	writeSimpleTable(os.Stdout, rows)
	return nil
}

func writeSimpleTable(w io.Writer, rows []map[string]any) {
	headers := collectHeaders(rows)
	if len(headers) == 0 {
		return
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, h := range headers {
			v := stringifyValue(row[h])
			if len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
	}

	writeRow := func(values []string) {
		for i, v := range values {
			if i > 0 {
				fmt.Fprint(w, " | ")
			}
			fmt.Fprint(w, padRight(v, widths[i]))
		}
		fmt.Fprintln(w)
	}

	headerValues := make([]string, len(headers))
	copy(headerValues, headers)
	writeRow(headerValues)

	separator := make([]string, len(headers))
	for i := range headers {
		separator[i] = strings.Repeat("-", widths[i])
	}
	writeRow(separator)

	for _, row := range rows {
		values := make([]string, len(headers))
		for i, h := range headers {
			values[i] = stringifyValue(row[h])
		}
		writeRow(values)
	}
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func joinAndSortUnique(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
