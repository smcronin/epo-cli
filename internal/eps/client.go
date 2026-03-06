package eps

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
)

const DefaultBaseURL = "https://data.epo.org/publication-server/rest/v1.2"

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type DocumentFormatLink struct {
	Format string `json:"format"`
	URL    string `json:"url"`
}

var (
	anchorHrefPattern      = regexp.MustCompile(`(?is)<a\b[^>]*?\bhref\s*=\s*["']([^"']+)["'][^>]*>`)
	publicationDatePattern = regexp.MustCompile(`(?i)/publication-dates/(\d{8})/patents\b`)
	patentIDPattern        = regexp.MustCompile(`(?i)/patents/([A-Z0-9]+)\b`)
	documentFormatPattern  = regexp.MustCompile(`(?i)/document\.(xml|html|pdf|zip)\b`)
)

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(DefaultBaseURL, "/"),
	}
}

func (c *Client) SetBaseURL(baseURL string) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return
	}
	c.baseURL = strings.TrimRight(baseURL, "/")
}

func (c *Client) Do(ctx context.Context, method, requestPath string, query url.Values, accept string) (Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}

	u, err := c.buildURL(requestPath, query)
	if err != nil {
		return Response{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return Response{}, fmt.Errorf("build request: %w", err)
	}
	if strings.TrimSpace(accept) != "" {
		httpReq.Header.Set("Accept", strings.TrimSpace(accept))
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("execute request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("read response body: %w", err)
	}

	resp := Response{
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header.Clone(),
		Body:       body,
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return resp, &epoerrors.APIError{
			StatusCode: httpResp.StatusCode,
			Message:    "EPS request failed",
			Body:       shortBody(body, 1000),
		}
	}
	return resp, nil
}

func (c *Client) ListPublicationDates(ctx context.Context) ([]string, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/publication-dates", nil, "text/html")
	if err != nil {
		return nil, err
	}
	return ParsePublicationDates(resp.Body), nil
}

func (c *Client) ListPatents(ctx context.Context, publicationDate string) ([]string, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/publication-dates/"+strings.TrimSpace(publicationDate)+"/patents", nil, "text/html")
	if err != nil {
		return nil, err
	}
	return ParsePatentIDs(resp.Body), nil
}

func (c *Client) ListDocumentFormats(ctx context.Context, patentNumber string) ([]DocumentFormatLink, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/patents/"+strings.TrimSpace(patentNumber), nil, "text/html")
	if err != nil {
		return nil, err
	}
	return ParseDocumentFormats(resp.Body), nil
}

func (c *Client) FetchDocument(ctx context.Context, patentNumber, format string) (Response, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	patentNumber = strings.TrimSpace(patentNumber)
	return c.Do(ctx, http.MethodGet, "/patents/"+patentNumber+"/document."+format, nil, "*/*")
}

func (c *Client) buildURL(requestPath string, query url.Values) (string, error) {
	base, err := url.Parse(strings.TrimRight(c.baseURL, "/"))
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}

	cleanPath := "/" + strings.Trim(strings.TrimSpace(requestPath), "/")
	base.Path = strings.TrimRight(base.Path, "/") + cleanPath
	if query != nil {
		base.RawQuery = query.Encode()
	}
	return base.String(), nil
}

func ParsePublicationDates(body []byte) []string {
	unique := map[string]struct{}{}
	for _, href := range parseAnchorHrefs(body) {
		match := publicationDatePattern.FindStringSubmatch(strings.ToLower(href))
		if len(match) < 2 {
			continue
		}
		unique[match[1]] = struct{}{}
	}
	out := make([]string, 0, len(unique))
	for date := range unique {
		out = append(out, date)
	}
	sort.Strings(out)
	return out
}

func ParsePatentIDs(body []byte) []string {
	unique := map[string]struct{}{}
	for _, href := range parseAnchorHrefs(body) {
		match := patentIDPattern.FindStringSubmatch(strings.ToUpper(href))
		if len(match) < 2 {
			continue
		}
		id := strings.ToUpper(strings.TrimSpace(match[1]))
		if id == "" {
			continue
		}
		unique[id] = struct{}{}
	}
	out := make([]string, 0, len(unique))
	for id := range unique {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func ParseDocumentFormats(body []byte) []DocumentFormatLink {
	byFormat := map[string]DocumentFormatLink{}
	for _, href := range parseAnchorHrefs(body) {
		match := documentFormatPattern.FindStringSubmatch(strings.ToLower(href))
		if len(match) < 2 {
			continue
		}
		format := strings.ToLower(strings.TrimSpace(match[1]))
		if format == "" {
			continue
		}
		link := DocumentFormatLink{
			Format: format,
			URL:    strings.TrimSpace(href),
		}
		byFormat[format] = link
	}
	keys := make([]string, 0, len(byFormat))
	for format := range byFormat {
		keys = append(keys, format)
	}
	sort.Strings(keys)
	out := make([]DocumentFormatLink, 0, len(keys))
	for _, format := range keys {
		out = append(out, byFormat[format])
	}
	return out
}

func parseAnchorHrefs(body []byte) []string {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil
	}
	matches := anchorHrefPattern.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	hrefs := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		href := strings.TrimSpace(string(match[1]))
		if href == "" {
			continue
		}
		hrefs = append(hrefs, href)
	}
	return hrefs
}

func shortBody(body []byte, max int) string {
	s := strings.TrimSpace(string(body))
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
