package dto

// CustomsCodeResponse is a new empty aggregated customs code (AIK).
type CustomsCodeResponse struct {
	AggregatedCustomsCode string `json:"aggregatedCustomsCode"`
}

// CustomsCodeDocument fills an AIK with codes and product data; it needs a signature.
type CustomsCodeDocument struct {
	AggregatedCustomsCode  string          `json:"aggregatedCustomsCode"`
	GTIN                   string          `json:"gtin"`
	ProductID              string          `json:"productId"` // product card ID
	ProductGroup           ProductGroup    `json:"productGroup"`
	Tnved                  string          `json:"tnved"`
	Quantity               int             `json:"quantity"`               // count of consumer packages
	AdditionalUnit         *AdditionalUnit `json:"additionalUnit"`         // null when not used
	AdditionalUnitQuantity *float64        `json:"additionalUnitQuantity"` // null when not used
	Codes                  []string        `json:"codes"`
}

// CustomsCodeWithdrawalDocument breaks a registered AIK; it needs a signature.
type CustomsCodeWithdrawalDocument struct {
	BusinessDatetime DateTime `json:"businessDatetime"`
	Code             string   `json:"code"`
}
