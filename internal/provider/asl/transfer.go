package asl

import (
	"context"

	"asl/internal/dto"
)

// Transfer paths (pharma and medicals only).
const (
	pathTransferSearch  = "/public/api/v1/doc/transfer/request/search"
	pathTransferRequest = "/public/api/v1/doc/transfer/request"
	pathTransferConfirm = "/public/api/v1/doc/transfer/confirmation"
)

func (c *Client) SearchTransferRequests(
	ctx context.Context, filter dto.TransferSearchFilter,
) (*dto.TransferList, error) {
	// GET /public/api/v1/doc/transfer/request/search. Works only for pharma and medicals,
	// for imported goods before they are on the market.
	if err := checkDateRange("date", filter.DateFrom, filter.DateTo); err != nil {
		return nil, err
	}
	err := checkDateRange("businessDate", filter.BusinessDateFrom, filter.BusinessDateTo)
	if err != nil {
		return nil, err
	}
	values := query{}.
		set("documentId", filter.DocumentID).
		setInt("ownerId", filter.OwnerID).
		setInt("buyerId", filter.BuyerID).
		setTime("dateFrom", filter.DateFrom).
		setTime("dateTo", filter.DateTo).
		setTime("businessDateFrom", filter.BusinessDateFrom).
		setTime("businessDateTo", filter.BusinessDateTo).
		setInt("limit", filter.Limit).
		set("cursor", filter.Cursor)
	values = setList(values, "statuses", filter.Statuses)
	values = setList(values, "confirmationStatuses", filter.ConfirmationStatuses)
	var out dto.TransferList
	if err := c.get(ctx, pathTransferSearch, values, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SendTransferRequest(
	ctx context.Context, doc dto.TransferRequestDocument,
) (*dto.DocumentIDResponse, error) {
	// POST /public/api/v1/doc/transfer/request with base64 document (max 30 000 codes).
	if doc.BuyerID <= 0 || doc.OwnerID <= 0 {
		return nil, invalidArgument("buyerId", "buyerId and ownerId must be positive")
	}
	if doc.BusinessDatetime.IsZero() {
		return nil, invalidArgument("businessDatetime", "businessDatetime is required")
	}
	if len(doc.Products) == 0 {
		return nil, invalidArgument("products", "products must have at least one item")
	}
	for index, product := range doc.Products {
		if err := requireGTIN("products.gtin", product.GTIN); err != nil {
			return nil, err
		}
		if product.Quantity < 1 {
			return nil, invalidArgument("products.quantity",
				"products[%d].quantity must be positive", index)
		}
	}
	if err := requireCodes("codes", doc.Codes, MaxReportCodes); err != nil {
		return nil, err
	}
	return c.sendDocument(ctx, pathTransferRequest, doc)
}

func (c *Client) SendTransferConfirmation(
	ctx context.Context, doc dto.TransferConfirmationDocument,
) (*dto.DocumentIDResponse, error) {
	// POST /public/api/v1/doc/transfer/confirmation. The answer ID is a new document ID,
	// it is not the ID of the transfer request.
	if err := requireText("transferRequestDocId", doc.TransferRequestDocID); err != nil {
		return nil, err
	}
	if doc.Resolution != dto.ResolutionConfirmed && doc.Resolution != dto.ResolutionRejected {
		return nil, invalidArgument("resolution", "resolution must be CONFIRMED or REJECTED")
	}
	return c.sendDocument(ctx, pathTransferConfirm, doc)
}
