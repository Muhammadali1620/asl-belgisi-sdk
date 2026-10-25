package asl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"asl/internal/dto"
)

// ErrorCode is our stable error kind; business code should check it instead of ASL texts.
type ErrorCode string

// Error codes returned by the provider.
const (
	CodeInvalidArgument  ErrorCode = "INVALID_ARGUMENT"   // bad input, request was not sent
	CodeInvalidInput     ErrorCode = "INVALID_INPUT"      // ASL rejected the request data (400)
	CodeUnauthorized     ErrorCode = "UNAUTHORIZED"       // token is wrong or expired (401)
	CodeForbidden        ErrorCode = "FORBIDDEN"          // no role or no access to the object (403)
	CodeNotFound         ErrorCode = "NOT_FOUND"          // object does not exist (404)
	CodeMethodNotAllowed ErrorCode = "METHOD_NOT_ALLOWED" // wrong HTTP method (405)
	CodeConflict         ErrorCode = "CONFLICT"           // object state does not allow it (409)
	CodeGone             ErrorCode = "GONE"               // object existed but is removed (410)
	CodePayloadTooLarge  ErrorCode = "PAYLOAD_TOO_LARGE"  // body or file is too big (413)
	CodeUnsupportedMedia ErrorCode = "UNSUPPORTED_MEDIA"  // wrong Content-Type (415)
	CodeUnprocessable    ErrorCode = "UNPROCESSABLE"      // format is ok but data check failed (422)
	CodeRateLimited      ErrorCode = "RATE_LIMITED"       // too many requests (429)
	CodeRedirect         ErrorCode = "REDIRECT"           // ASL answered with redirect (307)
	CodeUnavailable      ErrorCode = "UNAVAILABLE"        // ASL is down for a short time, retry later
	CodeServerError      ErrorCode = "SERVER_ERROR"       // ASL internal error (5xx)
	CodeNetwork          ErrorCode = "NETWORK"            // connection problem, no answer
	CodeTimeout          ErrorCode = "TIMEOUT"            // request took too long
	CodeCanceled         ErrorCode = "CANCELED"           // context was canceled
	CodeDecode           ErrorCode = "DECODE"             // answer can not be read
	CodeSignature        ErrorCode = "SIGNATURE"          // signature is missing, wrong or signer failed
	CodeUnknown          ErrorCode = "UNKNOWN"
)

// Error is the only error type returned by the provider.
// Code is our kind, ASLCode is the code from ASL body, HTTPStatus is the answer status.
type Error struct {
	Code       ErrorCode       // our error kind
	HTTPStatus int             // HTTP status of ASL answer, 0 when there was no answer
	ASLCode    string          // code from ASL body, e.g. "invalid-token" or "602"
	Message    string          // short human text
	ErrorID    string          // ASL error ID for support and logs
	Service    string          // ASL service that answered
	Parameter  string          // wrong request parameter, when ASL tells it
	Method     string          // HTTP method of the request
	Path       string          // URL path of the request
	Details    []dto.ErrorItem // all error items from ASL body
	Body       []byte          // raw answer body (cut to 4 KB)
	Err        error           // low level cause (network, JSON)
}

// Error builds one line with all main fields.
func (e *Error) Error() string {
	// Format: "asl: NOT_FOUND: message (http 404, asl entity-not-found, GET /path)".
	var parts []string
	if e.HTTPStatus != 0 {
		parts = append(parts, fmt.Sprintf("http %d", e.HTTPStatus))
	}
	if e.ASLCode != "" {
		parts = append(parts, "asl "+e.ASLCode)
	}
	if e.Parameter != "" {
		parts = append(parts, "param "+e.Parameter)
	}
	if e.Path != "" {
		parts = append(parts, strings.TrimSpace(e.Method+" "+e.Path))
	}
	text := fmt.Sprintf("asl: %s: %s", e.Code, e.Message)
	if len(parts) > 0 {
		text += " (" + strings.Join(parts, ", ") + ")"
	}
	return text
}

// Unwrap returns the low level cause.
func (e *Error) Unwrap() error {
	// Lets errors.Is find context.Canceled, net errors and others.
	return e.Err
}

// Is makes errors.Is(err, asl.ErrNotFound) work by comparing only Code.
func (e *Error) Is(target error) bool {
	// Only *Error targets with the same Code match.
	var other *Error
	if !errors.As(target, &other) {
		return false
	}
	return other.Code == e.Code
}

// Sentinel errors for errors.Is checks.
var (
	ErrInvalidArgument = &Error{Code: CodeInvalidArgument}
	ErrInvalidInput    = &Error{Code: CodeInvalidInput}
	ErrUnauthorized    = &Error{Code: CodeUnauthorized}
	ErrForbidden       = &Error{Code: CodeForbidden}
	ErrNotFound        = &Error{Code: CodeNotFound}
	ErrRateLimited     = &Error{Code: CodeRateLimited}
	ErrUnavailable     = &Error{Code: CodeUnavailable}
	ErrTimeout         = &Error{Code: CodeTimeout}
	ErrNetwork         = &Error{Code: CodeNetwork}
)

// AsError returns the provider error inside err; ok is false for other errors.
func AsError(err error) (*Error, bool) {
	// Thin wrapper over errors.As for shorter calls.
	var aslErr *Error
	ok := errors.As(err, &aslErr)
	return aslErr, ok
}

// HasCode reports whether err is a provider error with one of the codes.
func HasCode(err error, codes ...ErrorCode) bool {
	// Returns false for nil and for not provider errors.
	aslErr, ok := AsError(err)
	if !ok {
		return false
	}
	for _, code := range codes {
		if aslErr.Code == code {
			return true
		}
	}
	return false
}

// IsRetryable reports whether the same request may work later.
func IsRetryable(err error) bool {
	// Rate limits, short outages, timeouts and network problems are temporary.
	return HasCode(err, CodeRateLimited, CodeUnavailable, CodeTimeout, CodeNetwork)
}

// invalidArgument builds an error for bad input found before sending the request.
func invalidArgument(param, format string, args ...any) *Error {
	// No HTTP request is made, so HTTPStatus stays 0.
	return &Error{
		Code:      CodeInvalidArgument,
		Message:   fmt.Sprintf(format, args...),
		Parameter: param,
	}
}

// notFound builds NOT_FOUND for helper methods where ASL answers 200 with empty data.
func notFound(param, message, path string) *Error {
	// HTTPStatus is 200 because ASL itself did not fail.
	return &Error{
		Code: CodeNotFound, HTTPStatus: http.StatusOK, Message: message, Parameter: param, Path: path,
	}
}

// transportError converts a network or context error into a provider error.
func transportError(ctx context.Context, method, path string, err error) *Error {
	// Context errors win, because the network error is only their result.
	result := &Error{Code: CodeNetwork, Message: err.Error(), Method: method, Path: path, Err: err}
	var netErr net.Error
	switch {
	case errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled):
		result.Code = CodeCanceled
	case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded):
		result.Code = CodeTimeout
	case errors.As(err, &netErr) && netErr.Timeout():
		result.Code = CodeTimeout
	}
	return result
}

// maxErrorBody is how much of an error body we keep in Error.Body.
const maxErrorBody = 4 << 10

// responseError builds a provider error from a not successful ASL answer.
func responseError(method, path string, status int, body []byte) *Error {
	// ASL sends errors as array, as object, or as {"globalErrors": [...]}; all are supported.
	items := parseErrorItems(body)
	result := &Error{
		HTTPStatus: status,
		Method:     method,
		Path:       path,
		Details:    items,
		Body:       cutBody(body),
	}
	if len(items) > 0 {
		first := items[0]
		result.ASLCode = first.Code
		if result.ASLCode == "" {
			result.ASLCode = first.ErrorCode.String()
		}
		result.ErrorID = first.ErrorID
		result.Service = first.Service
		result.Parameter = errorParameter(first)
		result.Message = errorMessage(first)
	}
	if result.Message == "" {
		result.Message = plainMessage(status, body)
	}
	result.Code = classify(status, items)
	return result
}

// parseErrorItems reads all known ASL error body shapes.
func parseErrorItems(body []byte) []dto.ErrorItem {
	// Returns nil when the body is not JSON (for example an HTML page from a proxy).
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var items []dto.ErrorItem
		if json.Unmarshal(body, &items) == nil {
			return items
		}
		return nil
	}
	var object dto.ErrorBody
	if json.Unmarshal(body, &object) != nil {
		return nil
	}
	if len(object.GlobalErrors) > 0 {
		return object.GlobalErrors
	}
	if object.Code == "" && object.ErrorCode == "" && object.Error == "" && len(object.Context) == 0 {
		return nil
	}
	return []dto.ErrorItem{object.ErrorItem}
}

// bracketParam finds "parameter [name]" in ASL error texts like "Incorrect value [x] of ... [name]".
var bracketParam = regexp.MustCompile(`parameter \[([^\]]+)\]`)

// errorParameter finds the name of the wrong parameter in an error item.
func errorParameter(item dto.ErrorItem) string {
	// ASL puts it in different places depending on the service; junk values are skipped.
	candidates := []string{
		item.ContextText("parameter"),
		item.JSONPath,
		item.RequestBodyJSONPath,
		item.RequestQueryJSONPath,
		item.RequestPathJSONPath,
		item.ContextText("request-body-json-path"),
	}
	if match := bracketParam.FindStringSubmatch(item.Error); match != nil {
		candidates = append(candidates, match[1])
	}
	for _, candidate := range candidates {
		if isParameterName(candidate) {
			return candidate
		}
	}
	return ""
}

// isParameterName reports whether the text looks like a field name, not like a sentence.
func isParameterName(text string) bool {
	// ASL sometimes puts a whole message or a Java class name into jsonPath.
	return text != "" && text != "null" && !strings.ContainsAny(text, " \t") &&
		!strings.Contains(text, "Exception")
}

// errorMessage builds a short text from an error item: main text, cause and wrong value.
func errorMessage(item dto.ErrorItem) string {
	// Example: "invalid-input-parameter: bad-format (value FOO)".
	main := firstText(item.Error, item.ContextText("description"), item.ContextText("specificError"),
		item.ContextText("_message"), item.ContextText("exception"), item.Code)
	if cause := item.ContextText("_cause"); cause != "" && !strings.Contains(main, cause) {
		main = strings.TrimSpace(main + ": " + cause)
		main = strings.TrimPrefix(main, ": ")
	}
	if value := item.ContextText("value"); value != "" && item.Error == "" && !strings.Contains(main, value) {
		main += " (value " + value + ")"
	}
	return main
}

// firstText returns the first useful text.
func firstText(texts ...string) string {
	// "Unknown error" from ASL tells nothing, so it is skipped.
	for _, text := range texts {
		if text != "" && text != "Unknown error" {
			return text
		}
	}
	return ""
}

// plainMessage makes a message from a not JSON body or from the status text.
func plainMessage(status int, body []byte) string {
	// HTML pages and JSON without known fields give the status text; long texts are cut.
	text := strings.Join(strings.Fields(string(body)), " ")
	if text != "" && !strings.HasPrefix(text, "<") && !strings.HasPrefix(text, "{") &&
		!strings.HasPrefix(text, "[") {
		if len(text) > 300 {
			text = text[:300] + "..."
		}
		return text
	}
	if statusText := http.StatusText(status); statusText != "" {
		return statusText
	}
	return fmt.Sprintf("unexpected HTTP status %d", status)
}

// cutBody keeps only the first part of a body.
func cutBody(body []byte) []byte {
	// Copies the data so the caller can not change our error by accident.
	if len(body) > maxErrorBody {
		body = body[:maxErrorBody]
	}
	return append([]byte(nil), body...)
}

// symbolicCodes maps ASL text codes to our error codes.
var symbolicCodes = map[string]ErrorCode{
	"invalid-token":              CodeUnauthorized,
	"token-expired":              CodeUnauthorized,
	"unauthorized":               CodeUnauthorized,
	"authentication-failed":      CodeUnauthorized,
	"access-denied":              CodeForbidden,
	"forbidden":                  CodeForbidden,
	"permission-denied":          CodeForbidden,
	"invalid-input-parameter":    CodeInvalidInput,
	"missing-request-parameter":  CodeInvalidInput,
	"validation-error":           CodeInvalidInput,
	"bad-request":                CodeInvalidInput,
	"invalid-document-reference": CodeNotFound, // referenced document does not exist
	"invalid-signature":          CodeSignature,
	"too-many-requests":          CodeRateLimited,
	"rate-limit-exceeded":        CodeRateLimited,
	"service-unavailable":        CodeUnavailable,
	"internal-error":             CodeServerError,
}

// numericCodes maps ASL number codes (doc 1.6 and real stage answers) to our error codes.
var numericCodes = map[string]ErrorCode{
	"1":   CodeUnavailable,  // service is down for a few seconds
	"100": CodeUnauthorized, // wrong login or password
	"201": CodeForbidden,    // no rights for the operation
	"301": CodeNotFound,     // GTIN is not in the order
	"306": CodeNotFound,     // order is not found
	"307": CodeConflict,     // order status does not allow the operation
	"311": CodeInvalidInput, // package type does not match the product card
	"321": CodeNotFound,     // product (GTIN) is not found for the participant
	"602": CodeInvalidInput, // bad parameter value
	"603": CodeInvalidInput, // bad parameter format
	"725": CodeNotFound,     // report is not found
}

// classify chooses our error code. Some HTTP statuses always win (401, 307, 429, 5xx);
// otherwise ASL text code, ASL number code and "not found" text are checked, then status.
func classify(status int, items []dto.ErrorItem) ErrorCode {
	// For example ASL answers 401 with body code 201: the token is wrong, so it is UNAUTHORIZED.
	if code, ok := strongStatusCode(status, items); ok {
		return code
	}
	for _, item := range items {
		if code, ok := itemCode(item); ok {
			return code
		}
	}
	return statusCode(status)
}

// strongStatusCode returns the code for statuses that do not depend on the body.
func strongStatusCode(status int, items []dto.ErrorItem) (ErrorCode, bool) {
	// A 5xx answer with ASL code 1 means a short outage.
	switch {
	case status == http.StatusUnauthorized, status == http.StatusTemporaryRedirect,
		status == http.StatusPermanentRedirect, status == http.StatusTooManyRequests,
		status == http.StatusRequestEntityTooLarge, status == http.StatusUnsupportedMediaType,
		status == http.StatusMethodNotAllowed:
		return statusCode(status), true
	case status >= http.StatusInternalServerError:
		for _, item := range items {
			if item.ErrorCode.String() == "1" {
				return CodeUnavailable, true
			}
		}
		return statusCode(status), true
	default:
		return "", false
	}
}

// itemCode maps one ASL error item to our code; ok is false when nothing is known.
func itemCode(item dto.ErrorItem) (ErrorCode, bool) {
	// Text code is checked first, because it is the most exact.
	code := strings.ToLower(item.Code)
	if mapped, ok := symbolicCodes[code]; ok {
		return mapped, true
	}
	if strings.Contains(code, "not-found") {
		return CodeNotFound, true
	}
	if mapped, ok := numericCodes[item.ErrorCode.String()]; ok {
		return mapped, true
	}
	if strings.Contains(strings.ToLower(item.Error), "not found") {
		return CodeNotFound, true
	}
	return "", false
}

// documentErrorCodes maps error codes from document processing (doc 11.4) to our codes.
var documentErrorCodes = map[string]ErrorCode{
	"code-not-found":          CodeNotFound,
	"invalid-code-status":     CodeConflict,
	"invalid-code-owner":      CodeForbidden,
	"invalid-nested-codes":    CodeInvalidInput,
	"invalid-code-properties": CodeInvalidInput,
}

// documentErrorPriority says which code wins when a document has many errors.
var documentErrorPriority = map[ErrorCode]int{
	CodeNotFound:      4,
	CodeForbidden:     3,
	CodeConflict:      2,
	CodeInvalidInput:  1,
	CodeUnprocessable: 0,
}

// DocumentErrorCode maps an ASL document error code (e.g. code-not-found) to our error code.
func DocumentErrorCode(aslCode string) ErrorCode {
	// Unknown codes become UNPROCESSABLE: ASL read the document but did not accept the data.
	code := strings.ToLower(strings.TrimSpace(aslCode))
	if mapped, ok := documentErrorCodes[code]; ok {
		return mapped
	}
	if strings.Contains(code, "not-found") {
		return CodeNotFound
	}
	if strings.Contains(code, "owner") || strings.Contains(code, "access") {
		return CodeForbidden
	}
	return CodeUnprocessable
}

// statusCode maps an HTTP status to our error code.
func statusCode(status int) ErrorCode {
	// Used when the body has no known ASL code.
	switch {
	case status == http.StatusTemporaryRedirect || status == http.StatusPermanentRedirect:
		return CodeRedirect
	case status == http.StatusBadRequest:
		return CodeInvalidInput
	case status == http.StatusUnauthorized:
		return CodeUnauthorized
	case status == http.StatusForbidden:
		return CodeForbidden
	case status == http.StatusNotFound:
		return CodeNotFound
	case status == http.StatusMethodNotAllowed:
		return CodeMethodNotAllowed
	case status == http.StatusConflict:
		return CodeConflict
	case status == http.StatusGone:
		return CodeGone
	case status == http.StatusRequestEntityTooLarge:
		return CodePayloadTooLarge
	case status == http.StatusUnsupportedMediaType:
		return CodeUnsupportedMedia
	case status == http.StatusUnprocessableEntity:
		return CodeUnprocessable
	case status == http.StatusTooManyRequests:
		return CodeRateLimited
	case status == 499:
		return CodeCanceled
	case status == http.StatusBadGateway || status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout:
		return CodeUnavailable
	case status >= 500:
		return CodeServerError
	default:
		return CodeUnknown
	}
}
