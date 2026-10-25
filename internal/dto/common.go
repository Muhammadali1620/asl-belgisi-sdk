package dto

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// DateTimeLayout is the date-time format we send to ASL (UTC with milliseconds).
const DateTimeLayout = "2006-01-02T15:04:05.000Z"

// DateLayout is the date-only format used by ASL.
const DateLayout = "2006-01-02"

// dateTimeLayouts are all date-time formats we accept from ASL.
var dateTimeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05.999999999Z0700",
	DateLayout,
}

// DateTime is a time value in ASL format; zero value is sent as null.
type DateTime struct {
	time.Time
}

// NewDateTime wraps a time value into DateTime.
func NewDateTime(t time.Time) DateTime {
	return DateTime{Time: t}
}

// NewDateTimePtr wraps a time value into a DateTime pointer for optional fields.
func NewDateTimePtr(t time.Time) *DateTime {
	return &DateTime{Time: t}
}

// String returns the time in ASL format.
func (d DateTime) String() string {
	return d.UTC().Format(DateTimeLayout)
}

// MarshalJSON writes the time in ASL format, or null when time is zero.
func (d DateTime) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.Quote(d.String())), nil
}

// UnmarshalJSON reads every date-time format that ASL returns.
func (d *DateTime) UnmarshalJSON(data []byte) error {
	text, ok := jsonText(data)
	if !ok {
		d.Time = time.Time{}
		return nil
	}
	parsed, err := parseTime(text)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

// Date is a date without time (YYYY-MM-DD); ASL may also send full date-time here.
type Date struct {
	time.Time
}

// MarshalJSON writes the date as YYYY-MM-DD, or null when date is zero.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.Quote(d.Format(DateLayout))), nil
}

// UnmarshalJSON reads a date or a full date-time.
func (d *Date) UnmarshalJSON(data []byte) error {
	text, ok := jsonText(data)
	if !ok {
		d.Time = time.Time{}
		return nil
	}
	parsed, err := parseTime(text)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

// jsonText returns the text of a JSON string; ok is false for null or empty string.
func jsonText(data []byte) (string, bool) {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return "", false
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return strings.Trim(string(data), `"`), true
	}
	text = strings.TrimSpace(text)
	return text, text != ""
}

// parseTime tries all known ASL time layouts.
func parseTime(text string) (time.Time, error) {
	var lastErr error
	for _, layout := range dateTimeLayouts {
		parsed, err := time.Parse(layout, text)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

// LocalizedText is a text in many languages: key is language code ("ru", "uz", "en").
type LocalizedText map[string]string

// Get returns text for the language, or the first not empty text when it is missing.
func (l LocalizedText) Get(lang string) string {
	if text := l[lang]; text != "" {
		return text
	}
	for _, key := range []string{"ru", "uz", "en"} {
		if text := l[key]; text != "" {
			return text
		}
	}
	for _, text := range l {
		if text != "" {
			return text
		}
	}
	return ""
}

// DocumentEnvelope is the body of every signed document method (base64 JSON + signature).
type DocumentEnvelope struct {
	DocumentBody string `json:"documentBody"`        // base64 of JSON with A-Z sorted keys
	Signature    string `json:"signature,omitempty"` // detached or attached CMS signature
}

// DocumentIDResponse is the answer of methods that create a document.
type DocumentIDResponse struct {
	DocumentID string `json:"documentId"`
}

// FlexString is a JSON value that may come as string or number; it is kept as text.
type FlexString string

// UnmarshalJSON accepts string, number, bool or null.
func (f *FlexString) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*f = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		*f = FlexString(text)
		return nil
	}
	*f = FlexString(trimmed)
	return nil
}

// String returns the value as text.
func (f FlexString) String() string {
	return string(f)
}

// ErrorItem is one error object from an ASL error answer.
type ErrorItem struct {
	// Symbolic code, e.g. "invalid-token"
	Code    string `json:"code,omitempty"`
	ErrorID string `json:"errorId,omitempty"` // ID for searching in ASL logs
	Service string `json:"service,omitempty"` // ASL service that failed
	// Extra info: parameter, value, description
	Context              map[string]any `json:"context,omitempty"`
	ErrorCode            FlexString     `json:"errorCode,omitempty"` // numeric code, e.g. 602
	Error                string         `json:"error,omitempty"`     // text of the error
	JSONPath             string         `json:"jsonPath,omitempty"`  // wrong field in the request
	RequestBodyJSONPath  string         `json:"requestBodyJsonPath,omitempty"`
	RequestQueryJSONPath string         `json:"requestQueryJsonPath,omitempty"`
	RequestPathJSONPath  string         `json:"requestPathJsonPath,omitempty"`
}

// ContextText returns a text value from Context, or "" when it is missing.
func (e ErrorItem) ContextText(key string) string {
	value, ok := e.Context[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

// ErrorBody is an ASL error answer in object form; GlobalErrors is used by order methods.
type ErrorBody struct {
	ErrorItem
	GlobalErrors []ErrorItem `json:"globalErrors,omitempty"`
}
