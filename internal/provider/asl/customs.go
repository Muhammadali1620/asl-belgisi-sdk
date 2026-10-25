package asl

import (
	"context"
	"math"

	"asl/internal/dto"
)

// Customs code (AIK) paths.
const (
	pathCustomsGenerate = "/public/api/v1/customs-registry/code"
	pathCustomsRegister = "/public/api/v1/doc/customs-code"
	pathCustomsWithdraw = "/public/api/v1/doc/customs-code-withdrawal"
)

func (c *Client) GenerateCustomsCode(ctx context.Context) (*dto.CustomsCodeResponse, error) {
	// POST /public/api/v1/customs-registry/code without body. The new AIK is empty:
	// register it with RegisterCustomsCode.
	var out dto.CustomsCodeResponse
	if err := c.post(ctx, pathCustomsGenerate, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RegisterCustomsCode(
	ctx context.Context, doc dto.CustomsCodeDocument,
) (*dto.DocumentIDResponse, error) {
	// POST /public/api/v1/doc/customs-code with base64 document. ASL needs a signature,
	// so set Config.Signer. Use CalcAdditionalUnitQuantity for additionalUnitQuantity.
	if err := requireText("aggregatedCustomsCode", doc.AggregatedCustomsCode); err != nil {
		return nil, err
	}
	if err := requireGTIN("gtin", doc.GTIN); err != nil {
		return nil, err
	}
	if err := requireText("productId", doc.ProductID); err != nil {
		return nil, err
	}
	if err := requireText("productGroup", string(doc.ProductGroup)); err != nil {
		return nil, err
	}
	if err := requireText("tnved", doc.Tnved); err != nil {
		return nil, err
	}
	if doc.Quantity < 1 {
		return nil, invalidArgument("quantity", "quantity must be positive")
	}
	if err := requireCodes("codes", doc.Codes, MaxReportCodes); err != nil {
		return nil, err
	}
	return c.sendDocument(ctx, pathCustomsRegister, doc)
}

func (c *Client) WithdrawCustomsCode(
	ctx context.Context, doc dto.CustomsCodeWithdrawalDocument,
) (*dto.DocumentIDResponse, error) {
	// POST /public/api/v1/doc/customs-code-withdrawal with base64 document and signature.
	// Works only for a registered AIK that is not used by customs yet.
	if err := requireText("code", doc.Code); err != nil {
		return nil, err
	}
	if doc.BusinessDatetime.IsZero() {
		return nil, invalidArgument("businessDatetime", "businessDatetime is required")
	}
	return c.sendDocument(ctx, pathCustomsWithdraw, doc)
}

// CalcAdditionalUnitQuantity counts goods in the additional unit for an AIK (doc 7.2.2).
func CalcAdditionalUnitQuantity(
	unit dto.AdditionalUnit, perPackage float64, codes int,
) (float64, error) {
	// perPackage: pieces (796, 798) or liters (112) in one consumer package.
	if codes < 0 || perPackage < 0 {
		return 0, invalidArgument("additionalUnit", "values must not be negative")
	}
	var result float64
	switch unit {
	case dto.AdditionalUnitPiece, dto.AdditionalUnitLiter:
		result = perPackage * float64(codes)
	case dto.AdditionalUnitThousand:
		result = perPackage * float64(codes) / 1000
	default:
		return 0, invalidArgument("additionalUnit", "unknown additional unit %d", unit)
	}
	return math.Round(result*1e6) / 1e6, nil
}
