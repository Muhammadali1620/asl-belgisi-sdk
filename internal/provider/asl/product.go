package asl

import (
	"context"
	"strings"

	"asl/internal/dto"
)

// Product registry paths.
const (
	pathProducts     = "/public/api/v1/product-registry/product"
	pathProductGTIN  = "/public/api/v1/product-registry/product/search-by-gtin"
	pathProductFiles = "/public/api/v1/product-registry/file/"
)

func (c *Client) ListProducts(
	ctx context.Context, filter dto.ProductFilter,
) ([]dto.ProductSummary, error) {
	// GET /public/api/v1/product-registry/product. Only published and blocked cards are returned.
	// Without other filters ASL returns all cards of the product group.
	if err := requireText("productGroup", string(filter.ProductGroup)); err != nil {
		return nil, err
	}
	if err := checkDateRange("updatedAt", filter.UpdatedAtFrom, filter.UpdatedAtTo); err != nil {
		return nil, err
	}
	values := query{}.
		set("productGroup", string(filter.ProductGroup)).
		set("inn", filter.Inn).
		set("gtin", filter.GTIN).
		set("status", string(filter.Status)).
		set("tnved", filter.Tnved).
		setTime("updatedAtFrom", filter.UpdatedAtFrom).
		setTime("updatedAtTo", filter.UpdatedAtTo).
		set("packageType", string(filter.PackageType))
	var out []dto.ProductSummary
	if err := c.get(ctx, pathProducts, values, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SearchProductsByGTIN(
	ctx context.Context, filter dto.SearchByGTINFilter,
) (*dto.SearchByGTINResponse, error) {
	// GET /public/api/v1/product-registry/product/search-by-gtin. Use Limit/Offset for pages.
	if err := requireGTIN("gtin", filter.GTIN); err != nil {
		return nil, err
	}
	values := query{}.
		set("gtin", filter.GTIN).
		set("productGroup", string(filter.ProductGroup)).
		set("status", string(filter.Status)).
		setInt("limit", filter.Limit).
		setInt("offset", filter.Offset)
	var out dto.SearchByGTINResponse
	if err := c.get(ctx, pathProductGTIN, values, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetProduct(ctx context.Context, productID string) (*dto.ProductCard, error) {
	// GET /public/api/v1/product-registry/product/{id}. The card has all attributes, for example
	// nested_products_count (group capacity) and expiration_date (shelf life).
	id, err := pathID("productId", productID)
	if err != nil {
		return nil, err
	}
	var out dto.ProductCard
	if err := c.get(ctx, pathProducts+"/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetProductByGTIN(ctx context.Context, gtin string) (*dto.ProductCard, error) {
	// Finds the card ID by GTIN, then loads the full card. Returns NOT_FOUND when no card exists.
	// When many cards have this GTIN, the published one is preferred.
	found, err := c.SearchProductsByGTIN(ctx, dto.SearchByGTINFilter{GTIN: gtin})
	if err != nil {
		return nil, err
	}
	if len(found.Products) == 0 {
		return nil, notFound("gtin", "product card with gtin "+gtin+" is not found", pathProductGTIN)
	}
	chosen := found.Products[0]
	for _, product := range found.Products {
		if strings.EqualFold(product.Status.Value, string(dto.ProductCardPublished)) {
			chosen = product
			break
		}
	}
	return c.GetProduct(ctx, chosen.ID)
}

func (c *Client) GetProductFile(ctx context.Context, fileID string) ([]byte, error) {
	// GET /public/api/v1/product-registry/file/{id}. The ID is the fileName from a FILE_LIST
	// attribute of the card. Returns raw file bytes (often an image).
	id, err := pathID("fileId", fileID)
	if err != nil {
		return nil, err
	}
	return c.download(ctx, pathProductFiles+id)
}
