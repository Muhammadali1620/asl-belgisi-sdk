package test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"

	"asl/internal/provider/asl"
)

// liveTimeout is the time limit for one live test case.
const liveTimeout = 3 * time.Minute

var (
	liveOnce   sync.Once
	liveShared *asl.Client
	liveErr    error
)

// live returns the shared stage client, or skips the test when ASL_API_KEY is not set.
func live(t *testing.T) *asl.Client {
	t.Helper()
	if cfg.Token == "" {
		t.Skip("ASL_API_KEY is not set: live test skipped")
	}
	liveOnce.Do(func() {
		liveShared, liveErr = asl.New(asl.Config{BaseURL: cfg.BaseURL, Token: cfg.Token})
	})
	if liveErr != nil {
		t.Fatalf("can not create live client: %v", liveErr)
	}
	return liveShared
}

// liveWithToken returns a stage client with another token (for UNAUTHORIZED tests).
func liveWithToken(t *testing.T, token string) *asl.Client {
	t.Helper()
	live(t)
	client, err := asl.New(asl.Config{BaseURL: cfg.BaseURL, Token: token, MaxRetries: -1})
	if err != nil {
		t.Fatalf("can not create client: %v", err)
	}
	return client
}

// testContext returns a context that ends with the test or after the timeout.
func testContext(t *testing.T, timeout time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	t.Cleanup(cancel)
	return ctx
}

// env is what a table case gets: the context of the case and the provider.
type env struct {
	ctx context.Context
	asl asl.Provider
}

// testCase is one table row: a name and a function that runs and checks one call.
type testCase struct {
	name string
	run  func(t *testing.T, e env)
}

// pass makes a case that must succeed; checks run on the result.
func pass[T any](name string, call func(e env) (T, error), checks ...func(*testing.T, T)) testCase {
	return testCase{name: name, run: func(t *testing.T, e env) {
		result, err := call(e)
		requireOK(t, err)
		for _, check := range checks {
			check(t, result)
		}
	}}
}

// fail makes a case that must return *asl.Error with one of the codes.
func fail[T any](name string, call func(e env) (T, error), codes ...asl.ErrorCode) testCase {
	return testCase{name: name, run: func(t *testing.T, e env) {
		_, err := call(e)
		requireCode(t, err, codes...)
	}}
}

// runCases runs each case as a subtest, so one case can be run with -run Test/name.
func runCases(t *testing.T, client asl.Provider, cases []testCase) {
	t.Helper()
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			item.run(t, env{ctx: testContext(t, liveTimeout), asl: client})
		})
	}
}

// requireOK fails the test when err is not nil and prints all error details.
func requireOK(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if aslErr, ok := asl.AsError(err); ok {
		t.Fatalf("unexpected error: %v\n  details: %+v\n  body: %s", err, aslErr.Details, aslErr.Body)
	}
	t.Fatalf("unexpected error: %v", err)
}

// requireCode checks that err is *asl.Error with one of the codes and that its fields make sense.
func requireCode(t *testing.T, err error, codes ...asl.ErrorCode) *asl.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %v, got nil", codes)
	}
	aslErr, ok := asl.AsError(err)
	if !ok {
		t.Fatalf("expected *asl.Error, got %T: %v", err, err)
	}
	if !slices.Contains(codes, aslErr.Code) {
		t.Fatalf("expected code %v, got %s: %v\n  body: %s", codes, aslErr.Code, err, aslErr.Body)
	}
	if aslErr.Message == "" {
		t.Errorf("error has empty message: %#v", aslErr)
	}
	if aslErr.Code == asl.CodeInvalidArgument && aslErr.HTTPStatus != 0 {
		t.Errorf("INVALID_ARGUMENT must not have HTTP status, got %d", aslErr.HTTPStatus)
	}
	if !errors.Is(err, &asl.Error{Code: aslErr.Code}) {
		t.Errorf("errors.Is does not match own code %s", aslErr.Code)
	}
	t.Logf("got expected error: %v", err)
	return aslErr
}

// recorded is one request received by the mock server.
type recorded struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   []byte
	Proto  int // HTTP major version: 2 for HTTP/2
}

// reply is a mock answer; Next lets the answer change on each call (for retry tests).
type reply struct {
	Status int
	Body   string
	Header map[string]string
	Delay  time.Duration
	Next   func(call int) reply
}

// mockServer is a local HTTPS server with HTTP/2 that records requests.
type mockServer struct {
	server *httptest.Server
	mu     sync.Mutex
	calls  []recorded
	routes map[string]reply
}

// newMock starts a mock server; it is closed when the test ends.
func newMock(t *testing.T) *mockServer {
	t.Helper()
	mock := &mockServer{routes: map[string]reply{}}
	mock.server = httptest.NewUnstartedServer(http.HandlerFunc(mock.handle))
	mock.server.EnableHTTP2 = true
	mock.server.StartTLS()
	t.Cleanup(mock.server.Close)
	return mock
}

// on sets the answer for "METHOD /path".
func (m *mockServer) on(method, path string, answer reply) *mockServer {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routes[method+" "+path] = answer
	return m
}

// client builds a provider client for this server; options can change the config.
func (m *mockServer) client(t *testing.T, options ...func(*asl.Config)) *asl.Client {
	t.Helper()
	config := asl.Config{
		BaseURL:    m.server.URL,
		Token:      "test-token",
		HTTPClient: m.server.Client(),
		RetryWait:  time.Millisecond,
	}
	for _, option := range options {
		option(&config)
	}
	client, err := asl.New(config)
	if err != nil {
		t.Fatalf("can not create mock client: %v", err)
	}
	return client
}

// requests returns all recorded requests.
func (m *mockServer) requests() []recorded {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.calls)
}

// last returns the last recorded request or fails the test.
func (m *mockServer) last(t *testing.T) recorded {
	t.Helper()
	calls := m.requests()
	if len(calls) == 0 {
		t.Fatal("mock server got no requests")
	}
	return calls[len(calls)-1]
}

// handle records the request and writes the configured answer (404 when no route).
func (m *mockServer) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	m.mu.Lock()
	m.calls = append(m.calls, recorded{
		Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(),
		Header: r.Header.Clone(), Body: body, Proto: r.ProtoMajor,
	})
	count := 0
	for _, call := range m.calls {
		if call.Method == r.Method && call.Path == r.URL.Path {
			count++
		}
	}
	answer, ok := m.routes[r.Method+" "+r.URL.Path]
	m.mu.Unlock()
	if !ok {
		answer = reply{Status: http.StatusNotFound, Body: `[{"code":"entity-not-found","service":"mock"}]`}
	}
	if answer.Next != nil {
		answer = answer.Next(count)
	}
	if answer.Delay > 0 {
		select {
		case <-time.After(answer.Delay):
		case <-r.Context().Done():
			return
		}
	}
	for key, value := range answer.Header {
		w.Header().Set(key, value)
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(valueOrStatus(answer.Status))
	_, _ = io.WriteString(w, answer.Body)
}

// valueOrStatus returns 200 when status is 0.
func valueOrStatus(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}
