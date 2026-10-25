package dto

// TransferSearchFilter is the search filter for code transfer requests (pharma and medicals only).
type TransferSearchFilter struct {
	DocumentID           string
	Statuses             []DocumentStatus
	ConfirmationStatuses []ConfirmationStatus
	OwnerID              int // supplier participant ID
	BuyerID              int // buyer participant ID
	DateFrom             *DateTime
	DateTo               *DateTime
	BusinessDateFrom     *DateTime
	BusinessDateTo       *DateTime
	Limit                int    // ASL default is 10
	Cursor               string // UUID of the last document from the previous page
}

// TransferInfo is a short description of a transfer request.
type TransferInfo struct {
	DocumentID         string             `json:"documentId"`
	Status             DocumentStatus     `json:"status"`
	ConfirmationStatus ConfirmationStatus `json:"confirmationStatus,omitempty"`
	CreateDate         DateTime           `json:"createDate"`
}

// TransferList is a page of transfer requests.
type TransferList struct {
	DocumentInfos []TransferInfo `json:"documentInfos"`
}

// TransferRequestDocument asks a supplier to give codes to the buyer (max 30 000 codes).
type TransferRequestDocument struct {
	BuyerID                   int               `json:"buyerId"`
	OwnerID                   int               `json:"ownerId"`
	ExternalDocumentReference string            `json:"externalDocumentReference,omitempty"`
	BusinessDatetime          DateTime          `json:"businessDatetime"`
	Products                  []TransferProduct `json:"products"`
	Codes                     []string          `json:"codes"`
}

// TransferProduct is a GTIN and its count in consumer packages.
type TransferProduct struct {
	GTIN     string `json:"gtin"`
	Quantity int    `json:"quantity"`
}

// TransferConfirmationDocument is the answer to a transfer request.
type TransferConfirmationDocument struct {
	TransferRequestDocID string     `json:"transferRequestDocId"`
	Resolution           Resolution `json:"resolution"`
}
