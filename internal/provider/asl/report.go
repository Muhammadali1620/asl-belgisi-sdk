package asl

import (
	"context"
	"unicode/utf8"

	"asl/internal/dto"
)

// Report paths.
const (
	pathValidation     = "/public/api/v1/doc/validation"
	pathAggregation    = "/public/api/v1/doc/aggregation"
	pathCorrection     = "/public/api/v1/doc/correction"
	pathDisaggregation = "/public/api/v1/doc/transport-code-disaggregation"
)

func (c *Client) SendUtilisation(
	ctx context.Context, req dto.UtilisationRequest,
) (*dto.UtilisationResponse, error) {
	// POST /api/utilisation?productGroup=... Codes must be full (with GS and check code),
	// in status RECEIVED, of one product group. Check result with GetDocument(reportId).
	if err := validateUtilisation(req); err != nil {
		return nil, err
	}
	values := query{}.set("productGroup", string(req.ProductGroup))
	var out dto.UtilisationResponse
	if err := c.post(ctx, pathUtilisation, values, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// validateUtilisation checks a utilisation report before it is sent.
func validateUtilisation(req dto.UtilisationRequest) error {
	// Dates are checked by ASL, because only ASL knows its own clock and the order date.
	if err := requireText("productGroup", string(req.ProductGroup)); err != nil {
		return err
	}
	if err := requireCodes("sntins", req.Sntins, MaxReportCodes); err != nil {
		return err
	}
	if req.BusinessPlaceID <= 0 {
		return invalidArgument("businessPlaceId", "businessPlaceId must be positive")
	}
	if err := requireText("releaseType", string(req.ReleaseType)); err != nil {
		return err
	}
	if req.ReleaseType == dto.ReleaseCirculation {
		return invalidArgument("releaseType", "CIRCULATION is not allowed in utilisation")
	}
	if err := requireText("manufacturerCountry", string(req.ManufacturerCountry)); err != nil {
		return err
	}
	return checkSeries("seriesNumber", req.SeriesNumber)
}

// checkSeries checks the series (batch) number length.
func checkSeries(param, series string) error {
	// Empty series is allowed; ASL says when it is required for the product group.
	if length := utf8.RuneCountInString(series); length > MaxSeriesNumberLen {
		return invalidArgument(param, "%s must be 1..%d chars, got %d",
			param, MaxSeriesNumberLen, length)
	}
	return nil
}

func (c *Client) SendValidation(
	ctx context.Context, doc dto.ValidationDocument,
) (*dto.DocumentIDResponse, error) {
	// POST /public/api/v1/doc/validation with base64 document. Codes are identity codes
	// without check code; each has a print quality class by ISO/IEC 15415.
	if err := requireText("productGroup", string(doc.ProductGroup)); err != nil {
		return nil, err
	}
	if doc.BusinessPlaceID <= 0 {
		return nil, invalidArgument("businessPlaceId", "businessPlaceId must be positive")
	}
	if err := requireText("packageType", string(doc.PackageType)); err != nil {
		return nil, err
	}
	if len(doc.Codes) == 0 || len(doc.Codes) > MaxReportCodes {
		return nil, invalidArgument("codes", "codes must have 1..%d items, got %d",
			MaxReportCodes, len(doc.Codes))
	}
	for index, code := range doc.Codes {
		if code.Code == "" || code.PrintQualityClass == "" {
			return nil, invalidArgument("codes", "codes[%d] needs code and printQualityClass", index)
		}
	}
	return c.sendDocument(ctx, pathValidation, doc)
}

func (c *Client) SendAggregation(
	ctx context.Context, doc dto.AggregationDocument,
) (*dto.DocumentIDResponse, error) {
	// POST /public/api/v1/doc/aggregation with base64 document. One document may hold many
	// packages and many levels. Child codes are identity codes (no crypto tail) or SSCC.
	if err := validateAggregation(doc); err != nil {
		return nil, err
	}
	return c.sendDocument(ctx, pathAggregation, doc)
}

// validateAggregation checks counts and codes of all packages.
func validateAggregation(doc dto.AggregationDocument) error {
	// The total count of child codes in one document is limited too.
	if len(doc.AggregationUnits) == 0 {
		return invalidArgument("aggregationUnits", "add at least one package")
	}
	if doc.BusinessPlaceID <= 0 {
		return invalidArgument("businessPlaceId", "businessPlaceId must be positive")
	}
	if doc.DocumentDate.IsZero() {
		return invalidArgument("documentDate", "documentDate is required")
	}
	total := 0
	for index, unit := range doc.AggregationUnits {
		if err := requireText("aggregationUnits.unitSerialNumber", unit.UnitSerialNumber); err != nil {
			return err
		}
		if err := requireCodes("aggregationUnits.codes", unit.Codes, MaxReportCodes); err != nil {
			return err
		}
		count, capacity := unit.AggregationItemsCount, unit.AggregationUnitCapacity
		if count < 1 || count > capacity {
			return invalidArgument("aggregationUnits.aggregationItemsCount",
				"aggregationUnits[%d]: items count %d must be 1..capacity %d", index, count, capacity)
		}
		total += len(unit.Codes)
	}
	if total > MaxReportCodes {
		return invalidArgument("aggregationUnits.codes", "document has %d codes, max is %d",
			total, MaxReportCodes)
	}
	return nil
}

func (c *Client) SendCorrection(
	ctx context.Context, doc dto.CorrectionDocument,
) (*dto.DocumentIDResponse, error) {
	// POST /public/api/v1/doc/correction with base64 document (keys sorted A-Z).
	// Changes production date, expiration date, series or country for the codes.
	if err := requireCodes("codes", doc.Codes, MaxReportCodes); err != nil {
		return nil, err
	}
	if doc.BusinessDatetime.IsZero() {
		return nil, invalidArgument("businessDatetime", "businessDatetime is required")
	}
	fields := doc.UpdatedFields
	if fields.ExpirationDatetime == nil && fields.ProductionDatetime == nil &&
		fields.ManufacturerCountry == "" && fields.SeriesNumber == "" {
		return nil, invalidArgument("updatedFields", "set at least one field in updatedFields")
	}
	if err := checkSeries("updatedFields.seriesNumber", fields.SeriesNumber); err != nil {
		return nil, err
	}
	return c.sendDocument(ctx, pathCorrection, doc)
}

func (c *Client) SendDisaggregation(
	ctx context.Context, doc dto.DisaggregationDocument,
) (*dto.DocumentIDResponse, error) {
	// POST /public/api/v1/doc/transport-code-disaggregation with base64 document.
	// Children keep their status; the parent of the broken package is broken too.
	if err := requireCodes("codes", doc.Codes, MaxReportCodes); err != nil {
		return nil, err
	}
	if doc.BusinessDatetime.IsZero() {
		return nil, invalidArgument("businessDatetime", "businessDatetime is required")
	}
	return c.sendDocument(ctx, pathDisaggregation, doc)
}
