package dto

type ProductGroup string

// Product groups.
const (
	ProductGroupPharma       ProductGroup = "pharma"       // medicines
	ProductGroupMedicals     ProductGroup = "medicals"     // medical products
	ProductGroupWater        ProductGroup = "water"        // water and soft drinks
	ProductGroupVegetableOil ProductGroup = "vegetableoil" // food oil and fat products
	ProductGroupBio          ProductGroup = "bio"          // dietary supplements
	ProductGroupAntiseptic   ProductGroup = "antiseptic"   // non-food products with alcohol
	ProductGroupAppliances   ProductGroup = "appliances"   // home appliances
	ProductGroupFertilizers  ProductGroup = "fertilizers"  // mineral fertilizers and plant protection
)

// productGroupIDs maps product group alias to its numeric ASL ID.
var productGroupIDs = map[ProductGroup]int{
	ProductGroupPharma:       7,
	ProductGroupMedicals:     10,
	ProductGroupWater:        13,
	ProductGroupVegetableOil: 33,
	ProductGroupBio:          17,
	ProductGroupAntiseptic:   19,
	ProductGroupAppliances:   18,
	ProductGroupFertilizers:  53,
}

// ID returns the numeric ASL ID of the group, or 0 when the group is unknown.
func (g ProductGroup) ID() int {
	return productGroupIDs[g]
}

// ProductGroupByID returns the group alias for a numeric ASL ID; ok is false when unknown.
func ProductGroupByID(id int) (ProductGroup, bool) {
	for group, groupID := range productGroupIDs {
		if groupID == id {
			return group, true
		}
	}
	return "", false
}

// ReleaseMethodType is the marking goal of an order (doc 13.2).
type ReleaseMethodType string

// Marking goals.
const (
	ReleaseMethodPrimary   ReleaseMethodType = "PRIMARY"   // first marking (production or import)
	ReleaseMethodRemark    ReleaseMethodType = "REMARK"    // marking again
	ReleaseMethodRemains   ReleaseMethodType = "REMAINS"   // marking of goods in stock
	ReleaseMethodComission ReleaseMethodType = "COMISSION" // commission trade, used goods
)

// ReleaseType is how goods enter the market (doc 13.3 and 13.17).
type ReleaseType string

// Release types.
const (
	ReleaseProduction  ReleaseType = "PRODUCTION"  // made in Uzbekistan
	ReleaseImport      ReleaseType = "IMPORT"      // made in other country
	ReleaseCirculation ReleaseType = "CIRCULATION" // goods on market (not for utilisation)
	ReleaseProcessing  ReleaseType = "PROCESSING"  // processed on customs territory
)

// SerialNumberType is who makes serial numbers of codes (doc 13.4).
type SerialNumberType string

// Serial number types.
const (
	SerialNumberSelfMade SerialNumberType = "SELF_MADE" // we send serial numbers in the order
	SerialNumberOperator SerialNumberType = "OPERATOR"  // ASL makes serial numbers
)

// OrderStatus is the status of a code emission order (doc 13.5).
type OrderStatus string

// Order statuses.
const (
	OrderStatusCreated    OrderStatus = "CREATED"
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusReady      OrderStatus = "READY"
	OrderStatusRejected   OrderStatus = "REJECTED"
	OrderStatusClosed     OrderStatus = "CLOSED"
	OrderStatusOutsourced OrderStatus = "OUTSOURCED" // service provider got the codes
)

// BufferStatus is the status of a sub-order (one GTIN inside an order, doc 13.6).
type BufferStatus string

// Sub-order statuses.
const (
	BufferStatusPending   BufferStatus = "PENDING"   // codes are not ready yet
	BufferStatusActive    BufferStatus = "ACTIVE"    // codes are ready to download
	BufferStatusExhausted BufferStatus = "EXHAUSTED" // all codes are downloaded
	BufferStatusRejected  BufferStatus = "REJECTED"
	BufferStatusClosed    BufferStatus = "CLOSED"
)

// ReportStatus is the status of a report (doc 13.7).
type ReportStatus string

// Report statuses.
const (
	ReportStatusInProcess ReportStatus = "IN_PROCESS"
	ReportStatusSuccess   ReportStatus = "SUCCESS"
	ReportStatusError     ReportStatus = "ERROR"
)

// DocumentStatus is the status of a document (doc 13.8).
type DocumentStatus string

// Document statuses.
const (
	DocumentStatusCreated            DocumentStatus = "CREATED"
	DocumentStatusValidating         DocumentStatus = "VALIDATING"
	DocumentStatusInProcess          DocumentStatus = "IN_PROCESS"
	DocumentStatusPartiallyProcessed DocumentStatus = "PARTIALLY_PROCESSED" // some codes failed
	DocumentStatusSuccess            DocumentStatus = "SUCCESS"             // all codes are done
	DocumentStatusError              DocumentStatus = "ERROR"               // no code was done
)

// IsFinal reports whether ASL finished the document processing.
func (s DocumentStatus) IsFinal() bool {
	switch s {
	case DocumentStatusPartiallyProcessed, DocumentStatusSuccess, DocumentStatusError:
		return true
	default:
		return false
	}
}

// ReportType is the type of a report (doc 13.9).
type ReportType string

// Report types.
const (
	ReportTypeValidation  ReportType = "VALIDATION"
	ReportTypeUtilisation ReportType = "UTILISATION"
	ReportTypeAggregation ReportType = "AGGREGATION"
)

// ConfirmationStatus is the status of a code transfer request (doc 13.10).
type ConfirmationStatus string

// Confirmation statuses.
const (
	ConfirmationAwaiting  ConfirmationStatus = "AWAITING_CONFIRMATION"
	ConfirmationRevoked   ConfirmationStatus = "REVOKED" // not answered in time
	ConfirmationConfirmed ConfirmationStatus = "CONFIRMED"
	ConfirmationRejected  ConfirmationStatus = "REJECTED"
)

// Resolution is the answer to a code transfer request (doc 13.11).
type Resolution string

// Resolutions.
const (
	ResolutionConfirmed Resolution = "CONFIRMED"
	ResolutionRejected  Resolution = "REJECTED"
)

// DocumentType is the type of a document (doc 13.12).
type DocumentType string

// Document types.
const (
	DocumentTypeWithdrawal                  DocumentType = "WITHDRAWAL"
	DocumentTypeSalesReceipt                DocumentType = "SALES_RECEIPT"
	DocumentTypeRefundReceipt               DocumentType = "REFUND_RECEIPT"
	DocumentTypeAggregation                 DocumentType = "AGGREGATION"
	DocumentTypeUtilisation                 DocumentType = "UTILISATION"
	DocumentTypeImport                      DocumentType = "IMPORT"
	DocumentTypeCustomsCodeRegistration     DocumentType = "CUSTOMS_CODE_REGISTRATION"
	DocumentTypeCustomsCodeWithdrawal       DocumentType = "CUSTOMS_CODE_WITHDRAWAL"
	DocumentTypeOriginalCustomsDeclaration  DocumentType = "ORIGINAL_CUSTOMS_DECLARATION"
	DocumentTypeTransportCodeDisaggregation DocumentType = "TRANSPORT_CODE_DISAGGREGATION"
	DocumentTypeTransferRequest             DocumentType = "TRANSFER_REQUEST"
	DocumentTypeTransferConfirmation        DocumentType = "TRANSFER_CONFIRMATION"
	DocumentTypeWriteOffNotice              DocumentType = "WRITE_OFF_NOTICE"
	DocumentTypeStockReport                 DocumentType = "STOCK_REPORT"
	DocumentTypeExternalEnrollment          DocumentType = "EXTERNAL_ENROLLMENT"
	DocumentTypeValidation                  DocumentType = "VALIDATION"
	DocumentTypeTypographic                 DocumentType = "TYPOGRAPHIC"
	DocumentTypeOrder                       DocumentType = "ORDER"
	DocumentTypeOriginalInvoice             DocumentType = "ORIGINAL_INVOICE"
	DocumentTypeAdditionalInvoice           DocumentType = "ADDITIONAL_INVOICE"
	DocumentTypeCorrectedInvoice            DocumentType = "CORRECTED_INVOICE"
	DocumentTypeCancelledInvoice            DocumentType = "CANCELLED_INVOICE"
	DocumentTypeOwnerChangeNotice           DocumentType = "OWNER_CHANGE_NOTICE"
	DocumentTypeCancellation                DocumentType = "CANCELLATION"
	DocumentTypeCodeCorrection              DocumentType = "CODE_CORRECTION" // not in doc, real ASL value
)

// PackageType is the package level of a code; ASL calls it cisType or packageType (doc 13.13).
type PackageType string

// Package types.
const (
	PackageUnit   PackageType = "UNIT"     // consumer package (one item)
	PackageGroup  PackageType = "GROUP"    // group package (block)
	PackageSet    PackageType = "SET"      // set of different items
	PackageBoxLv1 PackageType = "BOX_LV_1" // transport package level 1 (box, SSCC)
	PackageBoxLv2 PackageType = "BOX_LV_2" // transport package level 2 (pallet, SSCC)
	PackageACC    PackageType = "ACC"      // aggregated customs code (AIK)
)

// Template is the format of a marking code (doc 13.14).
type Template string

// Code templates.
const (
	TemplateGS1AIStrShort    Template = "GS1_AISTR_SHORT"      // GS1, 4 or 8 chars check code
	TemplateGS1AIStrAsymShrt Template = "GS1_AISTR_ASYM_SHORT" // GS1, key and 44 chars check code
	TemplateGS1AIStr         Template = "GS1_AISTR"            // GS1, key and 88 chars check code
	TemplateSSCC             Template = "SSCC"                 // 20 digits with leading 00
	TemplateCustom           Template = "CUSTOM"               // any chars
)

// CodeStatus is the status of a marking code (doc 13.15).
type CodeStatus string

// Code statuses.
const (
	CodeStatusReceived   CodeStatus = "RECEIVED"    // code is downloaded from order
	CodeStatusApplied    CodeStatus = "APPLIED"     // code is printed on the goods
	CodeStatusIntroduced CodeStatus = "INTRODUCED"  // goods are on the market
	CodeStatusWithdrawn  CodeStatus = "WITHDRAWN"   // goods are sold or removed from market
	CodeStatusWrittenOff CodeStatus = "WRITTEN_OFF" // code is written off
)

// EmissionType is the marking goal of a code (doc 13.16).
type EmissionType string

// Emission types.
const (
	EmissionPrimary   EmissionType = "PRIMARY"
	EmissionRemains   EmissionType = "REMAINS"
	EmissionComission EmissionType = "COMISSION"
	EmissionRemark    EmissionType = "REMARK"
	EmissionExternal  EmissionType = "EXTERNAL" // code is accepted from outside
	EmissionShipping  EmissionType = "SHIPPING"
	EmissionCustoms   EmissionType = "CUSTOMS"
)

// ExtendedStatus is the extra status of a code (doc 13.18).
type ExtendedStatus string

// Extended statuses. ASL doc writes ACC with Cyrillic "С"; we use Latin letters as the API does.
const (
	ExtendedCustomsIssued    ExtendedStatus = "CUSTOMS_ISSUED"
	ExtendedCustomsAccepted  ExtendedStatus = "CUSTOMS_ACCEPTED"
	ExtendedInsertIntoACC    ExtendedStatus = "INSERT_INTO_ACC"
	ExtendedACCAggregated    ExtendedStatus = "ACC_AGGREGATED"
	ExtendedACCDisaggregated ExtendedStatus = "ACC_DISAGGREGATED"
	ExtendedWaitShipment     ExtendedStatus = "WAIT_SHIPMENT"
	ExtendedWaitingForImport ExtendedStatus = "WAITING_FOR_IMPORT"
	ExtendedConnected        ExtendedStatus = "CONNECTED"
)

// EventType is the type of an event in code history (doc 13.19).
type EventType string

// Event types.
const (
	EventUsage              EventType = "USAGE" // code is given to the owner
	EventUtilisation        EventType = "UTILISATION"
	EventValidation         EventType = "VALIDATION"
	EventPayment            EventType = "PAYMENT"
	EventChangeParent       EventType = "CHANGE_PARENT"
	EventUpdateChildren     EventType = "UPDATE_CHILDREN"
	EventIntroduction       EventType = "INTRODUCTION"
	EventWithdraw           EventType = "WITHDRAW"
	EventReturn             EventType = "RETURN"
	EventOwnerChange        EventType = "OWNER_CHANGE"
	EventEnrollment         EventType = "ENROLLMENT"
	EventWriteOff           EventType = "WRITE_OFF"
	EventOwnerReturn        EventType = "OWNER_RETURN"
	EventReversal           EventType = "REVERSAL"
	EventCustomsControl     EventType = "CUSTOMS_CONTROL"
	EventCustomsRefusal     EventType = "CUSTOMS_REFUSAL"
	EventCodeCorrection     EventType = "CODE_CORRECTION"
	EventStockUpdate        EventType = "STOCK_UPDATE"
	EventExternalEnrollment EventType = "EXTERNAL_ENROLLMENT"
	EventEnrichment         EventType = "ENRICHMENT"
)

// WithdrawalReason is why a code left the market or was written off (doc 13.20).
type WithdrawalReason string

// Withdrawal reasons.
const (
	WithdrawalDefect                WithdrawalReason = "DEFECT"
	WithdrawalSamples               WithdrawalReason = "SAMPLES"
	WithdrawalOther                 WithdrawalReason = "OTHER"
	WithdrawalProductionUse         WithdrawalReason = "PRODUCTION_USE"
	WithdrawalExpiration            WithdrawalReason = "EXPIRATION"
	WithdrawalConfiscation          WithdrawalReason = "CONFISCATION"
	WithdrawalProductRecall         WithdrawalReason = "PRODUCT_RECALL"
	WithdrawalComplaints            WithdrawalReason = "COMPLAINTS"
	WithdrawalLoss                  WithdrawalReason = "LOSS"
	WithdrawalDestruction           WithdrawalReason = "DESTRUCTION"
	WithdrawalRetail                WithdrawalReason = "RETAIL"
	WithdrawalExport                WithdrawalReason = "EXPORT"
	WithdrawalReturn                WithdrawalReason = "RETURN"
	WithdrawalDistance              WithdrawalReason = "DISTANCE"
	WithdrawalFreePrescription      WithdrawalReason = "FREE_PRESCRIPTION"
	WithdrawalMedicalPurpose        WithdrawalReason = "MEDICAL_PURPOSE"
	WithdrawalPacking               WithdrawalReason = "PACKING"
	WithdrawalReceiptSale           WithdrawalReason = "RECEIPT_SALE"
	WithdrawalAutoDisaggregated     WithdrawalReason = "AUTO_DISAGGREGATED"
	WithdrawalManualDisaggregated   WithdrawalReason = "MANUAL_DISAGGREGATED"
	WithdrawalInventory             WithdrawalReason = "INVENTORY"
	WithdrawalProcessing            WithdrawalReason = "PROCESSING"
	WithdrawalExportNotice          WithdrawalReason = "EXPORT_NOTICE"
	WithdrawalEnterpriseUse         WithdrawalReason = "ENTERPRISE_USE"
	WithdrawalIdleCodeCleanup       WithdrawalReason = "IDLE_CODE_CLEANUP"
	WithdrawalGNKInventory          WithdrawalReason = "GNK_INVENTORY"
	WithdrawalReceiptSaleHoreca     WithdrawalReason = "RECEIPT_SALE_HORECA"
	WithdrawalInvoicePurchaseHoreca WithdrawalReason = "INVOICE_PURCHASE_HORECA"
	WithdrawalUsedHoreca            WithdrawalReason = "USED_HORECA"
	WithdrawalVending               WithdrawalReason = "VENDING"
)

// ReturnReason is why a code came back to the market (doc 13.21).
type ReturnReason string

// Return reasons.
const (
	ReturnRetail              ReturnReason = "RETAIL_RETURN"
	ReturnReceipt             ReturnReason = "RECEIPT_RETURN"
	ReturnProductionUse       ReturnReason = "PRODUCTION_USE_RETURN"
	ReturnOwnUse              ReturnReason = "OWN_USE_RETURN"
	ReturnNotForSale          ReturnReason = "NOT_FOR_SALE_RETURN"
	ReturnReceiptHoreca       ReturnReason = "RECEIPT_RETURN_HORECA"
	ReturnInvoiceReturnHoreca ReturnReason = "INVOICE_RETURN_HORECA"
)

// ExportStatus is the status of a code list export (doc 13.22).
type ExportStatus string

// Export statuses.
const (
	ExportStatusCreated      ExportStatus = "CREATED"
	ExportStatusInProcessing ExportStatus = "IN_PROCESSING"
	ExportStatusSuccess      ExportStatus = "SUCCESS" // file is ready to download
	ExportStatusError        ExportStatus = "ERROR"   // also when the list is too big
	ExportStatusExpired      ExportStatus = "EXPIRED" // file was deleted after 60 minutes
)

// IsFinal reports whether ASL finished the export.
func (s ExportStatus) IsFinal() bool {
	return s == ExportStatusSuccess || s == ExportStatusError || s == ExportStatusExpired
}

// PrintQualityClass is the print quality of a DataMatrix code by ISO/IEC 15415 (doc 13.23).
type PrintQualityClass string

// Print quality classes.
const (
	PrintQualityA PrintQualityClass = "A" // best quality (4)
	PrintQualityB PrintQualityClass = "B" // good quality (3)
	PrintQualityC PrintQualityClass = "C" // normal quality (2)
	PrintQualityD PrintQualityClass = "D" // bad quality (1)
	PrintQualityF PrintQualityClass = "F" // broken print (0)
)

// AdditionalUnit is the extra unit of measure for customs (doc 13.24).
type AdditionalUnit int

// Additional units.
const (
	AdditionalUnitPiece    AdditionalUnit = 796 // pieces
	AdditionalUnitThousand AdditionalUnit = 798 // 1000 pieces
	AdditionalUnitLiter    AdditionalUnit = 112 // liters
)

// ProductCardStatus is the status of a product card (doc 13.25).
type ProductCardStatus string

// Product card statuses.
const (
	ProductCardPublished ProductCardStatus = "PUBLISHED"
	ProductCardBlocked   ProductCardStatus = "BLOCKED"
	ProductCardInactive  ProductCardStatus = "INACTIVE"
)

// ProductScope is the type of a product card barcode (doc 13.26).
type ProductScope string

// Product scopes.
const (
	ProductScopeLocal     ProductScope = "LOCAL"     // barcode registered in Uzbekistan
	ProductScopeGlobal    ProductScope = "GLOBAL"    // barcode registered in other country
	ProductScopeTechnical ProductScope = "TECHNICAL" // temporary or system barcode
)

// CodeState is the result for one code inside a processed document (doc 11.5).
type CodeState string

// Code states.
const (
	CodeStateSuccess CodeState = "SUCCESS"
	CodeStateWarning CodeState = "WARNING" // code is valid but its state did not change
	CodeStateError   CodeState = "ERROR"
)
