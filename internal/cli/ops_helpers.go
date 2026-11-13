package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/smcronin/epo-cli/internal/api"
	"github.com/smcronin/epo-cli/internal/auth"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

// newOPSHTTPClient builds the HTTP client used for OPS token and data
// requests. Tests replace it with a client that has an offline transport.
var newOPSHTTPClient = func() *http.Client {
	return &http.Client{Timeout: time.Duration(flagTimeout) * time.Second}
}

var (
	tokenManagersMu sync.Mutex
	tokenManagers   = map[string]*auth.TokenManager{}
)

// sharedTokenManager returns one token manager per credential pair for the
// life of the process, so the cached OAuth token is reused across requests.
func sharedTokenManager(clientID, clientSecret string) *auth.TokenManager {
	key := clientID + "\x00" + clientSecret

	tokenManagersMu.Lock()
	defer tokenManagersMu.Unlock()
	if manager, ok := tokenManagers[key]; ok {
		return manager
	}
	manager := auth.NewTokenManager(newOPSHTTPClient(), clientID, clientSecret, 2*time.Minute)
	tokenManagers[key] = manager
	return manager
}

func throttleEnvelope(resp api.Response) map[string]any {
	return map[string]any{
		"system":   resp.Metadata.Throttle.System,
		"services": resp.Metadata.Throttle.Services,
	}
}

func quotaEnvelope(resp api.Response) map[string]int {
	return map[string]int{
		"hourUsed":       resp.Metadata.Quota.IndividualPerHourUsed,
		"weekUsed":       resp.Metadata.Quota.RegisteredPerWeekUsed,
		"payingWeekUsed": resp.Metadata.Quota.RegisteredPayingPerWeekUsed,
	}
}

// batchRowError returns the typed error stored on a failed per-input row.
func batchRowError(row map[string]any) error {
	if cliErr, ok := row["error"].(*epoerrors.CLIError); ok && cliErr != nil {
		return cliErr
	}
	return &epoerrors.CLIError{
		Code:    1,
		Type:    "GENERAL_ERROR",
		Message: fmt.Sprintf("%v", row["error"]),
	}
}

// finishBatch emits per-input rows. A single input is unwrapped: its payload
// is built by single, or its typed error is returned. Multiple inputs are
// emitted as an array that keeps per-item errors.
func finishBatch(cmd *cobra.Command, service string, rows []map[string]any, single func(row map[string]any) responsePayload) error {
	if len(rows) == 1 {
		row := rows[0]
		if ok, _ := row["ok"].(bool); ok {
			return outputSuccess(cmd, single(row))
		}
		rowErr := batchRowError(row)
		if partial, _ := row["partial"].(bool); partial {
			envelope := buildSuccessEnvelope(cmd, single(row))
			envelope.OK = false
			return &partialError{cause: mapError(rowErr), envelope: envelope}
		}
		return rowErr
	}

	return outputSuccess(cmd, responsePayload{
		Service: service,
		Results: rows,
	})
}

// allFailureRow builds the per-query row for an --all run that failed. When
// the first page failed there is nothing to salvage and the row is the plain
// {query, ok:false, error}. After at least one fetched page the row is marked
// partial and salvage supplies what was fetched (request, pagination,
// results, warnings).
func allFailureRow(query string, err error, pagesFetched int, salvage func() map[string]any) map[string]any {
	row := map[string]any{
		"query": query,
		"ok":    false,
		"error": mapError(err),
	}
	if pagesFetched == 0 {
		return row
	}
	row["partial"] = true
	for key, value := range salvage() {
		row[key] = value
	}
	return row
}

// partialPagination describes how far an --all run got. failedRange is the
// page that failed; resumeRange is the --range to pass (with --all) to pick
// up from there.
func partialPagination(offset, pageSize, total, pagesFetched, returned int, failedRange string) map[string]any {
	return map[string]any{
		"offset":       offset,
		"limit":        pageSize,
		"total":        total,
		"pagesFetched": pagesFetched,
		"returned":     returned,
		"failedRange":  failedRange,
		"resumeRange":  failedRange,
	}
}

// partialAllWarning is the warning attached to a partial --all result.
// retrievable is how many results the run could have returned (the total,
// capped at the OPS ceiling where one applies); 0 when unknown.
func partialAllWarning(pagesFetched, returned, retrievable int, failedRange string) string {
	progress := fmt.Sprintf("%d results", returned)
	if retrievable > 0 {
		progress = fmt.Sprintf("%d of %d retrievable results", returned, retrievable)
	}
	pages := "pages"
	if pagesFetched == 1 {
		pages = "page"
	}
	return fmt.Sprintf("--all stopped after %d %s (%s); the rows fetched so far are included. Resume with --range %s --all.", pagesFetched, pages, progress, failedRange)
}

func validateRefType(refType string) error {
	if isOneOf(refType, "publication", "application", "priority") {
		return nil
	}
	return &epoerrors.CLIError{
		Code:    400,
		Type:    "VALIDATION_ERROR",
		Message: fmt.Sprintf("unsupported --ref-type %q", refType),
		Hint:    "Use publication, application, or priority",
	}
}

func validateInputFormat(inputFormat, hint string, options ...string) error {
	if isOneOf(inputFormat, options...) {
		return nil
	}
	return &epoerrors.CLIError{
		Code:    400,
		Type:    "VALIDATION_ERROR",
		Message: fmt.Sprintf("unsupported --input-format %q", inputFormat),
		Hint:    hint,
	}
}

func validatePubInputFormat(inputFormat string) error {
	return validateInputFormat(inputFormat, "Use auto, epodoc, or docdb", "epodoc", "docdb", "auto")
}

// buildSearchRequest builds a CQL search request (published-data or register).
// rangeValue is sent as the Range header when non-empty.
func buildSearchRequest(path, query, rangeValue string, usePost bool) api.Request {
	request := api.Request{
		Method: http.MethodGet,
		Path:   path,
		Accept: "application/json",
	}
	if v := strings.TrimSpace(rangeValue); v != "" {
		request.Headers = map[string]string{
			"Range": v,
		}
	}
	if usePost {
		request.Method = http.MethodPost
		request.ContentType = "text/plain"
		request.Body = []byte("q=" + url.QueryEscape(query))
	} else {
		request.Query = url.Values{}
		request.Query.Set("q", query)
	}
	return request
}

func searchRequestMeta(request api.Request) map[string]any {
	requestMeta := map[string]any{
		"method": request.Method,
		"path":   request.Path,
		"query":  compactQuery(request.Query),
	}
	if len(request.Headers) > 0 {
		requestMeta["headers"] = request.Headers
	}
	return requestMeta
}
