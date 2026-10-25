package test

import (
	"testing"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

func TestLiveProducts(t *testing.T) {
	client := live(t)
	products := func(filter dto.ProductFilter) func(e env) ([]dto.ProductSummary, error) {
		return func(e env) ([]dto.ProductSummary, error) { return e.asl.ListProducts(e.ctx, filter) }
	}
	search := func(filter dto.SearchByGTINFilter) func(e env) (*dto.SearchByGTINResponse, error) {
		return func(e env) (*dto.SearchByGTINResponse, error) {
			return e.asl.SearchProductsByGTIN(e.ctx, filter)
		}
	}
	cardByGTIN := func(gtin string) func(e env) (*dto.ProductCard, error) {
		return func(e env) (*dto.ProductCard, error) { return e.asl.GetProductByGTIN(e.ctx, gtin) }
	}
	runCases(t, client, []testCase{
		pass("list cards of the group",
			products(dto.ProductFilter{ProductGroup: cfg.ProductGroup, Inn: cfg.TIN}),
			func(t *testing.T, got []dto.ProductSummary) {
				found := false
				for _, card := range got {
					found = found || card.GTIN == cfg.GTIN
				}
				if !found {
					t.Errorf("our GTIN %s is not in %d cards", cfg.GTIN, len(got))
				}
			}),
		pass("list cards by gtin and package",
			products(dto.ProductFilter{ProductGroup: cfg.ProductGroup, GTIN: cfg.GroupGTIN,
				PackageType: dto.PackageGroup, Status: dto.ProductCardPublished}),
			func(t *testing.T, got []dto.ProductSummary) {
				if len(got) != 1 || got[0].PackageType.Code != string(dto.PackageGroup) {
					t.Errorf("want one GROUP card, got %+v", got)
				}
			}),
		pass("list cards of other group is empty",
			products(dto.ProductFilter{ProductGroup: dto.ProductGroupPharma})),
		fail("list cards with bad status", func(e env) (any, error) {
			return e.asl.ListProducts(e.ctx, dto.ProductFilter{ProductGroup: cfg.ProductGroup, Status: "FOO"})
		}, asl.CodeInvalidInput),
		pass("search by gtin",
			search(dto.SearchByGTINFilter{GTIN: cfg.GTIN, Limit: 5}),
			func(t *testing.T, got *dto.SearchByGTINResponse) {
				if len(got.Products) == 0 || got.Products[0].ProductGroup.Value != string(cfg.ProductGroup) {
					t.Errorf("wrong search result: %+v", got)
				}
			}),
		pass("search unknown gtin is empty",
			search(dto.SearchByGTINFilter{GTIN: "04600000000006"}),
			func(t *testing.T, got *dto.SearchByGTINResponse) {
				if len(got.Products) != 0 || got.HasNextPage {
					t.Errorf("unknown gtin must give nothing: %+v", got)
				}
			}),
		pass("group card has capacity and inner gtin", cardByGTIN(cfg.GroupGTIN),
			func(t *testing.T, got *dto.ProductCard) {
				capacity, okCapacity := got.IntAttribute(dto.AttrNestedProductsCount)
				inner, okInner := got.StringAttribute(dto.AttrNestedProduct)
				if !okCapacity || capacity < 1 || !okInner || inner != cfg.GTIN || got.Scope == "" {
					t.Errorf("capacity %d, inner %q, scope %q", capacity, inner, got.Scope)
				}
				t.Logf("group capacity from ASL card: %d", capacity)
			}),
		pass("unit card has shelf life", cardByGTIN(cfg.GTIN),
			func(t *testing.T, got *dto.ProductCard) {
				shelf, ok := got.Attribute(dto.AttrExpirationDate)
				months, isInt := shelf.Int()
				if !ok || !isInt || months < 1 || shelf.Unit == nil {
					t.Fatalf("shelf life is missing: %+v", shelf)
				}
				t.Logf("shelf life from ASL card: %d %s", months, shelf.Unit.Code)
			}),
		fail("card by unknown gtin", cardByGTIN("04600000000006"), asl.CodeNotFound),
		fail("card by unknown id",
			func(e env) (any, error) { return e.asl.GetProduct(e.ctx, zeroUUID) },
			asl.CodeNotFound),
		fail("card by bad id",
			func(e env) (any, error) { return e.asl.GetProduct(e.ctx, "abc") },
			asl.CodeInvalidInput),
		fail("file unknown",
			func(e env) (any, error) { return e.asl.GetProductFile(e.ctx, zeroUUID) },
			asl.CodeNotFound),
		fail("file id with extension",
			func(e env) (any, error) { return e.asl.GetProductFile(e.ctx, zeroUUID+".jpg") },
			asl.CodeInvalidInput),
	})
	t.Run("download card files", func(t *testing.T) {
		// Some stage files are missing in storage (404); at least one file must download.
		ctx := testContext(t, liveTimeout)
		downloaded := 0
		for _, gtin := range []string{cfg.GTIN, cfg.GroupGTIN} {
			card, err := client.GetProductByGTIN(ctx, gtin)
			requireOK(t, err)
			for _, file := range card.AllFiles() {
				data, err := client.GetProductFile(ctx, file.ID())
				if err != nil {
					requireCode(t, err, asl.CodeNotFound)
					continue
				}
				if len(data) == 0 {
					t.Errorf("file %s is empty", file.FileName)
				}
				downloaded++
			}
		}
		if downloaded == 0 {
			t.Error("no card file could be downloaded")
		}
		t.Logf("downloaded %d files", downloaded)
	})
}
