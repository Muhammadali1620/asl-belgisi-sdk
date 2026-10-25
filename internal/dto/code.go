package dto

// PublicCodesRequest asks for public info about codes (max 1000 identity codes, no check code).
type PublicCodesRequest struct {
	Codes          []string `json:"codes"`
	AddCodeHistory *bool    `json:"addCodeHistory,omitempty"` // ASL default is false
}

// PublicCodeInfo is short public info about one code; unknown codes are not returned.
type PublicCodeInfo struct {
	Code        string      `json:"code"`
	PackageType PackageType `json:"packageType"`
	Status      CodeStatus  `json:"status"`
	// Code had extra AI
	IsHadExtendedCode bool            `json:"isHadExtendedCode"`
	ExtendedStatus    ExtendedStatus  `json:"extendedStatus,omitempty"`
	IssuerShortInfo   IssuerShortInfo `json:"issuerShortInfo"`
	Template          Template        `json:"template,omitempty"`
	// Empty for BOX_LV_1/2
	GTIN                            string                  `json:"gtin,omitempty"`
	ProductID                       string                  `json:"productId,omitempty"`
	ProductGroupID                  int                     `json:"productGroupId,omitempty"`
	CategoryID                      string                  `json:"categoryId,omitempty"`
	EmissionDate                    *DateTime               `json:"emissionDate,omitempty"`
	IssueDate                       *DateTime               `json:"issueDate,omitempty"`
	ProductionDate                  *DateTime               `json:"productionDate,omitempty"`
	ExpirationDate                  *DateTime               `json:"expirationDate,omitempty"`
	ProductSeries                   string                  `json:"productSeries,omitempty"`
	MixedProductGroups              *bool                   `json:"mixedProductGroups,omitempty"`
	MixedCategories                 *bool                   `json:"mixedCategories,omitempty"`
	AggregateProductGroups          []AggregateProductGroup `json:"aggregateProductGroups,omitempty"`
	AggregateCategories             []AggregateCategory     `json:"aggregateCategories,omitempty"`
	FirstWithdrawCodeStateUnchanged *bool                   `json:"firstWithdrawCodeStateUnchanged,omitempty"`
	CodeHistory                     []CodeHistoryEvent      `json:"codeHistory,omitempty"`
}

// IssuerShortInfo is the TIN and name of the code issuer.
type IssuerShortInfo struct {
	IssuerTin  string        `json:"issuerTin"`
	IssuerName LocalizedText `json:"issuerName,omitempty"`
}

// AggregateProductGroup is how many consumer codes of one group are inside a package.
type AggregateProductGroup struct {
	ProductGroupID int `json:"productGroupId"`
	UnitsNumber    int `json:"unitsNumber"`
}

// AggregateCategory is how many consumer codes of one category are inside a package.
type AggregateCategory struct {
	CategoryID  string `json:"categoryId"`
	UnitsNumber int    `json:"unitsNumber"`
}

// PrivateCodesRequest asks for full info about our codes (max 1000 identity codes).
type PrivateCodesRequest struct {
	Codes          []string `json:"codes"`
	AddCodeHistory *bool    `json:"addCodeHistory,omitempty"` // ASL default is true
	AddChildCodes  *bool    `json:"addChildCodes,omitempty"`  // ASL default is true
}

// PrivateCodesResponse is full info about codes; codes without access are in ForbiddenCodes.
// When no code is accessible ASL redirects to public info: then Redirected and PublicResults are set.
type PrivateCodesResponse struct {
	Results        []PrivateCodeInfo `json:"results"`
	ForbiddenCodes []string          `json:"forbiddenCodes"`
	Redirected     bool              `json:"-"`
	PublicResults  []PublicCodeInfo  `json:"-"`
}

// PrivateCodeInfo is full info about one code.
type PrivateCodeInfo struct {
	CodeData     CodeData           `json:"codeData"`
	ProductData  *ProductData       `json:"productData,omitempty"`
	PackageData  PackageData        `json:"packageData"`
	MarkingData  MarkingData        `json:"markingData"`
	TurnoverData *TurnoverData      `json:"turnoverData,omitempty"`
	CodeHistory  []CodeHistoryEvent `json:"codeHistory,omitempty"`
}

// CodeData is the code, its status and template.
type CodeData struct {
	Code                            string         `json:"code"`
	Status                          CodeStatus     `json:"status"`
	Template                        Template       `json:"template,omitempty"`
	ExtendedStatus                  ExtendedStatus `json:"extendedStatus,omitempty"`
	FirstWithdrawCodeStateUnchanged *bool          `json:"firstWithdrawCodeStateUnchanged,omitempty"`
}

// ProductData is info about the marked product.
type ProductData struct {
	ProductID           string      `json:"productId,omitempty"`
	GTIN                string      `json:"gtin,omitempty"`
	ProductGroupID      int         `json:"productGroupId,omitempty"`
	CategoryID          string      `json:"categoryId,omitempty"`
	ProductionDate      *DateTime   `json:"productionDate,omitempty"`
	ExpirationDate      *DateTime   `json:"expirationDate,omitempty"`
	ProductSeries       string      `json:"productSeries,omitempty"`
	ManufacturerCountry CountryCode `json:"manufacturerCountry,omitempty"`
	MixedProductGroups  *bool       `json:"mixedProductGroups,omitempty"`
	MixedCategories     *bool       `json:"mixedCategories,omitempty"`
}

// PackageData is the package level, parent code and child codes.
type PackageData struct {
	PackageType  PackageType `json:"packageType"`
	EmptyPackage bool        `json:"emptyPackage"`
	ParentCode   string      `json:"parentCode,omitempty"`
	// Empty when addChildCodes=false
	Children               []ChildCode             `json:"children,omitempty"`
	ActuallyPacked         int                     `json:"actuallyPacked"` // count of child codes
	AggregateProductGroups []AggregateProductGroup `json:"aggregateProductGroups,omitempty"`
	AggregateCategories    []AggregateCategory     `json:"aggregateCategories,omitempty"`
}

// ChildCode is one code inside a package.
type ChildCode struct {
	Code               string         `json:"code"`
	Status             CodeStatus     `json:"status"`
	ExtendedStatus     ExtendedStatus `json:"extendedStatus,omitempty"`
	PackageType        PackageType    `json:"packageType"`
	EmptyPackage       *bool          `json:"emptyPackage,omitempty"`
	MixedProductGroups *bool          `json:"mixedProductGroups,omitempty"`
	MixedCategories    *bool          `json:"mixedCategories,omitempty"`
	ProductID          string         `json:"productId,omitempty"`
	ProductGroupID     int            `json:"productGroupId,omitempty"`
	CategoryID         string         `json:"categoryId,omitempty"`
}

// MarkingData is info about code emission, printing and payment.
type MarkingData struct {
	EmissionDate          *DateTime          `json:"emissionDate,omitempty"`
	IssueDate             *DateTime          `json:"issueDate,omitempty"`
	IssuerInfo            *IssuerInfo        `json:"issuerInfo,omitempty"`
	ContractorInfo        *ContractorDetails `json:"contractorInfo,omitempty"`
	EmissionType          EmissionType       `json:"emissionType,omitempty"`
	IssuerBusinessPlaceID int                `json:"issuerBusinessPlaceId,omitempty"`
	ValidationDate        *DateTime          `json:"validationDate,omitempty"`
	UtilisationDate       *DateTime          `json:"utilisationDate,omitempty"`
	PaymentDate           *DateTime          `json:"paymentDate,omitempty"`
}

// IssuerInfo is the participant who ordered the code.
type IssuerInfo struct {
	IssuerID              int           `json:"issuerId,omitempty"`
	IssuerTin             string        `json:"issuerTin,omitempty"`
	IssuerName            LocalizedText `json:"issuerName,omitempty"`
	IssuerBusinessPlaceID int           `json:"issuerBusinessPlaceId,omitempty"`
	IssuerPartyKindID     int           `json:"issuerPartyKindId,omitempty"`
}

// ContractorDetails is the service provider who got the code.
type ContractorDetails struct {
	ContractorID              int           `json:"contractorId,omitempty"`
	ContractorTin             string        `json:"contractorTin,omitempty"`
	ContractorName            LocalizedText `json:"contractorName,omitempty"`
	ContractorBusinessPlaceID int           `json:"contractorBusinessPlaceId,omitempty"`
}

// TurnoverData is info about the code owner and market state.
type TurnoverData struct {
	OriginalReleaseMethod ReleaseType         `json:"originalReleaseMethod,omitempty"`
	OwnerInfo             OwnerInfo           `json:"ownerInfo"`
	WithdrawalDate        *DateTime           `json:"withdrawalDate,omitempty"`
	WithdrawalReason      WithdrawalReason    `json:"withdrawalReason,omitempty"`
	PartialQuantity       *float64            `json:"partialQuantity,omitempty"`
	ReturnDate            *DateTime           `json:"returnDate,omitempty"`
	ReturnReason          ReturnReason        `json:"returnReason,omitempty"`
	CustomsDeclaration    *CustomsDeclaration `json:"customsDeclaration,omitempty"`
	ManufacturerCountry   CountryCode         `json:"manufacturerCountry,omitempty"`
}

// OwnerInfo is the current owner of the code.
type OwnerInfo struct {
	OwnerID              int           `json:"ownerId,omitempty"`
	OwnerTin             string        `json:"ownerTin"`
	OwnerName            LocalizedText `json:"ownerName,omitempty"`
	OwnerBusinessPlaceID int           `json:"ownerBusinessPlaceId,omitempty"`
	OwnerPartyKindID     int           `json:"ownerPartyKindId,omitempty"`
}

// CustomsDeclaration is the customs declaration of imported goods.
type CustomsDeclaration struct {
	Date          *Date  `json:"date,omitempty"`
	Number        string `json:"number,omitempty"`
	AuthorityCode string `json:"authorityCode,omitempty"`
}

// CodeHistoryEvent is one event with the code; only events made by us are returned.
type CodeHistoryEvent struct {
	EventBusinessDate         *DateTime         `json:"eventBusinessDate,omitempty"`
	EventDate                 *DateTime         `json:"eventDate,omitempty"`
	EventType                 EventType         `json:"eventType"`
	EventSourceID             string            `json:"eventSourceId"` // document ID of the event
	DocumentType              DocumentType      `json:"documentType,omitempty"`
	ExternalDocID             string            `json:"externalDocId,omitempty"`
	SenderTin                 string            `json:"senderTin,omitempty"`
	SenderName                LocalizedText     `json:"senderName,omitempty"`
	ReceiverTin               string            `json:"receiverTin,omitempty"`
	ReceiverName              LocalizedText     `json:"receiverName,omitempty"`
	EventChangedCodeStatus    CodeStatus        `json:"eventChangedCodeStatus,omitempty"`
	EventChangedCodeExtStatus ExtendedStatus    `json:"eventChangedCodeExtStatus,omitempty"`
	CodeStateChanged          bool              `json:"codeStateChanged"`
	EventDescription          *EventDescription `json:"eventDescription,omitempty"`
}

// EventDescription holds extra data of an event; only one field is set for each event type.
type EventDescription struct {
	Validation         *EventValidationInfo         `json:"eventValidationShortResponse,omitempty"`
	Utilisation        *EventUtilisationInfo        `json:"eventUtilisationShortResponse,omitempty"`
	Payment            *EventPaymentInfo            `json:"eventPaymentShortResponse,omitempty"`
	UpdateChildren     *EventUpdateChildrenInfo     `json:"eventUpdateChildrenShortResponse,omitempty"`
	ChangeParent       *EventChangeParentInfo       `json:"eventChangeParentShortResponse,omitempty"`
	Introduction       *EventIntroductionInfo       `json:"eventIntroductionShortResponse,omitempty"`
	Withdraw           *EventWithdrawInfo           `json:"eventWithdrawShortResponse,omitempty"`
	Dropout            *EventDropoutInfo            `json:"eventDropoutShortResponse,omitempty"`
	Return             *EventReturnInfo             `json:"eventReturnShortResponse,omitempty"`
	OwnerChange        *EventOwnerChangeInfo        `json:"eventOwnerChangeShortResponse,omitempty"`
	CodeCorrection     *EventCodeCorrectionInfo     `json:"eventCodeCorrectionShortResponse,omitempty"`
	OwnerReturn        *EventOwnerReturnInfo        `json:"eventOwnerReturnShortResponse,omitempty"`
	Reversal           *EventReversalInfo           `json:"eventReversalShortResponse,omitempty"`
	CustomsControl     *EventCustomsControlInfo     `json:"eventCustomsControlShortResponse,omitempty"`
	StockUpdate        *EventStockUpdateInfo        `json:"eventStockUpdateShortResponse,omitempty"`
	ExternalEnrollment *EventExternalEnrollmentInfo `json:"eventExternalEnrollmentShortResponse,omitempty"`
}

// EventValidationInfo is data of a print quality event.
type EventValidationInfo struct {
	PrintQualityClass PrintQualityClass `json:"printQualityClass,omitempty"`
}

// EventUtilisationInfo is data of a printing (utilisation) event.
type EventUtilisationInfo struct {
	ManufacturerCountry         CountryCode     `json:"manufacturerCountry,omitempty"`
	AddedApplicationIdentifiers []ApplicationID `json:"addedApplicationIdentifier,omitempty"`
}

// ApplicationID is one GS1 data group added to the code during printing.
type ApplicationID struct {
	AICode  string `json:"aiCode,omitempty"`
	AIValue string `json:"aiValue,omitempty"`
}

// EventPaymentInfo is data of a payment event.
type EventPaymentInfo struct {
	ChargeID string `json:"chargeId,omitempty"`
}

// EventUpdateChildrenInfo is data of an event that adds or removes child codes.
type EventUpdateChildrenInfo struct {
	MaxPackageCapacity int   `json:"maxPackageCapacity,omitempty"`
	ActuallyPacked     int   `json:"actuallyPacked"`
	ActuallyRemoved    int   `json:"actuallyRemoved"`
	EmptyPackage       *bool `json:"emptyPackage,omitempty"`
}

// EventChangeParentInfo is data of an event that changes the parent code.
type EventChangeParentInfo struct {
	FormerParentCode string `json:"formerParentCode,omitempty"`
	ActualParentCode string `json:"actualParentCode,omitempty"`
}

// EventIntroductionInfo is data of an event that puts goods on the market.
type EventIntroductionInfo struct {
	OriginalReleaseMethod ReleaseType `json:"originalReleaseMethod,omitempty"`
}

// EventWithdrawInfo is data of a sale (withdraw) event.
type EventWithdrawInfo struct {
	WithdrawalReason                WithdrawalReason `json:"withdrawalReason,omitempty"`
	PartialQuantity                 *float64         `json:"partialQuantity,omitempty"`
	ChangedPartialQuantity          *float64         `json:"changedPartialQuantity,omitempty"`
	Price                           *float64         `json:"price,omitempty"`   // price with VAT
	VatRate                         *float64         `json:"vatRate,omitempty"` // VAT in percent
	FirstWithdrawCodeStateUnchanged *bool            `json:"firstWithdrawCodeStateUnchanged,omitempty"`
}

// EventDropoutInfo is data of a write-off event.
type EventDropoutInfo struct {
	WithdrawalReason WithdrawalReason `json:"withdrawalReason,omitempty"`
	ChildrenWriteOff *bool            `json:"childrenWriteOff,omitempty"`
}

// EventReturnInfo is data of a return to market event.
type EventReturnInfo struct {
	ReturnReason           ReturnReason `json:"returnReason,omitempty"`
	PartialQuantity        *float64     `json:"partialQuantity,omitempty"`
	ChangedPartialQuantity *float64     `json:"changedPartialQuantity,omitempty"`
}

// EventOwnerChangeInfo is data of an owner change event.
type EventOwnerChangeInfo struct {
	ReceiverTin  string        `json:"receiverTin,omitempty"`
	ReceiverName LocalizedText `json:"receiverName,omitempty"`
}

// EventCodeCorrectionInfo is data of a correction event: new values and old values.
type EventCodeCorrectionInfo struct {
	ProductionDate         *DateTime         `json:"productionDate,omitempty"`
	ProductSeries          string            `json:"productSeries,omitempty"`
	ManufacturerCountry    CountryCode       `json:"manufacturerCountry,omitempty"`
	ExpirationDate         *DateTime         `json:"expirationDate,omitempty"`
	ValuesBeforeCorrection *CorrectionValues `json:"valuesBeforeCorrection,omitempty"`
}

// CorrectionValues are code values before a correction.
type CorrectionValues struct {
	ProductionDate      *DateTime   `json:"productionDate,omitempty"`
	ProductSeries       string      `json:"productSeries,omitempty"`
	ManufacturerCountry CountryCode `json:"manufacturerCountry,omitempty"`
	ExpirationDate      *DateTime   `json:"expirationDate,omitempty"`
}

// EventOwnerReturnInfo is data of a return to supplier event.
type EventOwnerReturnInfo struct {
	ReceiverTin   string        `json:"receiverTin,omitempty"`
	ReceiverName  LocalizedText `json:"receiverName,omitempty"`
	PrincipalTin  string        `json:"principalTin,omitempty"`
	PrincipalName LocalizedText `json:"principalName,omitempty"`
}

// EventReversalInfo is data of a reversal (cancel) event.
type EventReversalInfo struct {
	ReceiverTin   string        `json:"receiverTin,omitempty"`
	ReceiverName  LocalizedText `json:"receiverName,omitempty"`
	PrincipalTin  string        `json:"principalTin,omitempty"`
	PrincipalName LocalizedText `json:"principalName,omitempty"`
	OwnerTin      string        `json:"ownerTin,omitempty"`
	OwnerName     LocalizedText `json:"ownerName,omitempty"`
}

// EventCustomsControlInfo is data of an AIK registration event.
type EventCustomsControlInfo struct {
	ExtendedStatus ExtendedStatus `json:"extendedStatus,omitempty"`
}

// EventStockUpdateInfo is data of a stock check event.
type EventStockUpdateInfo struct {
	OwnerTin string `json:"ownerTin,omitempty"`
}

// EventExternalEnrollmentInfo is data of an external code registration event.
type EventExternalEnrollmentInfo struct {
	OwnerPartyKindID    int                 `json:"ownerPartyKindId,omitempty"`
	SenderPartyKindID   int                 `json:"senderPartyKindId,omitempty"`
	ExtendedStatus      ExtendedStatus      `json:"extendedStatus,omitempty"`
	ProductSeries       string              `json:"productSeries,omitempty"`
	ProductionDate      *DateTime           `json:"productionDate,omitempty"`
	ManufacturerCountry CountryCode         `json:"manufacturerCountry,omitempty"`
	ExpirationDate      *DateTime           `json:"expirationDate,omitempty"`
	CustomsDeclaration  *CustomsDeclaration `json:"customsDeclaration,omitempty"`
}

// OwnerCheckRequest checks the owner of codes and gets child codes (max 100 codes).
type OwnerCheckRequest struct {
	Codes    []string `json:"codes"`
	OwnerTin string   `json:"ownerTin"`
}

// OwnerCheckResponse lists found codes, codes of other owners and unknown codes.
type OwnerCheckResponse struct {
	Results        []OwnerCheckResult `json:"results"`
	ForbiddenCodes []string           `json:"forbiddenCodes"` // owner TIN is different
	MissingCodes   []string           `json:"missingCodes"`   // codes not found in ASL
}

// OwnerCheckResult is one found code and its child codes.
type OwnerCheckResult struct {
	Code                            string          `json:"code"`
	ProductGroupID                  int             `json:"productGroupId,omitempty"`
	PackageType                     PackageType     `json:"packageType"`
	Status                          CodeStatus      `json:"status"`
	ExtendedStatus                  ExtendedStatus  `json:"extendedStatus,omitempty"`
	IssuerShortInfo                 IssuerShortInfo `json:"issuerShortInfo"`
	FirstWithdrawCodeStateUnchanged *bool           `json:"firstWithdrawCodeStateUnchanged,omitempty"`
	Children                        []string        `json:"children,omitempty"`
}

// VerifyResult tells if a full code is real; transport codes are always false.
type VerifyResult struct {
	Code         string `json:"code"`
	Verified     bool   `json:"verified"`
	ProductGroup int    `json:"productGroup,omitempty"` // group ID, only for real codes
}
