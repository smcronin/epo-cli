package cli

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/smcronin/epo-cli/internal/api"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var (
		legalInputFormat string
		registerRef      string
	)

	cmd := &cobra.Command{
		Use:   "status <reference>",
		Short: "Build combined legal + register + procedural status timeline",
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

			legalFormat := legalInputFormat
			if legalFormat == "auto" {
				if looksDocdbPublicationReference(reference) {
					legalFormat = "docdb"
				} else {
					legalFormat = "epodoc"
				}
			}

			legalResp, legalErr := executeOPSRequest(cmd.Context(), api.Request{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("/legal/publication/%s/%s", legalFormat, reference),
				Accept: "application/json",
			})
			if legalErr != nil {
				return legalErr
			}
			legalParsed, legalWarnings := parseJSONBody(legalResp.Body)
			legalEvents := flattenLegalEvents(legalParsed)

			effectiveRegisterRef := strings.TrimSpace(registerRef)
			if effectiveRegisterRef == "" && looksApplicationEpRef(reference) {
				effectiveRegisterRef = reference
			}

			status := map[string]any{
				"reference":        reference,
				"legalEventCount":  len(legalEvents),
				"legalEvents":      legalEvents,
				"registerRef":      effectiveRegisterRef,
				"registerSummary":  map[string]any{},
				"proceduralSteps":  []map[string]any{},
				"timelineWarnings": legalWarnings,
			}
			if effectiveRegisterRef == "" {
				status["timelineWarnings"] = append(toAnyStringSlice(status["timelineWarnings"]), "Register reference unresolved. Provide --register-ref EP<application> for combined register timeline.")
				return outputSuccess(cmd, responsePayload{
					Service: "status",
					Results: status,
				})
			}

			registerResp, registerErr := executeOPSRequest(cmd.Context(), api.Request{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("/register/application/epodoc/%s", effectiveRegisterRef),
				Accept: "application/json",
			})
			if registerErr == nil {
				registerParsed, _ := parseJSONBody(registerResp.Body)
				status["registerSummary"] = summarizeRegisterPayload(registerParsed)
			} else {
				status["timelineWarnings"] = append(toAnyStringSlice(status["timelineWarnings"]), "Register get failed: "+registerErr.Error())
			}

			procResp, procErr := executeOPSRequest(cmd.Context(), api.Request{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("/register/application/epodoc/%s/procedural-steps", effectiveRegisterRef),
				Accept: "application/json",
			})
			if procErr == nil {
				procParsed, _ := parseJSONBody(procResp.Body)
				status["proceduralSteps"] = collectProceduralStepLabels(procParsed)
			} else {
				status["timelineWarnings"] = append(toAnyStringSlice(status["timelineWarnings"]), "Register procedural-steps failed: "+procErr.Error())
			}

			return outputSuccess(cmd, responsePayload{
				Service: "status",
				Results: status,
			})
		},
	}

	cmd.Flags().StringVar(&legalInputFormat, "input-format", "auto", "Legal input format: auto, docdb, epodoc")
	cmd.Flags().StringVar(&registerRef, "register-ref", "", "Explicit register application reference (epodoc), e.g. EP99203729")
	return cmd
}

func collectProceduralStepLabels(v any) []map[string]any {
	out := []map[string]any{}
	seen := map[string]struct{}{}
	var walk func(any)
	walk = func(node any) {
		switch t := node.(type) {
		case map[string]any:
			for key, child := range t {
				if key == "reg:procedural-step" {
					for _, rawStep := range asAnySliceOrSingleton(child) {
						step := asAnyMap(rawStep)
						if len(step) == 0 {
							continue
						}
						code := strings.TrimSpace(textValue(step["reg:procedural-step-code"]))
						description := strings.TrimSpace(proceduralStepDescription(step["reg:procedural-step-text"]))
						phase := strings.TrimSpace(textValue(step["@procedure-step-phase"]))
						date := strings.TrimSpace(proceduralStepDate(step["reg:procedural-step-date"]))
						signature := strings.Join([]string{code, description, date, phase}, "|")
						if _, exists := seen[signature]; exists {
							continue
						}
						seen[signature] = struct{}{}
						row := map[string]any{}
						if code != "" {
							row["code"] = code
						}
						if description != "" {
							row["description"] = description
						}
						if date != "" {
							row["date"] = date
						}
						if phase != "" {
							row["phase"] = phase
						}
						if len(row) > 0 {
							out = append(out, row)
						}
					}
				}
				walk(child)
			}
		case []any:
			for _, child := range t {
				walk(child)
			}
		}
	}
	walk(v)
	sort.Slice(out, func(i, j int) bool {
		leftDate := textValue(out[i]["date"])
		rightDate := textValue(out[j]["date"])
		if leftDate != rightDate {
			return leftDate < rightDate
		}
		leftCode := textValue(out[i]["code"])
		rightCode := textValue(out[j]["code"])
		return leftCode < rightCode
	})
	return out
}

func proceduralStepDescription(v any) string {
	switch t := v.(type) {
	case map[string]any:
		if strings.EqualFold(textValue(t["@step-text-type"]), "STEP_DESCRIPTION") {
			return textValue(t["$"])
		}
		return firstNonEmpty(textValue(t["$"]), textValue(t["text"]))
	case []any:
		fallback := ""
		for _, item := range t {
			text := proceduralStepDescription(item)
			if text == "" {
				continue
			}
			itemMap := asAnyMap(item)
			if strings.EqualFold(textValue(itemMap["@step-text-type"]), "STEP_DESCRIPTION") {
				return text
			}
			if fallback == "" {
				fallback = text
			}
		}
		return fallback
	default:
		return textValue(t)
	}
}

func proceduralStepDate(v any) string {
	switch t := v.(type) {
	case map[string]any:
		return firstNonEmpty(textValue(t["reg:date"]), textValue(t["date"]))
	case []any:
		best := ""
		for _, item := range t {
			date := proceduralStepDate(item)
			if date == "" {
				continue
			}
			if best == "" || date < best {
				best = date
			}
		}
		return best
	default:
		return ""
	}
}

func toAnyStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			text := textValue(item)
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return []string{}
	}
}
