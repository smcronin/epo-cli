package cli

import (
	"fmt"
	"net/http"
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
				"proceduralSteps":  []any{},
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

func collectProceduralStepLabels(v any) []string {
	values := map[string]struct{}{}
	collectStringValuesByKey(v, "reg:step-name", values)
	collectStringValuesByKey(v, "reg:procedural-step", values)
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return out
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
