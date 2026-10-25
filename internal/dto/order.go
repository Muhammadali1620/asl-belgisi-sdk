package dto

import "strings"

// CreateOrderRequest is a new order for marking codes (max 10 products in one order).
type CreateOrderRequest struct {
	ProductGroup ProductGroup `json:"productGroup"`
	// Empty only for service provider orders
	BusinessPlaceID   int               `json:"businessPlaceId,omitempty"`
	ReleaseMethodType ReleaseMethodType `json:"releaseMethodType"`
	Products          []OrderProduct    `json:"products"`
	IsPaid            *bool             `json:"isPaid,omitempty"`   // false means "free service only"
	PoNumber          string            `json:"poNumber,omitempty"` // our production order number
	ContractorInfo    *ContractorInfo   `json:"contractorInfo,omitempty"`
}

// OrderProduct is one sub-order: codes for one GTIN (max 150 000 codes).
type OrderProduct struct {
	GTIN             string           `json:"gtin"`
	Quantity         int              `json:"quantity"`
	CisType          PackageType      `json:"cisType"`
	SerialNumberType SerialNumberType `json:"serialNumberType"`
	SerialNumbers    []string         `json:"serialNumbers,omitempty"` // only for SELF_MADE
}

// ContractorInfo is the service provider who will make the order.
type ContractorInfo struct {
	ContractorTin         string      `json:"contractorTin"`
	ContractorCountryCode CountryCode `json:"contractorCountryCode"`
}

// CreateOrderResponse is the ID of the new order.
type CreateOrderResponse struct {
	OrderID string `json:"orderId"`
}

// OrderFilter is the search filter for orders; all fields are optional.
type OrderFilter struct {
	OrderID       string
	Status        OrderStatus
	ProductGroup  ProductGroup
	ContractorTin string
	PoNumber      string
	DateFrom      *DateTime
	DateTo        *DateTime
	Limit         int    // page size, ASL default is 100
	Cursor        string // ID of the last order from the previous page
}

// OrderInfo is a short description of an order.
type OrderInfo struct {
	OrderID           string            `json:"orderId"`
	ProductGroup      ProductGroup      `json:"productGroup"`
	OrderStatus       OrderStatus       `json:"orderStatus"`
	ReleaseMethodType ReleaseMethodType `json:"releaseMethodType"`
	PoNumber          string            `json:"poNumber,omitempty"`
	CreateDate        DateTime          `json:"createDate"`
	RejectionReason   string            `json:"rejectionReason,omitempty"`
}

// OrderList is a page of orders.
type OrderList struct {
	OrderInfos []OrderInfo `json:"orderInfos"`
}

// SubOrderFilter is the search filter for sub-orders; all fields are optional.
type SubOrderFilter struct {
	Status   BufferStatus
	OrderID  string
	GTIN     string
	CisType  PackageType
	DateFrom *DateTime
	DateTo   *DateTime
	Limit    int    // page size, ASL default is 100
	Cursor   string // ID of the last record from the previous page
}

// SubOrderInfo is the state of codes for one GTIN inside an order.
type SubOrderInfo struct {
	ParentOrderID   string       `json:"parentOrderId"`
	GTIN            string       `json:"gtin"`
	BufferStatus    BufferStatus `json:"bufferStatus"`
	CisType         PackageType  `json:"cisType"`
	AvailableCodes  int          `json:"availableCodes"` // codes made by ASL
	LeftInBuffer    int          `json:"leftInBuffer"`   // codes not downloaded yet
	TotalPassed     int          `json:"totalPassed"`    // codes already downloaded
	LastPackID      string       `json:"lastPackId,omitempty"`
	CreateDate      DateTime     `json:"createDate"`
	RejectionReason string       `json:"rejectionReason,omitempty"`
}

// RejectionCode returns the number at the start of RejectionReason ("321: ..." gives "321").
func (s SubOrderInfo) RejectionCode() string {
	code, _, found := strings.Cut(s.RejectionReason, ":")
	if !found {
		return ""
	}
	return strings.TrimSpace(code)
}

// SubOrderList is a page of sub-orders.
type SubOrderList struct {
	SubOrderInfos []SubOrderInfo `json:"subOrderInfos"`
}

// GetCodesRequest asks for the next pack of codes from a sub-order.
type GetCodesRequest struct {
	OrderID    string
	GTIN       string
	Quantity   int    // how many codes to download in this pack
	LastPackID string // empty for the first pack; ID of the last pack for the next one
}

// CodesPack is a downloaded pack of full marking codes (with GS separators and crypto tail).
type CodesPack struct {
	PackID string   `json:"packId"`
	Codes  []string `json:"codes"`
}

// CodePackList is the list of packs downloaded from a sub-order.
type CodePackList struct {
	OrderID string         `json:"orderId"`
	GTIN    string         `json:"gtin"`
	Packs   []CodePackInfo `json:"packs"`
}

// CodePackInfo is one downloaded pack.
type CodePackInfo struct {
	PackID       string   `json:"packId"`
	PackDateTime DateTime `json:"packDateTime"`
	Quantity     int      `json:"quantity"`
}

// CloseOrderResponse is the closed order (and GTIN when one sub-order was closed).
type CloseOrderResponse struct {
	OrderID string `json:"orderId"`
	GTIN    string `json:"gtin,omitempty"`
}

// UtilisationStatus is the status of a utilisation report (old method, removed from doc 1.26).
type UtilisationStatus struct {
	ReportID         string       `json:"reportId"`
	ReportStatus     ReportStatus `json:"reportStatus"`
	CreatedTimestamp DateTime     `json:"createdTimestamp"`
}
