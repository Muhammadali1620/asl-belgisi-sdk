package asl

import (
	"context"
	"net/http"
	"strings"
	"time"

	"asl/internal/dto"
)

// Order paths (ICOM).
const (
	pathOrders      = "/api/orders"
	pathSubOrders   = "/api/orders/sub-orders"
	pathCodes       = "/api/codes"
	pathCodePacks   = "/api/codes/packs"
	pathOrderClose  = "/api/order/close"
	pathUtilisation = "/api/utilisation"
)

// defaultPollInterval is used by Wait methods when interval is not positive.
const defaultPollInterval = 2 * time.Second

func (c *Client) CreateOrder(
	ctx context.Context, req dto.CreateOrderRequest,
) (*dto.CreateOrderResponse, error) {
	// POST /api/orders. Max 10 products, max 150 000 codes per product, max 100 active orders.
	// Close the order after all codes are downloaded, or ASL closes it in 7 days.
	if err := validateOrder(req); err != nil {
		return nil, err
	}
	var out dto.CreateOrderResponse
	if err := c.post(ctx, pathOrders, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// validateOrder checks an order before it is sent.
func validateOrder(req dto.CreateOrderRequest) error {
	// Checks only rules written in the doc; ASL checks the rest (GTIN card, rights).
	if err := requireText("productGroup", string(req.ProductGroup)); err != nil {
		return err
	}
	if err := requireText("releaseMethodType", string(req.ReleaseMethodType)); err != nil {
		return err
	}
	if len(req.Products) == 0 || len(req.Products) > MaxOrderProducts {
		return invalidArgument("products", "order must have 1 to %d products, got %d",
			MaxOrderProducts, len(req.Products))
	}
	for index, product := range req.Products {
		if err := validateOrderProduct(index, product); err != nil {
			return err
		}
	}
	if info := req.ContractorInfo; info != nil {
		if err := requireTIN("contractorInfo.contractorTin", info.ContractorTin); err != nil {
			return err
		}
	}
	return nil
}

// validateOrderProduct checks one sub-order.
func validateOrderProduct(index int, product dto.OrderProduct) error {
	// Serial numbers are allowed only when we make them ourselves (SELF_MADE).
	if err := requireGTIN("products.gtin", product.GTIN); err != nil {
		return err
	}
	if product.Quantity < 1 || product.Quantity > MaxCodesPerProduct {
		return invalidArgument("products.quantity", "products[%d].quantity must be 1..%d, got %d",
			index, MaxCodesPerProduct, product.Quantity)
	}
	if err := requireText("products.cisType", string(product.CisType)); err != nil {
		return err
	}
	serialType := string(product.SerialNumberType)
	if err := requireText("products.serialNumberType", serialType); err != nil {
		return err
	}
	selfMade := product.SerialNumberType == dto.SerialNumberSelfMade
	if selfMade && len(product.SerialNumbers) == 0 {
		return invalidArgument("products.serialNumbers",
			"products[%d] is SELF_MADE but has no serial numbers", index)
	}
	if !selfMade && len(product.SerialNumbers) > 0 {
		return invalidArgument("products.serialNumbers",
			"products[%d] has serial numbers but is not SELF_MADE", index)
	}
	return nil
}

func (c *Client) ListOrders(ctx context.Context, filter dto.OrderFilter) (*dto.OrderList, error) {
	// GET /api/orders with filters. Use the last order ID as Cursor to get the next page.
	if err := checkDateRange("date", filter.DateFrom, filter.DateTo); err != nil {
		return nil, err
	}
	values := query{}.
		set("orderId", filter.OrderID).
		set("status", string(filter.Status)).
		set("productGroup", string(filter.ProductGroup)).
		set("contractorTin", filter.ContractorTin).
		set("poNumber", filter.PoNumber).
		setTime("dateFrom", filter.DateFrom).
		setTime("dateTo", filter.DateTo).
		setInt("limit", filter.Limit).
		set("cursor", filter.Cursor)
	var out dto.OrderList
	if err := c.get(ctx, pathOrders, values, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetOrder(ctx context.Context, orderID string) (*dto.OrderInfo, error) {
	// Uses ListOrders with orderId filter; returns NOT_FOUND when ASL gives no order.
	orderID = strings.TrimSpace(orderID)
	if err := requireText("orderId", orderID); err != nil {
		return nil, err
	}
	list, err := c.ListOrders(ctx, dto.OrderFilter{OrderID: orderID})
	if err != nil {
		return nil, err
	}
	for _, order := range list.OrderInfos {
		if strings.EqualFold(order.OrderID, orderID) {
			return &order, nil
		}
	}
	return nil, notFound("orderId", "order "+orderID+" is not found", pathOrders)
}

func (c *Client) ListSubOrders(
	ctx context.Context, filter dto.SubOrderFilter,
) (*dto.SubOrderList, error) {
	// GET /api/orders/sub-orders with filters. Shows how many codes are ready in each buffer.
	if err := checkDateRange("date", filter.DateFrom, filter.DateTo); err != nil {
		return nil, err
	}
	values := query{}.
		set("status", string(filter.Status)).
		set("orderId", filter.OrderID).
		set("gtin", filter.GTIN).
		set("cisType", string(filter.CisType)).
		setTime("dateFrom", filter.DateFrom).
		setTime("dateTo", filter.DateTo).
		setInt("limit", filter.Limit).
		set("cursor", filter.Cursor)
	var out dto.SubOrderList
	if err := c.get(ctx, pathSubOrders, values, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSubOrder(ctx context.Context, orderID, gtin string) (*dto.SubOrderInfo, error) {
	// Uses ListSubOrders with orderId and gtin; returns NOT_FOUND when ASL gives no sub-order.
	orderID = strings.TrimSpace(orderID)
	if err := requireText("orderId", orderID); err != nil {
		return nil, err
	}
	if err := requireGTIN("gtin", gtin); err != nil {
		return nil, err
	}
	list, err := c.ListSubOrders(ctx, dto.SubOrderFilter{OrderID: orderID, GTIN: gtin})
	if err != nil {
		return nil, err
	}
	for _, sub := range list.SubOrderInfos {
		if sub.GTIN == gtin && strings.EqualFold(sub.ParentOrderID, orderID) {
			return &sub, nil
		}
	}
	message := "sub-order of order " + orderID + " and gtin " + gtin + " is not found"
	return nil, notFound("orderId", message, pathSubOrders)
}

func (c *Client) WaitSubOrder(
	ctx context.Context, orderID, gtin string, interval time.Duration,
) (*dto.SubOrderInfo, error) {
	// Asks ASL every interval (default 2s) while the status is PENDING; stops when ctx ends.
	// REJECTED gives the sub-order and *Error (321 -> NOT_FOUND, 311 -> INVALID_INPUT).
	interval = valueOr(interval, defaultPollInterval)
	for {
		sub, err := c.GetSubOrder(ctx, orderID, gtin)
		if err != nil && !HasCode(err, CodeNotFound) {
			return nil, err
		}
		if err == nil && sub.BufferStatus == dto.BufferStatusRejected {
			return sub, rejectedError(sub)
		}
		if err == nil && sub.BufferStatus != dto.BufferStatusPending {
			return sub, nil
		}
		if waitErr := sleep(ctx, interval); waitErr != nil {
			return nil, transportError(ctx, http.MethodGet, pathSubOrders, waitErr)
		}
	}
}

// rejectedError builds the error for a rejected sub-order from its rejection reason.
func rejectedError(sub *dto.SubOrderInfo) *Error {
	// The reason looks like "321: The product code ... was not found".
	aslCode := sub.RejectionCode()
	item := dto.ErrorItem{ErrorCode: dto.FlexString(aslCode), Error: sub.RejectionReason}
	code, ok := itemCode(item)
	if !ok {
		code = CodeUnprocessable
	}
	return &Error{
		Code:       code,
		HTTPStatus: http.StatusOK,
		ASLCode:    aslCode,
		Message:    "sub-order is rejected: " + sub.RejectionReason,
		Parameter:  "gtin",
		Path:       pathSubOrders,
		Details:    []dto.ErrorItem{item},
	}
}

func (c *Client) GetCodes(ctx context.Context, req dto.GetCodesRequest) (*dto.CodesPack, error) {
	// GET /api/codes. Without LastPackID ASL gives the first pack again (not new codes).
	// For the next pack send the ID of the last received pack. Order must be READY.
	if err := requireText("orderId", req.OrderID); err != nil {
		return nil, err
	}
	if err := requireGTIN("gtin", req.GTIN); err != nil {
		return nil, err
	}
	if req.Quantity < 1 || req.Quantity > MaxCodesPerProduct {
		return nil, invalidArgument("quantity", "quantity must be 1..%d, got %d",
			MaxCodesPerProduct, req.Quantity)
	}
	values := query{}.
		set("orderId", strings.TrimSpace(req.OrderID)).
		set("gtin", req.GTIN).
		setInt("quantity", req.Quantity).
		set("lastPackId", strings.TrimSpace(req.LastPackID))
	var out dto.CodesPack
	if err := c.get(ctx, pathCodes, values, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListCodePacks(ctx context.Context, orderID, gtin string) (*dto.CodePackList, error) {
	// GET /api/codes/packs. Unknown order gives an empty list, not an error.
	if err := requireText("orderId", orderID); err != nil {
		return nil, err
	}
	if err := requireGTIN("gtin", gtin); err != nil {
		return nil, err
	}
	values := query{}.set("orderId", strings.TrimSpace(orderID)).set("gtin", gtin)
	var out dto.CodePackList
	if err := c.get(ctx, pathCodePacks, values, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CloseOrder(ctx context.Context, orderID, gtin string) (*dto.CloseOrderResponse, error) {
	// POST /api/order/close?orderId=...&gtin=... Not downloaded codes are cancelled.
	// Empty gtin closes the whole order; closing the last sub-order also closes the order.
	if err := requireText("orderId", orderID); err != nil {
		return nil, err
	}
	if gtin != "" {
		if err := requireGTIN("gtin", gtin); err != nil {
			return nil, err
		}
	}
	values := query{}.set("orderId", strings.TrimSpace(orderID)).set("gtin", gtin)
	var out dto.CloseOrderResponse
	if err := c.post(ctx, pathOrderClose, values, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetUtilisationStatus(
	ctx context.Context, reportID string,
) (*dto.UtilisationStatus, error) {
	// Deprecated: removed from doc 1.26 but still works. GET /api/utilisation/{reportId}.
	// Use GetDocument with the same ID: it gives more details.
	id, err := pathID("reportId", reportID)
	if err != nil {
		return nil, err
	}
	var out dto.UtilisationStatus
	if err := c.get(ctx, pathUtilisation+"/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
