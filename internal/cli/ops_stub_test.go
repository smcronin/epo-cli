package cli

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/smcronin/epo-cli/internal/auth"
	"github.com/spf13/cobra"
)

// opsStub is an offline http.RoundTripper standing in for OPS. It answers the
// OAuth token request itself and delegates every other request to respond.
type opsStub struct {
	mu         sync.Mutex
	tokenCalls int
	requests   []*http.Request
	respond    func(req *http.Request) (status int, header http.Header, body string)
}

func (s *opsStub) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, header, body := http.StatusOK, http.Header{}, ""
	if strings.HasSuffix(req.URL.Path, "/auth/accesstoken") {
		s.tokenCalls++
		body = `{"access_token":"test-token","expires_in":"1200","token_type":"BearerToken"}`
	} else {
		s.requests = append(s.requests, req)
		status, header, body = s.respond(req)
		if header == nil {
			header = http.Header{}
		}
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func (s *opsStub) dataRequests() []*http.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*http.Request(nil), s.requests...)
}

// installOPSStub routes all OPS traffic for the test to an offline stub and
// supplies throwaway credentials, so command paths run without the network.
func installOPSStub(t *testing.T, respond func(req *http.Request) (int, http.Header, string)) *opsStub {
	t.Helper()

	t.Setenv("EPO_CLIENT_ID", "test-client-id")
	t.Setenv("EPO_CLIENT_SECRET", "test-client-secret")
	t.Setenv("EPO_CLI_CONFIG_FILE", filepath.Join(t.TempDir(), "config.json"))

	stub := &opsStub{respond: respond}
	original := newOPSHTTPClient
	newOPSHTTPClient = func() *http.Client {
		return &http.Client{Transport: stub}
	}
	resetSharedTokenManagers()
	t.Cleanup(func() {
		newOPSHTTPClient = original
		resetSharedTokenManagers()
	})
	return stub
}

func resetSharedTokenManagers() {
	tokenManagersMu.Lock()
	defer tokenManagersMu.Unlock()
	tokenManagers = map[string]*auth.TokenManager{}
}

func opsNotFound(req *http.Request) (int, http.Header, string) {
	return http.StatusNotFound, nil, `<error><code>404</code><message>No results found</message></error>`
}

// runForTest executes a freshly built command tree with args and returns what
// it wrote to stdout plus the error cobra returned.
func runForTest(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)

	var runErr error
	out := captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	return out, runErr
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	captured := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(reader)
		captured <- string(data)
	}()

	fn()

	os.Stdout = original
	_ = writer.Close()
	return <-captured
}

// setGlobalFlag overrides a persistent-flag variable for one test.
func setGlobalFlag[T any](t *testing.T, target *T, value T) {
	t.Helper()
	original := *target
	*target = value
	t.Cleanup(func() { *target = original })
}
