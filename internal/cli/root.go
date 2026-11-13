package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/smcronin/epo-cli/internal/throttle"
	"github.com/spf13/cobra"
)

var version = "0.4.0"

var (
	flagClientID     string
	flagClientSecret string
	flagFormat       string
	flagMinify       bool
	flagQuiet        bool
	flagTimeout      int
	flagAll          bool
	flagPick         string
	flagStdin        bool
)

type successEnvelope struct {
	OK         bool     `json:"ok"`
	Command    string   `json:"command"`
	Service    string   `json:"service,omitempty"`
	Request    any      `json:"request,omitempty"`
	Pagination any      `json:"pagination,omitempty"`
	Throttle   any      `json:"throttle,omitempty"`
	Quota      any      `json:"quota,omitempty"`
	Results    any      `json:"results,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	Version    string   `json:"version"`
}

// errorEnvelope is the failure envelope. The fields between error and
// version are set only for a partial failure: an --all run that failed after
// at least one page, where the rows fetched so far are returned with the
// error instead of being thrown away.
type errorEnvelope struct {
	OK         bool                `json:"ok"`
	Partial    bool                `json:"partial,omitempty"`
	Error      *epoerrors.CLIError `json:"error"`
	Command    string              `json:"command,omitempty"`
	Service    string              `json:"service,omitempty"`
	Request    any                 `json:"request,omitempty"`
	Pagination any                 `json:"pagination,omitempty"`
	Results    any                 `json:"results,omitempty"`
	Warnings   []string            `json:"warnings,omitempty"`
	Version    string              `json:"version"`
}

// partialError is returned when an --all run fails mid-way. It unwraps to the
// original typed error, so the error type and exit code are unchanged, and
// carries the envelope of what was fetched before the failure.
type partialError struct {
	cause    *epoerrors.CLIError
	envelope successEnvelope
}

func (e *partialError) Error() string {
	return e.cause.Error()
}

func (e *partialError) Unwrap() error {
	return e.cause
}

type responsePayload struct {
	Service    string
	Request    any
	Pagination any
	Throttle   any
	Quota      any
	Results    any
	Warnings   []string
}

var rootCmd = &cobra.Command{
	Use:           "epo",
	Short:         "EPO patent CLI for OPS and EPS data access",
	Long:          "EPO patent CLI for agent-ready access to Open Patent Services (OPS) and European Publication Server (EPS) data.",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&flagClientID, "client-id", "", "EPO OPS client ID (consumer key)")
	pf.StringVar(&flagClientSecret, "client-secret", "", "EPO OPS client secret")
	pf.StringVarP(&flagFormat, "format", "f", "json", "Output format: json, ndjson, csv, or table")
	pf.BoolVar(&flagMinify, "minify", false, "Compact JSON output")
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-data output")
	pf.IntVar(&flagTimeout, "timeout", 30, "Request timeout in seconds")
	pf.BoolVar(&flagAll, "all", false, "Auto-paginate all results where supported")
	pf.StringVar(&flagPick, "pick", "", "Project result-row fields (comma-separated, dot paths supported); a list stays a list")
	pf.BoolVar(&flagStdin, "stdin", false, "Read batch inputs from stdin (newline-separated)")

	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newPubCmd())
	rootCmd.AddCommand(newFamilyCmd())
	rootCmd.AddCommand(newNumberCmd())
	rootCmd.AddCommand(newRegisterCmd())
	rootCmd.AddCommand(newLegalCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newCPCCmd())
	rootCmd.AddCommand(newUsageCmd())
	rootCmd.AddCommand(newEPSCmd())
	rootCmd.AddCommand(newRawCmd())
	rootCmd.AddCommand(newUpdateCmd())
	rootCmd.AddCommand(newMethodsCmd())

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return validateOutputFormat()
	}
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return usageError(cmd, err.Error())
	})
	applyUsageErrors(rootCmd)
}

// applyUsageErrors makes every command report misuse as a usage error
// (VALIDATION_ERROR, exit 2). Command groups reject unknown subcommands
// instead of printing help with exit 0, and positional-argument validators
// return typed errors.
func applyUsageErrors(cmd *cobra.Command) {
	if cmd.HasSubCommands() && cmd.Run == nil && cmd.RunE == nil {
		cmd.Args = cobra.ArbitraryArgs
		cmd.RunE = runCommandGroup
	} else if cmd.Args != nil {
		validate := cmd.Args
		cmd.Args = func(c *cobra.Command, args []string) error {
			if err := validate(c, args); err != nil {
				return usageError(c, err.Error())
			}
			return nil
		}
	}
	for _, child := range cmd.Commands() {
		applyUsageErrors(child)
	}
}

func runCommandGroup(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	message := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		message += fmt.Sprintf(" (did you mean %q?)", suggestions[0])
	}
	return usageError(cmd, message)
}

func usageError(cmd *cobra.Command, message string) error {
	return &epoerrors.CLIError{
		Code:    400,
		Type:    "VALIDATION_ERROR",
		Message: message,
		Hint:    fmt.Sprintf("Run `%s --help` for usage", cmd.CommandPath()),
	}
}

// validateOutputFormat rejects an unsupported --format before a command does
// any work (OPS calls, file writes).
func validateOutputFormat() error {
	switch flagFormat {
	case "json", "ndjson", "csv", "table":
		return nil
	}
	return &epoerrors.CLIError{
		Code:    400,
		Type:    "VALIDATION_ERROR",
		Message: fmt.Sprintf("unsupported format %q", flagFormat),
		Hint:    "Use --format json, ndjson, csv, or table",
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(handleError(err))
	}
}

func outputSuccess(cmd *cobra.Command, data any) error {
	env := buildSuccessEnvelope(cmd, data)

	var writeErr error
	switch flagFormat {
	case "json":
		writeErr = writeJSON(env)
	case "ndjson":
		writeErr = writeNDJSON(env)
	case "csv":
		writeErr = writeCSV(env)
	case "table":
		writeErr = writeTable(env)
	default:
		return validateOutputFormat()
	}
	if writeErr != nil {
		return writeErr
	}

	if !flagQuiet {
		emitQuotaWarnings(env)
	}
	return nil
}

func buildSuccessEnvelope(cmd *cobra.Command, data any) successEnvelope {
	env := successEnvelope{
		OK:      true,
		Command: cmd.CommandPath(),
		Version: version,
	}
	if payload, ok := data.(responsePayload); ok {
		env.Service = payload.Service
		env.Request = payload.Request
		env.Pagination = payload.Pagination
		env.Throttle = payload.Throttle
		env.Quota = payload.Quota
		env.Results = payload.Results
		env.Warnings = payload.Warnings
	} else {
		env.Results = data
	}
	results, pickWarnings := pickResults(env)
	env.Results = results
	env.Warnings = append(env.Warnings, pickWarnings...)
	return env
}

func writeJSON(v any) error {
	var (
		out []byte
		err error
	)
	if flagMinify {
		out, err = json.Marshal(v)
	} else {
		out, err = json.MarshalIndent(v, "", "  ")
	}
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(out))
	return nil
}

func writeTable(v any) error {
	switch data := v.(type) {
	case string:
		fmt.Fprintln(os.Stdout, data)
	case successEnvelope:
		return writeTableEnvelope(data)
	default:
		rows, ok := normalizeRows(data)
		if !ok || len(rows) == 0 {
			b, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal table fallback: %w", err)
			}
			fmt.Fprintln(os.Stdout, string(b))
			return nil
		}
		writeSimpleTable(os.Stdout, rows)
	}
	return nil
}

func writeNDJSON(env successEnvelope) error {
	if rows, ok := normalizeRows(env.Results); ok {
		enc := json.NewEncoder(os.Stdout)
		for _, row := range rows {
			if err := enc.Encode(row); err != nil {
				return fmt.Errorf("write ndjson row: %w", err)
			}
		}
		return nil
	}

	b, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal ndjson envelope: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(b))
	return nil
}

func writeCSV(env successEnvelope) error {
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

	headers := collectHeaders(rows)
	if len(headers) == 0 {
		return nil
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, row := range rows {
		record := make([]string, len(headers))
		for i, header := range headers {
			record[i] = stringifyValue(row[header])
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}

	fmt.Fprint(os.Stdout, buf.String())
	return nil
}

func normalizeRows(v any) ([]map[string]any, bool) {
	rows, _, ok := normalizeRowsShape(v)
	return rows, ok
}

// normalizeRowsShape flattens v into rows and reports whether v is
// list-shaped (search hits, family members, a slice) or a single record (one
// object, one number conversion). --pick uses this to keep the output shape
// stable: a list stays a list even with a single hit.
func normalizeRowsShape(v any) (rows []map[string]any, isList bool, ok bool) {
	if v == nil {
		return nil, false, false
	}
	if knownRows, ok := extractKnownRows(v); ok {
		return knownRows, !isSingleRecordPayload(v), true
	}
	if batchRows, ok := extractKnownBatchRows(v); ok {
		return batchRows, true, true
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false, false
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		rows := make([]map[string]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			row := asRowMap(rv.Index(i).Interface())
			rows = append(rows, row)
		}
		return rows, true, true
	case reflect.Map, reflect.Struct:
		return []map[string]any{asRowMap(rv.Interface())}, false, true
	default:
		return []map[string]any{{"value": rv.Interface()}}, false, true
	}
}

// isSingleRecordPayload reports whether a raw OPS payload describes one
// record rather than a result list. A number conversion is the only known-row
// payload of that kind.
func isSingleRecordPayload(v any) bool {
	root, ok := v.(map[string]any)
	if !ok {
		return false
	}
	worldPatentData := asAnyMap(root["ops:world-patent-data"])
	return len(asAnyMap(worldPatentData["ops:standardization"])) > 0
}

func extractKnownBatchRows(v any) ([]map[string]any, bool) {
	items, ok := asAnySlice(v)
	if !ok || len(items) == 0 {
		return nil, false
	}

	rows := make([]map[string]any, 0, len(items))
	matched := false
	for _, item := range items {
		entry := asAnyMap(item)
		if len(entry) == 0 {
			return nil, false
		}
		rawResults, hasResults := entry["results"]
		if !hasResults {
			return nil, false
		}

		resultMap := asAnyMap(rawResults)
		if len(resultMap) == 0 {
			return nil, false
		}

		known, ok := extractKnownRows(resultMap)
		if !ok || len(known) == 0 {
			return nil, false
		}

		matched = true
		for _, knownRow := range known {
			combined := map[string]any{}
			if input := textValue(entry["input"]); input != "" {
				combined["input"] = input
			}
			if query := textValue(entry["query"]); query != "" {
				combined["query"] = query
			}
			for key, value := range knownRow {
				combined[key] = value
			}
			rows = append(rows, combined)
		}
	}

	if !matched {
		return nil, false
	}
	return rows, true
}

func extractKnownRows(v any) ([]map[string]any, bool) {
	root, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}

	if commands, ok := root["commands"]; ok {
		if rows, ok := normalizeRows(commands); ok {
			return rows, true
		}
	}

	if rows, ok := extractSearchRows(root); ok {
		return rows, true
	}
	if rows, ok := extractRegisterRows(root); ok {
		return rows, true
	}
	if rows, ok := extractFamilyRows(root); ok {
		return rows, true
	}
	if rows, ok := extractNumberRows(root); ok {
		return rows, true
	}
	if rows, ok := extractUsageRows(root); ok {
		return rows, true
	}
	return nil, false
}

func extractSearchRows(root map[string]any) ([]map[string]any, bool) {
	worldPatentData := asAnyMap(root["ops:world-patent-data"])
	biblioSearch := asAnyMap(worldPatentData["ops:biblio-search"])
	searchResult := asAnyMap(biblioSearch["ops:search-result"])

	if exchangeDocsRaw, ok := searchResult["exchange-documents"]; ok {
		rawRows, ok := asAnySlice(exchangeDocsRaw)
		if !ok {
			if one, ok := exchangeDocsRaw.(map[string]any); ok {
				rawRows = []any{one}
			}
		}
		if len(rawRows) > 0 {
			rows := make([]map[string]any, 0, len(rawRows))
			for _, rawRow := range rawRows {
				row := flattenPublishedSearchItem(rawRow)
				rows = append(rows, row)
			}
			return rows, true
		}
	}

	if exchangeDocRaw, ok := searchResult["exchange-document"]; ok {
		rawRows, ok := asAnySlice(exchangeDocRaw)
		if !ok {
			if one, ok := exchangeDocRaw.(map[string]any); ok {
				rawRows = []any{one}
			}
		}
		if len(rawRows) > 0 {
			rows := make([]map[string]any, 0, len(rawRows))
			for _, rawRow := range rawRows {
				row := flattenPublishedSearchItem(rawRow)
				rows = append(rows, row)
			}
			return rows, true
		}
	}

	publicationRefs, ok := searchResult["ops:publication-reference"]
	if !ok {
		return nil, false
	}

	rawRows, ok := asAnySlice(publicationRefs)
	if !ok {
		if one, ok := publicationRefs.(map[string]any); ok {
			rawRows = []any{one}
		} else {
			return nil, false
		}
	}

	rows := make([]map[string]any, 0, len(rawRows))
	for _, rawRow := range rawRows {
		ref := asAnyMap(rawRow)
		row := map[string]any{
			"familyId": ref["@family-id"],
			"system":   ref["@system"],
		}

		documentID := firstDocumentID(ref["document-id"])
		row["documentIdType"] = documentID["@document-id-type"]
		row["country"] = asAnyMap(documentID["country"])["$"]
		row["docNumber"] = asAnyMap(documentID["doc-number"])["$"]
		row["kind"] = asAnyMap(documentID["kind"])["$"]
		row["pubDate"] = asAnyMap(documentID["date"])["$"]
		rows = append(rows, row)
	}
	return rows, len(rows) > 0
}

func extractFamilyRows(root map[string]any) ([]map[string]any, bool) {
	worldPatentData := asAnyMap(root["ops:world-patent-data"])
	patentFamily := asAnyMap(worldPatentData["ops:patent-family"])
	membersRaw, ok := patentFamily["ops:family-member"]
	if !ok {
		return nil, false
	}

	memberList, ok := asAnySlice(membersRaw)
	if !ok {
		if one, ok := membersRaw.(map[string]any); ok {
			memberList = []any{one}
		} else {
			return nil, false
		}
	}

	rows := make([]map[string]any, 0, len(memberList))
	for _, rawMember := range memberList {
		member := asAnyMap(rawMember)
		appRef := asAnyMap(member["application-reference"])
		pubRef := asAnyMap(member["publication-reference"])

		appDocID := firstDocumentID(appRef["document-id"])
		pubDocID := firstDocumentID(pubRef["document-id"])

		row := map[string]any{
			"familyId":     member["@family-id"],
			"appCountry":   asAnyMap(appDocID["country"])["$"],
			"appDocNumber": asAnyMap(appDocID["doc-number"])["$"],
			"appKind":      asAnyMap(appDocID["kind"])["$"],
			"appDate":      asAnyMap(appDocID["date"])["$"],
			"pubCountry":   asAnyMap(pubDocID["country"])["$"],
			"pubDocNumber": asAnyMap(pubDocID["doc-number"])["$"],
			"pubKind":      asAnyMap(pubDocID["kind"])["$"],
			"pubDate":      asAnyMap(pubDocID["date"])["$"],
		}
		rows = append(rows, row)
	}
	return rows, len(rows) > 0
}

func extractNumberRows(root map[string]any) ([]map[string]any, bool) {
	worldPatentData := asAnyMap(root["ops:world-patent-data"])
	standardization := asAnyMap(worldPatentData["ops:standardization"])
	if len(standardization) == 0 {
		return nil, false
	}

	inputContainer := asAnyMap(standardization["ops:input"])
	outputContainer := asAnyMap(standardization["ops:output"])
	inputRef := firstReferenceMap(inputContainer)
	outputRef := firstReferenceMap(outputContainer)
	inputDocID := firstDocumentID(inputRef["document-id"])
	outputDocID := firstDocumentID(outputRef["document-id"])

	row := map[string]any{
		"inputFormat":     standardization["@inputFormat"],
		"outputFormat":    standardization["@outputFormat"],
		"inputType":       referenceTypeFromMap(inputContainer),
		"inputCountry":    asAnyMap(inputDocID["country"])["$"],
		"inputDocNumber":  asAnyMap(inputDocID["doc-number"])["$"],
		"inputKind":       asAnyMap(inputDocID["kind"])["$"],
		"outputType":      referenceTypeFromMap(outputContainer),
		"outputCountry":   asAnyMap(outputDocID["country"])["$"],
		"outputDocNumber": asAnyMap(outputDocID["doc-number"])["$"],
		"outputKind":      asAnyMap(outputDocID["kind"])["$"],
	}
	return []map[string]any{row}, true
}

func extractRegisterRows(root map[string]any) ([]map[string]any, bool) {
	worldPatentData := asAnyMap(root["ops:world-patent-data"])
	registerSearch := asAnyMap(worldPatentData["ops:register-search"])
	if len(registerSearch) == 0 {
		return nil, false
	}

	registerDocs := asAnyMap(registerSearch["reg:register-documents"])
	rawDocs, ok := registerDocs["reg:register-document"]
	if !ok {
		return nil, false
	}

	docList, ok := asAnySlice(rawDocs)
	if !ok {
		if one, ok := rawDocs.(map[string]any); ok {
			docList = []any{one}
		} else {
			return nil, false
		}
	}

	rows := make([]map[string]any, 0, len(docList))
	for _, rawDoc := range docList {
		doc := asAnyMap(rawDoc)
		biblio := asAnyMap(doc["reg:bibliographic-data"])

		appDoc := asAnyMap(asAnyMap(biblio["reg:application-reference"])["reg:document-id"])
		pubDoc := registerPublicationDocumentID(biblio)
		status := registerStatusText(doc)

		row := map[string]any{
			"appCountry":     textValue(appDoc["reg:country"]),
			"appDocNumber":   textValue(appDoc["reg:doc-number"]),
			"pubCountry":     textValue(pubDoc["reg:country"]),
			"pubDocNumber":   textValue(pubDoc["reg:doc-number"]),
			"pubDate":        textValue(pubDoc["reg:date"]),
			"title":          firstTitleText(biblio["reg:invention-title"]),
			"epPatentStatus": status,
		}
		rows = append(rows, row)
	}
	return rows, len(rows) > 0
}

func extractUsageRows(root map[string]any) ([]map[string]any, bool) {
	environments, ok := asAnySlice(root["environments"])
	if !ok || len(environments) == 0 {
		return nil, false
	}

	notices := asStringSlice(asAnyMap(root["metaData"])["notices"])
	rows := make([]map[string]any, 0, len(environments)*8)
	for _, rawEnv := range environments {
		env := asAnyMap(rawEnv)
		flattened := flattenUsageEnvironment(env, notices)
		if len(flattened) == 0 {
			rows = append(rows, map[string]any{
				"environment": textValue(env["name"]),
				"notices":     strings.Join(notices, " | "),
			})
			continue
		}
		rows = append(rows, flattened...)
	}
	return rows, true
}

func flattenUsageEnvironment(env map[string]any, notices []string) []map[string]any {
	environment := textValue(env["name"])
	dimensions, ok := asAnySlice(env["dimensions"])
	if !ok || len(dimensions) == 0 {
		return nil
	}

	grouped := map[string]map[string]any{}
	for _, rawDim := range dimensions {
		dim := asAnyMap(rawDim)
		metrics, ok := asAnySlice(dim["metrics"])
		if !ok {
			continue
		}
		for _, rawMetric := range metrics {
			metric := asAnyMap(rawMetric)
			metricName := normalizeUsageMetricName(firstNonEmpty(
				textValue(metric["name"]),
				textValue(metric["metric"]),
				textValue(metric["@name"]),
			))
			if metricName == "" {
				continue
			}

			points := asAnySliceOrSingleton(metric["points"])
			if len(points) == 0 {
				points = asAnySliceOrSingleton(metric["values"])
			}
			if len(points) == 0 {
				points = []any{metric}
			}
			for _, rawPoint := range points {
				point := asAnyMap(rawPoint)
				date := firstNonEmpty(textValue(point["date"]), textValue(point["day"]))
				if date == "" {
					if epoch := toEpochInt(point["timestamp"]); epoch > 0 {
						date = time.Unix(epoch, 0).UTC().Format("2006-01-02")
					} else {
						date = textValue(point["timestamp"])
					}
				}
				value := firstNonEmpty(
					textValue(point["value"]),
					textValue(point["count"]),
					textValue(point["total"]),
					textValue(point["amount"]),
				)
				if date == "" && value == "" {
					continue
				}
				key := environment + "|" + date
				if _, exists := grouped[key]; !exists {
					grouped[key] = map[string]any{
						"environment": environment,
						"date":        date,
						"notices":     strings.Join(notices, " | "),
					}
				}
				grouped[key][metricName] = value
			}
		}
	}

	if len(grouped) == 0 {
		return nil
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, grouped[key])
	}
	return rows
}

func normalizeUsageMetricName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	switch {
	case strings.Contains(name, "message") && strings.Contains(name, "count"):
		return "message_count"
	case strings.Contains(name, "response") && strings.Contains(name, "size"):
		return "total_response_size"
	default:
		return name
	}
}

func referenceTypeFromMap(v map[string]any) string {
	for key := range v {
		if strings.HasSuffix(key, "publication-reference") {
			return "publication"
		}
		if strings.HasSuffix(key, "application-reference") {
			return "application"
		}
		if strings.HasSuffix(key, "priority-reference") {
			return "priority"
		}
	}
	return ""
}

func firstReferenceMap(v map[string]any) map[string]any {
	for _, candidate := range []string{
		"ops:publication-reference",
		"ops:application-reference",
		"ops:priority-reference",
	} {
		if ref := asAnyMap(v[candidate]); len(ref) > 0 {
			return ref
		}
	}
	for _, value := range v {
		if ref := asAnyMap(value); len(ref) > 0 {
			return ref
		}
	}
	return map[string]any{}
}

func firstDocumentID(v any) map[string]any {
	documentID := asAnyMap(v)
	if len(documentID) != 0 {
		return documentID
	}
	if docList, ok := asAnySlice(v); ok && len(docList) > 0 {
		for _, raw := range docList {
			doc := asAnyMap(raw)
			if strings.EqualFold(fmt.Sprintf("%v", doc["@document-id-type"]), "docdb") {
				return doc
			}
		}
		return asAnyMap(docList[0])
	}
	return map[string]any{}
}

func asAnyMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func asAnySlice(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []map[string]any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, item)
		}
		return out, true
	default:
		return nil, false
	}
}

func asStringSlice(v any) []string {
	items, ok := asAnySlice(v)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, textValue(item))
	}
	return out
}

func asRowMap(v any) map[string]any {
	if v == nil {
		return map[string]any{"value": ""}
	}
	switch row := v.(type) {
	case map[string]any:
		return row
	case map[string]string:
		out := make(map[string]any, len(row))
		for k, val := range row {
			out[k] = val
		}
		return out
	}

	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{"value": fmt.Sprintf("%v", v)}
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{"value": string(b)}
	}
	return out
}

func collectHeaders(rows []map[string]any) []string {
	set := map[string]struct{}{}
	for _, row := range rows {
		for key := range row {
			set[key] = struct{}{}
		}
	}
	headers := make([]string, 0, len(set))
	for key := range set {
		headers = append(headers, key)
	}
	sort.Strings(headers)
	return headers
}

func stringifyValue(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case fmt.Stringer:
		return t.String()
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("%d", rv.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", rv.Float())
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func firstTitleText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		return textValue(t["$"])
	default:
		list, ok := asAnySlice(v)
		if !ok || len(list) == 0 {
			return ""
		}
		for _, candidate := range list {
			titleMap := asAnyMap(candidate)
			lang := strings.ToLower(textValue(titleMap["@lang"]))
			if lang == "en" {
				return textValue(titleMap["$"])
			}
		}
		return textValue(asAnyMap(list[0])["$"])
	}
}

func textValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case map[string]any:
		return strings.TrimSpace(fmt.Sprintf("%v", t["$"]))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func handleError(err error) int {
	cliErr := mapError(err)
	var partial *partialError
	hasPartial := errors.As(err, &partial)

	if flagFormat == "json" {
		env := errorEnvelope{
			OK:      false,
			Error:   cliErr,
			Version: version,
		}
		if hasPartial {
			env.Partial = true
			env.Command = partial.envelope.Command
			env.Service = partial.envelope.Service
			env.Request = partial.envelope.Request
			env.Pagination = partial.envelope.Pagination
			env.Results = partial.envelope.Results
			env.Warnings = partial.envelope.Warnings
		}
		_ = writeJSON(env)
	} else {
		if hasPartial {
			// The rows fetched before the failure go to stdout through the
			// normal writer; the error still goes to stderr.
			switch flagFormat {
			case "ndjson":
				_ = writeNDJSON(partial.envelope)
			case "csv":
				_ = writeCSV(partial.envelope)
			case "table":
				_ = writeTable(partial.envelope)
			}
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", cliErr.Message)
		if cliErr.Hint != "" {
			fmt.Fprintf(os.Stderr, "Hint: %s\n", cliErr.Hint)
		}
		if hasPartial {
			for _, warning := range partial.envelope.Warnings {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
			}
		}
	}
	return mapExitCode(cliErr)
}

func mapError(err error) *epoerrors.CLIError {
	if err == nil {
		return &epoerrors.CLIError{
			Code:    0,
			Type:    "UNKNOWN",
			Message: "unknown error",
		}
	}

	var cliErr *epoerrors.CLIError
	if errors.As(err, &cliErr) {
		return cliErr
	}

	var apiErr *epoerrors.APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == 401:
			return &epoerrors.CLIError{
				Code:    apiErr.StatusCode,
				Type:    "AUTH_FAILURE",
				Message: apiErr.Error(),
				Hint:    "Verify client credentials via `epo auth check`",
			}
		case apiErr.StatusCode == 403:
			return mapForbidden(apiErr)
		case apiErr.StatusCode == 404:
			return &epoerrors.CLIError{
				Code:    apiErr.StatusCode,
				Type:    "NOT_FOUND",
				Message: apiErr.Error(),
				Hint:    build404Hint(apiErr.Error(), apiErr.Body),
			}
		case apiErr.StatusCode == 429:
			hint := "Retry with backoff; OPS may return Retry-After"
			if apiErr.RetryAfter > 0 {
				hint = fmt.Sprintf("OPS asked to wait %s before retrying (Retry-After: %d ms)", apiErr.RetryAfter, apiErr.RetryAfter.Milliseconds())
			}
			return &epoerrors.CLIError{
				Code:    apiErr.StatusCode,
				Type:    "RATE_LIMITED",
				Message: apiErr.Error(),
				Hint:    hint,
			}
		case apiErr.StatusCode >= 500:
			return &epoerrors.CLIError{
				Code:    apiErr.StatusCode,
				Type:    "SERVER_ERROR",
				Message: apiErr.Error(),
			}
		default:
			return &epoerrors.CLIError{
				Code:    apiErr.StatusCode,
				Type:    "API_ERROR",
				Message: apiErr.Error(),
			}
		}
	}

	return &epoerrors.CLIError{
		Code:    1,
		Type:    "GENERAL_ERROR",
		Message: err.Error(),
	}
}

// mapForbidden classifies a 403. OPS uses 403 for quota / fair-use rejections
// (with an X-Rejection-Reason header), for blocked developer accounts and for
// blacklisting or robot detection; bad credentials are a 400/401 at the token
// endpoint. Only the first group is a rate limit, and none of them is fixed by
// re-entering credentials.
func mapForbidden(apiErr *epoerrors.APIError) *epoerrors.CLIError {
	body := strings.ToLower(apiErr.Body)
	upstream := "OPS"
	if apiErr.Service == epoerrors.ServiceEPS {
		upstream = "EPS"
	}

	if apiErr.RejectionReason != "" || strings.Contains(body, "fair use") {
		var hint string
		switch {
		case upstream == "EPS":
			hint = "EPS rejected the request under its fair-use policy. EPS takes no credentials, so this is not a credentials problem; " +
				"its fair-use limit is IP-based (10 GB in any sliding 7-day window). Stop and retry after the window rolls forward."
		case apiErr.RejectionReason != "":
			hint = fmt.Sprintf("OPS rejected the request because a quota is exhausted (X-Rejection-Reason: %s). ", apiErr.RejectionReason) +
				"This is not a credentials problem; do not re-enter or rotate them. " +
				"The hourly quota frees up on a rolling one-hour window and the weekly quota resets at the start of the calendar week (midnight UTC). Stop and retry after the quota resets."
		default:
			hint = "OPS rejected the request under its fair-use policy (no X-Rejection-Reason header was sent). " +
				"This is not a credentials problem; do not re-enter or rotate them. " +
				"The hourly quota frees up on a rolling one-hour window and the weekly quota resets at the start of the calendar week (midnight UTC). Stop and retry later."
		}
		if apiErr.RetryAfter > 0 {
			hint += fmt.Sprintf(" %s asked to wait %s before retrying (Retry-After: %d ms).", upstream, apiErr.RetryAfter, apiErr.RetryAfter.Milliseconds())
		}
		return &epoerrors.CLIError{
			Code:    apiErr.StatusCode,
			Type:    "RATE_LIMITED",
			Message: apiErr.Error(),
			Hint:    hint,
			Reason:  apiErr.RejectionReason,
		}
	}

	hint := "OPS refused the request (403) without a quota reason. This does not by itself mean the credentials are wrong; " +
		"the response body is in the message. Do not loop on re-entering credentials; wait before retrying and check the account in the EPO developer portal if it persists."
	switch {
	case strings.Contains(body, "developer account is blocked"):
		hint = "EPO has blocked this developer account. Re-entering credentials will not help; contact EPO support."
	case upstream == "EPS":
		hint = "EPS refused the request (403). EPS takes no credentials, so this is not a credentials problem; " +
			"the response body is in the message. EPS limits are IP-based, so wait before retrying."
	}
	return &epoerrors.CLIError{
		Code:    apiErr.StatusCode,
		Type:    "AUTH_FAILURE",
		Message: apiErr.Error(),
		Hint:    hint,
	}
}

func mapExitCode(err *epoerrors.CLIError) int {
	if err == nil {
		return epoerrors.ExitGeneralError
	}
	switch err.Type {
	case "VALIDATION_ERROR":
		return epoerrors.ExitUsageError
	case "AUTH_FAILURE":
		return epoerrors.ExitAuthFailure
	case "NOT_FOUND":
		return epoerrors.ExitNotFound
	case "RATE_LIMITED":
		return epoerrors.ExitRateLimited
	case "SERVER_ERROR":
		return epoerrors.ExitServerError
	default:
		return epoerrors.ExitGeneralError
	}
}

func emitQuotaWarnings(env successEnvelope) {
	if env.Quota != nil {
		if qMap, ok := env.Quota.(map[string]int); ok {
			q := throttle.Quota{
				IndividualPerHourUsed: qMap["hourUsed"],
				RegisteredPerWeekUsed: qMap["weekUsed"],
			}
			if near, msg := q.NearLimit(); near {
				fmt.Fprintf(os.Stderr, "warning: %s\n", msg)
			}
		}
	}
	if env.Throttle != nil {
		if tMap, ok := env.Throttle.(map[string]any); ok {
			if services, ok := tMap["services"].(map[string]throttle.ServiceState); ok {
				s := throttle.State{Services: services}
				if black, msg := s.HasBlackService(); black {
					fmt.Fprintf(os.Stderr, "warning: %s\n", msg)
				}
			}
		}
	}
}

func build404Hint(msg string, body string) string {
	return "Patent not found. Verify the number and kind code (A1/A2/B1/B2). " +
		"For claims/description, use docdb format (EP.1000000.A1 --input-format docdb). " +
		"Run: epo number normalize <ref>"
}
