package asl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"asl/internal/dto"
)

// Document storage paths.
const (
	pathDocSearch = "/public/api/v1/doc/storage/docs/search"
	pathDocs      = "/public/api/v1/doc/storage/docs/"
	pathDocJSON   = "/public/api/v1/doc/storage/json/"
	pathDocErrors = "/public/api/v1/doc/storage/errors/"
)

func (c *Client) SearchDocuments(
	ctx context.Context, filter dto.DocumentFilter,
) (*dto.DocumentList, error) {
	// GET /public/api/v1/doc/storage/docs/search. Use the last document ID as Cursor for next page.
	if err := checkDateRange("date", filter.DateFrom, filter.DateTo); err != nil {
		return nil, err
	}
	for _, docType := range filter.Types {
		if docType == dto.DocumentTypeSalesReceipt || docType == dto.DocumentTypeRefundReceipt {
			return nil, invalidArgument("types", "type %s is not allowed in search", docType)
		}
	}
	values := query{}.
		set("documentId", filter.DocumentID).
		setBool("withWarning", filter.WithWarning).
		setTime("dateFrom", filter.DateFrom).
		setTime("dateTo", filter.DateTo).
		setInt("limit", filter.Limit).
		set("cursor", filter.Cursor)
	values = setList(values, "productGroups", filter.ProductGroups)
	values = setList(values, "status", filter.Statuses)
	values = setList(values, "types", filter.Types)
	var out dto.DocumentList
	if err := c.get(ctx, pathDocSearch, values, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetDocument(ctx context.Context, documentID string) (*dto.DocumentCard, error) {
	// GET /public/api/v1/doc/storage/docs/{id}. Use it to see the status of any sent report.
	id, err := pathID("documentId", documentID)
	if err != nil {
		return nil, err
	}
	var out dto.DocumentCard
	if err := c.get(ctx, pathDocs+id, nil, &out); err != nil {
		return nil, notFoundOnMissedParam(err)
	}
	return &out, nil
}

func (c *Client) WaitDocument(
	ctx context.Context, documentID string, interval time.Duration,
) (*dto.DocumentCard, error) {
	// Asks the document every interval (default 2s) until SUCCESS, PARTIALLY_PROCESSED or ERROR.
	// A new document may be NOT_FOUND for a few seconds, so NOT_FOUND is waited too.
	interval = valueOr(interval, defaultPollInterval)
	for {
		card, err := c.GetDocument(ctx, documentID)
		if err != nil && !HasCode(err, CodeNotFound) {
			return nil, err
		}
		if err == nil && card.Status.IsFinal() {
			return card, nil
		}
		if waitErr := sleep(ctx, interval); waitErr != nil {
			return card, transportError(ctx, http.MethodGet, pathDocs, waitErr)
		}
	}
}

func (c *Client) WaitDocumentResult(
	ctx context.Context, documentID string, interval time.Duration,
) (*dto.DocumentCard, error) {
	// Like WaitDocument, but ERROR and PARTIALLY_PROCESSED give the card and *Error.
	// The most important document error gives the code: code-not-found -> NOT_FOUND and so on.
	card, err := c.WaitDocument(ctx, documentID, interval)
	if err != nil || card.Status == dto.DocumentStatusSuccess {
		return card, err
	}
	list, err := c.GetDocumentErrors(ctx, documentID, dto.DocumentErrorsFilter{})
	if err != nil {
		return card, err
	}
	return card, documentError(card, list.DocumentErrors)
}

// documentError builds the error for a document that ASL did not accept fully.
func documentError(card *dto.DocumentCard, docErrors []dto.DocumentError) *Error {
	// All document errors are kept in Details: Code is errorCode, Context is errorTags.
	// Priority: NOT_FOUND, FORBIDDEN, CONFLICT, INVALID_INPUT, UNPROCESSABLE.
	result := &Error{
		Code:       CodeUnprocessable,
		HTTPStatus: http.StatusOK,
		Message:    "document " + card.DocumentID + " is " + string(card.Status),
		Path:       pathDocs + card.DocumentID,
	}
	for _, docErr := range docErrors {
		tags := map[string]any{"index": docErr.Index, "propertyName": docErr.PropertyName}
		for key, value := range docErr.ErrorTags {
			tags[key] = value
		}
		result.Details = append(result.Details, dto.ErrorItem{Code: docErr.ErrorCode, Context: tags})
	}
	main := -1
	for index, docErr := range docErrors {
		code := DocumentErrorCode(docErr.ErrorCode)
		if main < 0 || documentErrorPriority[code] > documentErrorPriority[result.Code] {
			main, result.Code, result.ASLCode = index, code, docErr.ErrorCode
		}
	}
	if main >= 0 {
		result.Message += fmt.Sprintf(": %s at index %d (%d errors)", result.ASLCode,
			docErrors[main].Index, len(docErrors))
	}
	return result
}

func (c *Client) GetDocumentJSON(ctx context.Context, documentID string) (json.RawMessage, error) {
	// GET /public/api/v1/doc/storage/json/{id}. Returns the document body as it was sent.
	id, err := pathID("documentId", documentID)
	if err != nil {
		return nil, err
	}
	var out json.RawMessage
	if err := c.get(ctx, pathDocJSON+id, nil, &out); err != nil {
		return nil, notFoundOnMissedParam(err)
	}
	return out, nil
}

func (c *Client) GetDocumentErrors(
	ctx context.Context, documentID string, filter dto.DocumentErrorsFilter,
) (*dto.DocumentErrorList, error) {
	// GET /public/api/v1/doc/storage/errors/{id}. ASL needs limit, so 100 is sent when it is 0.
	id, err := pathID("documentId", documentID)
	if err != nil {
		return nil, err
	}
	if filter.Limit <= 0 {
		filter.Limit = DefaultDocErrorLimit
	}
	values := query{}.
		set("propertyName", filter.PropertyName).
		setInt("limit", filter.Limit).
		setIntPtr("lastIndex", filter.LastIndex)
	var out dto.DocumentErrorList
	if err := c.get(ctx, pathDocErrors+id, values, &out); err != nil {
		return nil, notFoundOnMissedParam(err)
	}
	return &out, nil
}

func (c *Client) GetDocumentCodes(
	ctx context.Context, documentID string, filter dto.DocumentCodesFilter,
) ([]dto.DocumentCodeState, error) {
	// GET /public/api/v1/doc/storage/docs/{id}/codes. Each code is SUCCESS, WARNING or ERROR;
	// Result has the reason for WARNING and ERROR (e.g. code-not-found).
	id, err := pathID("documentId", documentID)
	if err != nil {
		return nil, err
	}
	values := query{}.setInt("limit", filter.Limit).setIntPtr("lastIndex", filter.LastIndex)
	var out []dto.DocumentCodeState
	if err := c.get(ctx, pathDocs+id+"/codes", values, &out); err != nil {
		return nil, notFoundOnMissedParam(err)
	}
	return out, nil
}

func (c *Client) GetPartyStatus(ctx context.Context, tin string) (*dto.PartyStatus, error) {
	// GET /public/api/v1/party/parties/{tin}/status. Unknown TIN gives an empty answer:
	// then NOT_FOUND is returned. Limit: 10 000 calls per hour.
	if err := requireTIN("tin", tin); err != nil {
		return nil, err
	}
	var out dto.PartyStatus
	path := pathParties + strings.TrimSpace(tin) + "/status"
	if err := c.get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	if !out.Found() {
		return nil, notFound("tin", "participant "+tin+" is not found", path)
	}
	return &out, nil
}

// notFoundOnMissedParam turns the ASL "missed-param doc_id" answer into NOT_FOUND.
func notFoundOnMissedParam(err error) error {
	// Document storage says "invalid-input-parameter / missed-param doc_id" for unknown IDs.
	aslErr, ok := AsError(err)
	if !ok || aslErr.Code != CodeInvalidInput || len(aslErr.Details) == 0 {
		return err
	}
	first := aslErr.Details[0]
	if first.ContextText("_cause") == "missed-param" && first.ContextText("parameter") == "doc_id" {
		aslErr.Code = CodeNotFound
		aslErr.Message = "document is not found"
	}
	return err
}
