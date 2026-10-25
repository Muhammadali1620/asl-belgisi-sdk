package dto

// UtilisationRequest is a report that codes are printed on goods (max 30 000 codes).
type UtilisationRequest struct {
	ProductGroup        ProductGroup `json:"-"`      // sent in URL query, not in body
	Sntins              []string     `json:"sntins"` // full codes with GS separators and check code
	BusinessPlaceID     int          `json:"businessPlaceId"`
	ReleaseType         ReleaseType  `json:"releaseType"`         // CIRCULATION is not allowed
	ManufacturerCountry CountryCode  `json:"manufacturerCountry"` // UZ needs PRODUCTION, others IMPORT
	ProductionOrderID   string       `json:"productionOrderId,omitempty"`
	ProductionDate      *DateTime    `json:"productionDate,omitempty"` // required except appliances
	ExpirationDate      *DateTime    `json:"expirationDate,omitempty"` // required except appliances
	SeriesNumber        string       `json:"seriesNumber,omitempty"`   // 1-20 chars, required for pharma
}

// UtilisationResponse is the ID of the utilisation report (it is also a document ID).
type UtilisationResponse struct {
	ReportID string `json:"reportId"`
}

// ValidationDocument is a print quality report; it is sent as base64 JSON.
type ValidationDocument struct {
	ProductGroup      ProductGroup     `json:"productGroup"`
	BusinessPlaceID   int              `json:"businessPlaceId"`
	ProductionOrderID string           `json:"productionOrderId,omitempty"`
	PackageType       PackageType      `json:"packageType"`
	DocumentDate      DateTime         `json:"documentDate"`
	Codes             []ValidationCode `json:"codes"`
}

// ValidationCode is one checked code (identity code without check code) and its print quality.
type ValidationCode struct {
	Code              string            `json:"code"`
	PrintQualityClass PrintQualityClass `json:"printQualityClass"`
}

// AggregationDocument is a report that items are packed into parent packages.
type AggregationDocument struct {
	AggregationUnits  []AggregationUnit `json:"aggregationUnits"`
	BusinessPlaceID   int               `json:"businessPlaceId"`
	DocumentDate      DateTime          `json:"documentDate"` // must be after production date
	ProductionOrderID string            `json:"productionOrderId,omitempty"`
}

// AggregationUnit is one parent package and its child codes.
type AggregationUnit struct {
	UnitSerialNumber        string   `json:"unitSerialNumber"`        // parent code (group code or SSCC)
	AggregationUnitCapacity int      `json:"aggregationUnitCapacity"` // planned capacity
	AggregationItemsCount   int      `json:"aggregationItemsCount"`   // real count, <= capacity
	Codes                   []string `json:"codes"`                   // child identity codes or SSCC
	ShouldBeUnbundled       bool     `json:"shouldBeUnbundled"`       // true: take out of old parent
}

// CorrectionDocument changes dates, series or country of codes.
type CorrectionDocument struct {
	BusinessDatetime DateTime      `json:"businessDatetime"`
	UpdatedFields    UpdatedFields `json:"updatedFields"`
	Codes            []string      `json:"codes"`
}

// UpdatedFields are new values for codes; set one or more fields.
type UpdatedFields struct {
	ExpirationDatetime  *DateTime   `json:"expirationDatetime,omitempty"`
	ProductionDatetime  *DateTime   `json:"productionDatetime,omitempty"`
	ManufacturerCountry CountryCode `json:"manufacturerCountry,omitempty"`
	SeriesNumber        string      `json:"seriesNumber,omitempty"`
}

// DisaggregationDocument breaks group or transport packages; children keep their status.
type DisaggregationDocument struct {
	BusinessDatetime DateTime `json:"businessDatetime"`
	Codes            []string `json:"codes"` // parent codes (group codes or SSCC)
}
