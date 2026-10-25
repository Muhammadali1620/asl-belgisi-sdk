package test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// TestUnitErrorMapping sends real ASL error bodies (taken from stage) and checks our error fields.
func TestUnitErrorMapping(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		code      asl.ErrorCode
		aslCode   string
		parameter string
		message   string // part of the message
	}{
		{"array invalid token", 401,
			`[{"code":"invalid-token","service":"sec-party","errorId":"kdoprRBfXf","context":{}}]`,
			asl.CodeUnauthorized, "invalid-token", "", "invalid-token"},
		{"array entity not found", 404,
			`[{"code":"entity-not-found","errorId":"1","service":"cod-reader","context":` +
				`{"description":"CodeExportNotFoundException","specificError":"Unknown error"}}]`,
			asl.CodeNotFound, "entity-not-found", "", "CodeExportNotFoundException"},
		{"entity not found with 400", 400,
			`[{"code":"entity-not-found","service":"cod-reader",` +
				`"context":{"description":"ProductNotFoundException"}}]`,
			asl.CodeNotFound, "entity-not-found", "", "ProductNotFoundException"},
		{"object bad format", 400,
			`{"code":"invalid-input-parameter","errorId":"8","service":"doc-storage-gateway",` +
				`"context":{"_cause":"bad-format","value":"abc","parameter":"documentId"},` +
				`"requestBodyJsonPath":null}`,
			asl.CodeInvalidInput, "invalid-input-parameter", "documentId", "bad-format"},
		{"global errors 603", 400,
			`{"globalErrors":[{"error":"Invalid format","errorCode":603,"jsonPath":"dateFrom"}]}`,
			asl.CodeInvalidInput, "603", "dateFrom", "Invalid format"},
		{"global errors 602 with param in text", 400,
			`{"globalErrors":[{"error":"Incorrect value [999999] of input parameter [businessPlaceId]",` +
				`"errorCode":602}]}`,
			asl.CodeInvalidInput, "602", "businessPlaceId", "Incorrect value"},
		{"order not found 306 with junk jsonPath", 400,
			`{"globalErrors":[{"error":"Order 1 not found","errorCode":306,"jsonPath":"Order 1 not found"}]}`,
			asl.CodeNotFound, "306", "", "Order 1 not found"},
		{"report not found 725", 400,
			`{"globalErrors":[{"error":"Report ID 1 not found","errorCode":725}]}`,
			asl.CodeNotFound, "725", "", "Report ID"},
		{"close in wrong status 307", 400,
			`{"globalErrors":[{"error":"Closing an order is allowed only in the READY or ` +
				`OUTSOURCED statuses","errorCode":307}]}`,
			asl.CodeConflict, "307", "", "Closing an order"},
		{"wrong login 100", 400,
			`{"globalErrors":[{"error":"Incorrect login or password","errorCode":100}]}`,
			asl.CodeUnauthorized, "100", "", "Incorrect login"},
		{"401 status wins over body 201", 401,
			`{"globalErrors":[{"error":"No authorization","errorCode":201}]}`,
			asl.CodeUnauthorized, "201", "", "No authorization"},
		{"no rights 201 with 403", 403,
			`{"globalErrors":[{"error":"No rights","errorCode":201}]}`,
			asl.CodeForbidden, "201", "", "No rights"},
		{"referenced document is missing", 422,
			`{"code":"invalid-document-reference","service":"doc-transfer","context":` +
				`{"_cause":"missed-param","value":"0000","parameter":"transferRequestDocId"}}`,
			asl.CodeNotFound, "invalid-document-reference", "transferRequestDocId", "missed-param"},
		{"missing signature", 400,
			`{"code":"missing-request-parameter","context":` +
				`{"_cause":"missed-param","value":"signature","parameter":"signature"}}`,
			asl.CodeInvalidInput, "missing-request-parameter", "signature", "missed-param"},
		{"invalid signature", 400,
			`{"code":"invalid-signature","context":{"_cause":"invalid-signature"}}`,
			asl.CodeSignature, "invalid-signature", "", "invalid-signature"},
		{"java class in parameter", 400,
			`{"code":"invalid-input-parameter","context":{"_message":"Type mismatch.",` +
				`"parameter":"org.springframework.web.server.ServerWebInputException"}}`,
			asl.CodeInvalidInput, "invalid-input-parameter", "", "Type mismatch"},
		{"service down code 1", 500,
			`{"globalErrors":[{"error":"The service is temporarily unavailable","errorCode":1}]}`,
			asl.CodeUnavailable, "1", "", "temporarily unavailable"},
		{"html bad gateway", 502, `<html><body>Bad Gateway</body></html>`,
			asl.CodeUnavailable, "", "", "Bad Gateway"},
		{"plain text server error", 500, `internal problem`,
			asl.CodeServerError, "", "", "internal problem"},
		{"empty 404", 404, ``, asl.CodeNotFound, "", "", "Not Found"},
		{"conflict", 409, `{}`, asl.CodeConflict, "", "", "Conflict"},
		{"gone", 410, ``, asl.CodeGone, "", "", "Gone"},
		{"too large", 413, ``, asl.CodePayloadTooLarge, "", "", "Too Large"},
		{"media", 415, ``, asl.CodeUnsupportedMedia, "", "", "Unsupported"},
		{"unprocessable", 422, `{}`, asl.CodeUnprocessable, "", "", "Unprocessable"},
		{"rate limit", 429, `{"code":"invalid-input-parameter"}`,
			asl.CodeRateLimited, "invalid-input-parameter", "", ""},
		{"method not allowed", 405, ``, asl.CodeMethodNotAllowed, "", "", "Method Not Allowed"},
		{"client closed", 499, ``, asl.CodeCanceled, "", "", ""},
		{"unknown status", 418, ``, asl.CodeUnknown, "", "", "teapot"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			mock := newMock(t).on(http.MethodPost, "/public/api/cod/exports",
				reply{Status: item.status, Body: item.body})
			client := mock.client(t, func(c *asl.Config) { c.MaxRetries = -1 })
			_, err := client.CreateExport(t.Context(), dto.ExportRequest{GTIN: "04605246003585"})
			aslErr := requireCode(t, err, item.code)
			if aslErr.HTTPStatus != item.status {
				t.Errorf("HTTPStatus = %d, want %d", aslErr.HTTPStatus, item.status)
			}
			if aslErr.ASLCode != item.aslCode {
				t.Errorf("ASLCode = %q, want %q", aslErr.ASLCode, item.aslCode)
			}
			if aslErr.Parameter != item.parameter {
				t.Errorf("Parameter = %q, want %q", aslErr.Parameter, item.parameter)
			}
			if !strings.Contains(aslErr.Message, item.message) {
				t.Errorf("Message = %q, want to contain %q", aslErr.Message, item.message)
			}
			if aslErr.Method != http.MethodPost || aslErr.Path != "/public/api/cod/exports" {
				t.Errorf("wrong method or path: %s %s", aslErr.Method, aslErr.Path)
			}
			if !strings.Contains(err.Error(), string(item.code)) {
				t.Errorf("Error() must contain the code: %q", err.Error())
			}
		})
	}
}

func TestUnitErrorHelpers(t *testing.T) {
	notFound := &asl.Error{Code: asl.CodeNotFound, Message: "x"}
	wrapped := errors.Join(errors.New("context"), notFound)
	if !errors.Is(wrapped, asl.ErrNotFound) || errors.Is(wrapped, asl.ErrForbidden) {
		t.Error("errors.Is must compare only Code")
	}
	if got, ok := asl.AsError(wrapped); !ok || got != notFound {
		t.Error("AsError must find wrapped error")
	}
	if _, ok := asl.AsError(errors.New("plain")); ok {
		t.Error("AsError must be false for other errors")
	}
	if !asl.HasCode(wrapped, asl.CodeForbidden, asl.CodeNotFound) || asl.HasCode(nil, asl.CodeNotFound) {
		t.Error("HasCode is wrong")
	}
	retryable := []asl.ErrorCode{asl.CodeRateLimited, asl.CodeUnavailable, asl.CodeTimeout, asl.CodeNetwork}
	for _, code := range retryable {
		if !asl.IsRetryable(&asl.Error{Code: code}) {
			t.Errorf("%s must be retryable", code)
		}
	}
	if asl.IsRetryable(notFound) || asl.IsRetryable(context.Canceled) {
		t.Error("NOT_FOUND and plain errors are not retryable")
	}
	cause := errors.New("low level")
	if !errors.Is(&asl.Error{Code: asl.CodeNetwork, Err: cause}, cause) {
		t.Error("Unwrap must return the cause")
	}
	documentCodes := map[string]asl.ErrorCode{
		"code-not-found":          asl.CodeNotFound,
		"CODE-NOT-FOUND":          asl.CodeNotFound,
		"parent-not-found":        asl.CodeNotFound,
		"invalid-code-status":     asl.CodeConflict,
		"invalid-code-owner":      asl.CodeForbidden,
		"invalid-nested-codes":    asl.CodeInvalidInput,
		"invalid-code-properties": asl.CodeInvalidInput,
		"something-new":           asl.CodeUnprocessable,
	}
	for aslCode, want := range documentCodes {
		if got := asl.DocumentErrorCode(aslCode); got != want {
			t.Errorf("DocumentErrorCode(%q) = %s, want %s", aslCode, got, want)
		}
	}
}

// TestUnitDocumentResultError checks that the most important document error gives the code.
func TestUnitDocumentResultError(t *testing.T) {
	const docID = "145dac06-3182-45a2-4c02-10d66a7774e9"
	mock := newMock(t).
		on(http.MethodGet, "/public/api/v1/doc/storage/docs/"+docID, reply{Body: `{"documentId":"` + docID +
			`","type":"AGGREGATION","status":"ERROR","createDate":"2026-10-25T03:29:34.524Z"}`}).
		on(http.MethodGet, "/public/api/v1/doc/storage/errors/"+docID, reply{Body: `{"documentErrors":[
			{"propertyName":"CODE","index":0,"errorCode":"invalid-nested-codes",
				"errorTags":{"nested_codes":"1"}},
			{"propertyName":"CODE","index":1,"errorCode":"code-not-found",
				"errorTags":{"_service":"doc-aggregation"}}]}`})
	client := mock.client(t)
	card, err := client.WaitDocumentResult(t.Context(), docID, 0)
	aslErr := requireCode(t, err, asl.CodeNotFound)
	if card == nil || card.Status != dto.DocumentStatusError {
		t.Fatalf("card must be returned with the error, got %+v", card)
	}
	if aslErr.ASLCode != "code-not-found" || len(aslErr.Details) != 2 || aslErr.HTTPStatus != 200 {
		t.Errorf("wrong document error: %+v", aslErr)
	}
	if aslErr.Details[0].ContextText("nested_codes") != "1" {
		t.Errorf("error tags must be kept in Details: %+v", aslErr.Details[0])
	}
	if limit := mock.last(t).Query.Get("limit"); limit != "100" {
		t.Errorf("errors limit must default to 100, got %q", limit)
	}
}
