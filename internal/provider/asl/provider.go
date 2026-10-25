// Package asl is a client (provider) for ASL BELGISI OPEN API v1.45 — the Uzbek marking system.
package asl

import (
	"context"
	"encoding/json"
	"time"

	"asl/internal/dto"
)

// Provider is the full ASL BELGISI API. Every method returns *Error on failure.
type Provider interface {
	// SetToken changes the bearer token (API key or technical user access token).
	SetToken(token string)

	// CheckAPIKey checks that the current API key belongs to the TIN and shows its expire date.
	CheckAPIKey(ctx context.Context, tin string) (*dto.APIKeyCheckResponse, error)
	// RefreshAPIKey replaces an API key with a new one; the old key stops working.
	RefreshAPIKey(
		ctx context.Context, tin string, req dto.APIKeyRefreshRequest,
	) (*dto.APIKeyRefreshResponse, error)
	// AuthenticateTechUser logs in a technical user and returns access and refresh tokens.
	AuthenticateTechUser(ctx context.Context, req dto.TechAuthRequest) (*dto.TokenResponse, error)
	// RefreshTechToken gets new technical user tokens by the refresh token.
	RefreshTechToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error)

	// CreateOrder registers an order for marking codes and returns the order ID.
	CreateOrder(ctx context.Context, req dto.CreateOrderRequest) (*dto.CreateOrderResponse, error)
	// ListOrders returns orders by filter.
	ListOrders(ctx context.Context, filter dto.OrderFilter) (*dto.OrderList, error)
	// GetOrder returns one order by ID.
	GetOrder(ctx context.Context, orderID string) (*dto.OrderInfo, error)
	// ListSubOrders returns sub-orders (code buffers per GTIN) by filter.
	ListSubOrders(ctx context.Context, filter dto.SubOrderFilter) (*dto.SubOrderList, error)
	// GetSubOrder returns the sub-order of one GTIN inside an order.
	GetSubOrder(ctx context.Context, orderID, gtin string) (*dto.SubOrderInfo, error)
	// WaitSubOrder polls a sub-order until it is not PENDING; REJECTED also returns *Error.
	WaitSubOrder(
		ctx context.Context, orderID, gtin string, interval time.Duration,
	) (*dto.SubOrderInfo, error)
	// GetCodes downloads the next pack of codes from a sub-order.
	GetCodes(ctx context.Context, req dto.GetCodesRequest) (*dto.CodesPack, error)
	// ListCodePacks returns all packs downloaded from a sub-order.
	ListCodePacks(ctx context.Context, orderID, gtin string) (*dto.CodePackList, error)
	// CloseOrder closes the whole order, or one sub-order when gtin is not empty.
	CloseOrder(ctx context.Context, orderID, gtin string) (*dto.CloseOrderResponse, error)
	// GetUtilisationStatus returns a utilisation report status (old method, use GetDocument).
	GetUtilisationStatus(ctx context.Context, reportID string) (*dto.UtilisationStatus, error)

	// SendUtilisation reports that codes are printed on goods.
	SendUtilisation(ctx context.Context, req dto.UtilisationRequest) (*dto.UtilisationResponse, error)
	// SendValidation reports print quality of codes.
	SendValidation(ctx context.Context, doc dto.ValidationDocument) (*dto.DocumentIDResponse, error)
	// SendAggregation reports that codes are packed into group or transport packages.
	SendAggregation(ctx context.Context, doc dto.AggregationDocument) (*dto.DocumentIDResponse, error)
	// SendCorrection changes dates, series or country of codes.
	SendCorrection(ctx context.Context, doc dto.CorrectionDocument) (*dto.DocumentIDResponse, error)
	// SendDisaggregation breaks group or transport packages.
	SendDisaggregation(
		ctx context.Context, doc dto.DisaggregationDocument,
	) (*dto.DocumentIDResponse, error)

	// SearchTransferRequests returns code transfer requests (pharma and medicals only).
	SearchTransferRequests(
		ctx context.Context, filter dto.TransferSearchFilter,
	) (*dto.TransferList, error)
	// SendTransferRequest asks a supplier to transfer codes to us.
	SendTransferRequest(
		ctx context.Context, doc dto.TransferRequestDocument,
	) (*dto.DocumentIDResponse, error)
	// SendTransferConfirmation confirms or rejects a code transfer request.
	SendTransferConfirmation(
		ctx context.Context, doc dto.TransferConfirmationDocument,
	) (*dto.DocumentIDResponse, error)

	// GenerateCustomsCode makes a new empty aggregated customs code (AIK).
	GenerateCustomsCode(ctx context.Context) (*dto.CustomsCodeResponse, error)
	// RegisterCustomsCode fills an AIK with codes and product data.
	RegisterCustomsCode(
		ctx context.Context, doc dto.CustomsCodeDocument,
	) (*dto.DocumentIDResponse, error)
	// WithdrawCustomsCode breaks a registered AIK.
	WithdrawCustomsCode(
		ctx context.Context, doc dto.CustomsCodeWithdrawalDocument,
	) (*dto.DocumentIDResponse, error)

	// CreateExport starts making a file with our codes.
	CreateExport(ctx context.Context, req dto.ExportRequest) (*dto.ExportResponse, error)
	// GetExportStatus returns the export task status.
	GetExportStatus(ctx context.Context, exportID string) (dto.ExportStatus, error)
	// WaitExport polls the export task until it is finished.
	WaitExport(ctx context.Context, exportID string, interval time.Duration) (dto.ExportStatus, error)
	// DownloadExport downloads the export result as zip bytes.
	DownloadExport(ctx context.Context, exportID string) ([]byte, error)

	// GetPublicCodes returns public info about codes.
	GetPublicCodes(ctx context.Context, req dto.PublicCodesRequest) ([]dto.PublicCodeInfo, error)
	// GetPrivateCodes returns full info about codes we have access to.
	GetPrivateCodes(ctx context.Context, req dto.PrivateCodesRequest) (*dto.PrivateCodesResponse, error)
	// CheckCodeOwner checks the code owner and returns child codes of packages.
	CheckCodeOwner(ctx context.Context, req dto.OwnerCheckRequest) (*dto.OwnerCheckResponse, error)
	// VerifyCodes checks that full codes (with check code) are real.
	VerifyCodes(ctx context.Context, codes []string) ([]dto.VerifyResult, error)

	// ListProducts returns product cards by filter.
	ListProducts(ctx context.Context, filter dto.ProductFilter) ([]dto.ProductSummary, error)
	// SearchProductsByGTIN returns product cards with the GTIN.
	SearchProductsByGTIN(
		ctx context.Context, filter dto.SearchByGTINFilter,
	) (*dto.SearchByGTINResponse, error)
	// GetProduct returns the full product card by its ID.
	GetProduct(ctx context.Context, productID string) (*dto.ProductCard, error)
	// GetProductByGTIN returns the full product card by GTIN.
	GetProductByGTIN(ctx context.Context, gtin string) (*dto.ProductCard, error)
	// GetProductFile downloads a file (photo or document) of a product card.
	GetProductFile(ctx context.Context, fileID string) ([]byte, error)

	// SearchDocuments returns documents by filter.
	SearchDocuments(ctx context.Context, filter dto.DocumentFilter) (*dto.DocumentList, error)
	// GetDocument returns the main data and status of a document.
	GetDocument(ctx context.Context, documentID string) (*dto.DocumentCard, error)
	// WaitDocument polls a document until ASL finishes processing it.
	WaitDocument(
		ctx context.Context, documentID string, interval time.Duration,
	) (*dto.DocumentCard, error)
	// WaitDocumentResult polls a document; ERROR or PARTIALLY_PROCESSED also returns *Error.
	WaitDocumentResult(
		ctx context.Context, documentID string, interval time.Duration,
	) (*dto.DocumentCard, error)
	// GetDocumentJSON returns the document body as it was sent.
	GetDocumentJSON(ctx context.Context, documentID string) (json.RawMessage, error)
	// GetDocumentErrors returns errors found in a document.
	GetDocumentErrors(
		ctx context.Context, documentID string, filter dto.DocumentErrorsFilter,
	) (*dto.DocumentErrorList, error)
	// GetDocumentCodes returns the processing result of each code in a document.
	GetDocumentCodes(
		ctx context.Context, documentID string, filter dto.DocumentCodesFilter,
	) ([]dto.DocumentCodeState, error)

	// GetPartyStatus returns info about a participant by TIN.
	GetPartyStatus(ctx context.Context, tin string) (*dto.PartyStatus, error)
}

// Client must implement the whole Provider.
var _ Provider = (*Client)(nil)
