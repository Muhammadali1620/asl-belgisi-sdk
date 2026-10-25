package asl

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"asl/internal/dto"
)

// Content types used by ASL.
const (
	contentTypeJSON = "application/json;charset=UTF-8"
	contentTypeForm = "application/x-www-form-urlencoded"
	acceptJSON      = "application/json"
	acceptAny       = "*/*"
)

// maxBodySize limits answers we read; ASL export files are up to 10 MB.
const maxBodySize = 64 << 20

// maxRetryWait is the longest wait between two tries.
const maxRetryWait = 10 * time.Second

// request describes one ASL call.
type request struct {
	method string
	path   string
	query  url.Values
	json   any        // body sent as JSON
	form   url.Values // body sent as form (refresh token method)
	noAuth bool       // do not send the Authorization header
	accept string     // Accept header, JSON by default
}

// call sends the request and decodes a successful JSON answer into out (out may be nil).
func (c *Client) call(ctx context.Context, req request, out any) error {
	// Answer body is decoded only for 2xx statuses; other statuses become *Error.
	body, err := c.send(ctx, req)
	if err != nil {
		return err
	}
	if out == nil || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &Error{
			Code:       CodeDecode,
			HTTPStatus: http.StatusOK,
			Message:    "can not read ASL answer: " + err.Error(),
			Method:     req.method,
			Path:       req.path,
			Body:       cutBody(body),
			Err:        err,
		}
	}
	return nil
}

// get sends a GET request and decodes the JSON answer into out.
func (c *Client) get(ctx context.Context, path string, values query, out any) error {
	// values may be nil when the method has no query.
	return c.call(ctx, request{method: http.MethodGet, path: path, query: url.Values(values)}, out)
}

// post sends a POST request with a JSON body and decodes the JSON answer into out.
func (c *Client) post(ctx context.Context, path string, values query, body, out any) error {
	// body may be nil: then an empty body is sent.
	req := request{method: http.MethodPost, path: path, query: url.Values(values), json: body}
	return c.call(ctx, req, out)
}

// download sends a GET request and returns the raw answer bytes (zip, image).
func (c *Client) download(ctx context.Context, path string) ([]byte, error) {
	// Accept any type, because ASL sends files with different content types.
	return c.send(ctx, request{method: http.MethodGet, path: path, accept: acceptAny})
}

// send makes the HTTP call with retries and returns the raw body of a 2xx answer.
func (c *Client) send(ctx context.Context, req request) ([]byte, error) {
	// Only GET is retried, so a document or order is never created two times.
	payload, contentType, err := encodeBody(req)
	if err != nil {
		return nil, err
	}
	retries := 0
	if req.method == http.MethodGet {
		retries = c.maxRetries
	}
	for attempt := 0; ; attempt++ {
		body, retryAfter, err := c.sendOnce(ctx, req, payload, contentType)
		if err == nil {
			return body, nil
		}
		if attempt >= retries || !IsRetryable(err) || ctx.Err() != nil {
			return nil, err
		}
		if waitErr := sleep(ctx, c.backoff(attempt, retryAfter)); waitErr != nil {
			return nil, transportError(ctx, req.method, req.path, waitErr)
		}
	}
}

// sendOnce makes one HTTP try; retryAfter is the server wait hint for 429 answers.
func (c *Client) sendOnce(
	ctx context.Context, req request, payload []byte, contentType string,
) ([]byte, time.Duration, error) {
	// Each try has its own timeout, so one slow try does not eat the time of the next one.
	tryCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	httpReq, err := c.newHTTPRequest(tryCtx, req, payload, contentType)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, transportError(tryCtx, req.method, req.path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, 0, transportError(tryCtx, req.method, req.path, err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, 0, nil
	}
	aslErr := responseError(req.method, req.path, resp.StatusCode, body)
	if location := resp.Header.Get("Location"); location != "" && aslErr.Code == CodeRedirect {
		aslErr.Message = "redirect to " + location
	}
	return nil, parseRetryAfter(resp.Header.Get("Retry-After")), aslErr
}

// newHTTPRequest builds the HTTP request with URL, headers and body.
func (c *Client) newHTTPRequest(
	ctx context.Context, req request, payload []byte, contentType string,
) (*http.Request, error) {
	// GetBody is set by http.NewRequest for bytes.Reader, so HTTP/2 can resend the body if needed.
	target := c.baseURL.JoinPath(req.path)
	if len(req.query) > 0 {
		target.RawQuery = req.query.Encode()
	}
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.method, target.String(), reader)
	if err != nil {
		return nil, invalidArgument("path", "can not build request: %v", err)
	}
	httpReq.Header.Set("Accept", valueOrText(req.accept, acceptJSON))
	httpReq.Header.Set("User-Agent", c.userAgent)
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	if !req.noAuth {
		if token := c.Token(); token != "" {
			httpReq.Header.Set("Authorization", "Bearer "+token)
		}
	}
	return httpReq, nil
}

// encodeBody turns the request body into bytes and returns its content type.
func encodeBody(req request) ([]byte, string, error) {
	// Form body wins over JSON body; a request without body has no content type.
	switch {
	case req.form != nil:
		return []byte(req.form.Encode()), contentTypeForm, nil
	case req.json != nil:
		payload, err := marshalJSON(req.json)
		if err != nil {
			return nil, "", invalidArgument("body", "can not encode request body: %v", err)
		}
		return payload, contentTypeJSON, nil
	case req.method == http.MethodPost:
		return []byte{}, contentTypeJSON, nil
	default:
		return nil, "", nil
	}
}

// marshalJSON encodes without HTML escaping, so codes with <, > and & stay readable.
func marshalJSON(value any) ([]byte, error) {
	// json.Encoder adds a new line at the end; it is removed.
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buffer.Bytes(), "\n"), nil
}

// sortedJSON encodes a value with keys sorted A-Z on all levels (ASL needs it for documents).
func sortedJSON(value any) ([]byte, error) {
	// Go maps are always encoded with sorted keys, so we pass the value through a generic map.
	raw, err := marshalJSON(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var generic any
	if err := decoder.Decode(&generic); err != nil {
		return nil, err
	}
	return marshalJSON(generic)
}

// envelope builds the base64 document body and signs it when a signer is set.
func (c *Client) envelope(ctx context.Context, document any) (dto.DocumentEnvelope, error) {
	// The signer gets the same JSON bytes that are inside base64.
	raw, err := sortedJSON(document)
	if err != nil {
		return dto.DocumentEnvelope{}, invalidArgument("documentBody", "can not encode: %v", err)
	}
	result := dto.DocumentEnvelope{DocumentBody: base64.StdEncoding.EncodeToString(raw)}
	if c.signer == nil {
		return result, nil
	}
	signature, err := c.signer.Sign(ctx, raw)
	if err != nil {
		return dto.DocumentEnvelope{}, &Error{Code: CodeSignature, Message: err.Error(), Err: err}
	}
	result.Signature = signature
	return result, nil
}

// sendDocument posts a document envelope and returns the new document ID.
func (c *Client) sendDocument(
	ctx context.Context, path string, document any,
) (*dto.DocumentIDResponse, error) {
	// Shared by all methods that send base64 documents.
	body, err := c.envelope(ctx, document)
	if err != nil {
		return nil, err
	}
	var out dto.DocumentIDResponse
	if err := c.post(ctx, path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// backoff returns how long to wait before the next try.
func (c *Client) backoff(attempt int, retryAfter time.Duration) time.Duration {
	// Server hint is used when it is given; otherwise the wait doubles each try.
	wait := c.retryWait << attempt
	if retryAfter > wait {
		wait = retryAfter
	}
	return min(wait, maxRetryWait)
}

// parseRetryAfter reads the Retry-After header in seconds.
func parseRetryAfter(value string) time.Duration {
	// HTTP date format is rare for ASL, so it is ignored.
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// sleep waits for the duration or until the context ends.
func sleep(ctx context.Context, wait time.Duration) error {
	// Uses a timer, so it does not leak when the context ends first.
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// valueOrText returns value, or fallback when value is empty.
func valueOrText(value, fallback string) string {
	// Small helper for string defaults.
	if value == "" {
		return fallback
	}
	return value
}

// query is a small builder for URL query values that skips empty values.
type query url.Values

// set adds a text value when it is not empty.
func (q query) set(key, value string) query {
	// Empty values are not sent, because ASL may treat "" as a wrong value.
	if value != "" {
		url.Values(q).Set(key, value)
	}
	return q
}

// setInt adds a number when it is positive.
func (q query) setInt(key string, value int) query {
	// Zero means "not set" for all ASL number filters.
	if value > 0 {
		url.Values(q).Set(key, strconv.Itoa(value))
	}
	return q
}

// setIntPtr adds a number when the pointer is not nil (zero is a valid value here).
func (q query) setIntPtr(key string, value *int) query {
	// Used for cursors like lastIndex, where 0 is a real index.
	if value != nil {
		url.Values(q).Set(key, strconv.Itoa(*value))
	}
	return q
}

// setTime adds a date-time in ASL format when it is set.
func (q query) setTime(key string, value *dto.DateTime) query {
	// Nil and zero times are skipped.
	if value != nil && !value.IsZero() {
		url.Values(q).Set(key, value.String())
	}
	return q
}

// setBool adds a boolean when the pointer is not nil.
func (q query) setBool(key string, value *bool) query {
	// Nil means "do not filter".
	if value != nil {
		url.Values(q).Set(key, strconv.FormatBool(*value))
	}
	return q
}

// setList adds a list as repeated keys: key=a&key=b.
func setList[T ~string](q query, key string, values []T) query {
	// Empty items are skipped.
	for _, value := range values {
		if value != "" {
			url.Values(q).Add(key, string(value))
		}
	}
	return q
}
