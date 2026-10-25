package test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// TestUnitValidation checks that bad input gives INVALID_ARGUMENT and no request goes to ASL.
func TestUnitValidation(t *testing.T) {
	now := dto.NewDateTime(time.Now())
	later := dto.NewDateTimePtr(time.Now().Add(time.Hour))
	earlier := dto.NewDateTimePtr(time.Now().Add(-time.Hour))
	order := func(change func(*dto.CreateOrderRequest)) func(e env) (any, error) {
		return func(e env) (any, error) {
			req := validOrder()
			change(&req)
			return e.asl.CreateOrder(e.ctx, req)
		}
	}
	utilisation := func(change func(*dto.UtilisationRequest)) func(e env) (any, error) {
		return func(e env) (any, error) {
			req := dto.UtilisationRequest{ProductGroup: dto.ProductGroupFertilizers,
				Sntins: []string{stageUnitCode}, BusinessPlaceID: 734,
				ReleaseType: dto.ReleaseProduction, ManufacturerCountry: dto.CountryUZ}
			change(&req)
			return e.asl.SendUtilisation(e.ctx, req)
		}
	}
	type aggregationChange func(*dto.AggregationDocument, *dto.AggregationUnit)
	aggregation := func(change aggregationChange) func(e env) (any, error) {
		return func(e env) (any, error) {
			doc := validAggregation()
			change(&doc, &doc.AggregationUnits[0])
			return e.asl.SendAggregation(e.ctx, doc)
		}
	}
	tooManyUnits := func(d *dto.AggregationDocument, _ *dto.AggregationUnit) {
		unit := dto.AggregationUnit{UnitSerialNumber: stageSSCC, AggregationUnitCapacity: 10_001,
			AggregationItemsCount: 10_001, Codes: slices.Repeat([]string{"c"}, 10_001)}
		d.AggregationUnits = []dto.AggregationUnit{unit, unit, unit}
	}
	validation := func(codes ...dto.ValidationCode) func(e env) (any, error) {
		return func(e env) (any, error) {
			return e.asl.SendValidation(e.ctx, dto.ValidationDocument{
				ProductGroup:    dto.ProductGroupFertilizers,
				BusinessPlaceID: 734, PackageType: dto.PackageUnit, DocumentDate: now, Codes: codes})
		}
	}
	public := func(codes ...string) func(e env) (any, error) {
		return func(e env) (any, error) {
			return e.asl.GetPublicCodes(e.ctx, dto.PublicCodesRequest{Codes: codes})
		}
	}
	cases := map[string]func(e env) (any, error){
		"check key: tin with letters": func(e env) (any, error) {
			return e.asl.CheckAPIKey(e.ctx, "31245889A")
		},
		"check key: tin 10 digits": func(e env) (any, error) {
			return e.asl.CheckAPIKey(e.ctx, "3124588941")
		},
		"refresh key: no key and no id": func(e env) (any, error) {
			return e.asl.RefreshAPIKey(e.ctx, "312458894", dto.APIKeyRefreshRequest{})
		},
		"refresh key: key and id": func(e env) (any, error) {
			return e.asl.RefreshAPIKey(e.ctx, "312458894", dto.APIKeyRefreshRequest{APIKey: "a", ID: "b"})
		},
		"auth: empty login": func(e env) (any, error) {
			return e.asl.AuthenticateTechUser(e.ctx, dto.TechAuthRequest{Password: "p"})
		},
		"auth: empty password": func(e env) (any, error) {
			return e.asl.AuthenticateTechUser(e.ctx, dto.TechAuthRequest{Login: "l"})
		},
		"refresh token: empty": func(e env) (any, error) { return e.asl.RefreshTechToken(e.ctx, " ") },

		"order: no products": order(func(r *dto.CreateOrderRequest) { r.Products = nil }),
		"order: 11 products": order(func(r *dto.CreateOrderRequest) {
			r.Products = make([]dto.OrderProduct, 11)
		}),
		"order: no group":  order(func(r *dto.CreateOrderRequest) { r.ProductGroup = "" }),
		"order: no method": order(func(r *dto.CreateOrderRequest) { r.ReleaseMethodType = "" }),
		"order: short gtin": order(func(r *dto.CreateOrderRequest) {
			r.Products[0].GTIN = "4605246003585"
		}),
		"order: 0 codes":      order(func(r *dto.CreateOrderRequest) { r.Products[0].Quantity = 0 }),
		"order: 150001 codes": order(func(r *dto.CreateOrderRequest) { r.Products[0].Quantity = 150_001 }),
		"order: no cis type":  order(func(r *dto.CreateOrderRequest) { r.Products[0].CisType = "" }),
		"order: no serial":    order(func(r *dto.CreateOrderRequest) { r.Products[0].SerialNumberType = "" }),
		"order: self made without serials": order(func(r *dto.CreateOrderRequest) {
			r.Products[0].SerialNumberType = dto.SerialNumberSelfMade
		}),
		"order: operator with serials": order(func(r *dto.CreateOrderRequest) {
			r.Products[0].SerialNumbers = []string{"x"}
		}),
		"order: bad contractor tin": order(func(r *dto.CreateOrderRequest) {
			r.ContractorInfo = &dto.ContractorInfo{ContractorTin: "1", ContractorCountryCode: dto.CountryUZ}
		}),
		"orders: dateTo before dateFrom": func(e env) (any, error) {
			return e.asl.ListOrders(e.ctx, dto.OrderFilter{DateFrom: later, DateTo: earlier})
		},
		"sub orders: dateTo before dateFrom": func(e env) (any, error) {
			return e.asl.ListSubOrders(e.ctx, dto.SubOrderFilter{DateFrom: later, DateTo: earlier})
		},
		"get order: empty id": func(e env) (any, error) { return e.asl.GetOrder(e.ctx, " ") },
		"get sub order: bad gtin": func(e env) (any, error) {
			return e.asl.GetSubOrder(e.ctx, testUUID, "123")
		},
		"codes: no order": func(e env) (any, error) {
			return e.asl.GetCodes(e.ctx, dto.GetCodesRequest{GTIN: "04605246003585", Quantity: 1})
		},
		"codes: zero quantity": func(e env) (any, error) {
			return e.asl.GetCodes(e.ctx, dto.GetCodesRequest{OrderID: testUUID, GTIN: "04605246003585"})
		},
		"packs: bad gtin":   func(e env) (any, error) { return e.asl.ListCodePacks(e.ctx, testUUID, "x") },
		"close: empty id":   func(e env) (any, error) { return e.asl.CloseOrder(e.ctx, "", "") },
		"close: bad gtin":   func(e env) (any, error) { return e.asl.CloseOrder(e.ctx, testUUID, "12") },
		"report: empty id":  func(e env) (any, error) { return e.asl.GetUtilisationStatus(e.ctx, "") },
		"wait doc: empty":   func(e env) (any, error) { return e.asl.WaitDocument(e.ctx, "", 0) },
		"wait export: none": func(e env) (any, error) { return e.asl.WaitExport(e.ctx, "", 0) },

		"utilisation: no codes":   utilisation(func(r *dto.UtilisationRequest) { r.Sntins = nil }),
		"utilisation: empty code": utilisation(func(r *dto.UtilisationRequest) { r.Sntins = []string{" "} }),
		"utilisation: bad utf8": utilisation(func(r *dto.UtilisationRequest) {
			r.Sntins = []string{"\xff"}
		}),
		"utilisation: no group":   utilisation(func(r *dto.UtilisationRequest) { r.ProductGroup = "" }),
		"utilisation: no place":   utilisation(func(r *dto.UtilisationRequest) { r.BusinessPlaceID = 0 }),
		"utilisation: no release": utilisation(func(r *dto.UtilisationRequest) { r.ReleaseType = "" }),
		"utilisation: no country": utilisation(func(r *dto.UtilisationRequest) {
			r.ManufacturerCountry = ""
		}),
		"utilisation: circulation": utilisation(func(r *dto.UtilisationRequest) {
			r.ReleaseType = dto.ReleaseCirculation
		}),
		"utilisation: 30001 codes": utilisation(func(r *dto.UtilisationRequest) {
			r.Sntins = slices.Repeat([]string{"c"}, 30_001)
		}),
		"utilisation: long series": utilisation(func(r *dto.UtilisationRequest) {
			r.SeriesNumber = strings.Repeat("s", 21)
		}),

		"aggregation: no units": aggregation(func(d *dto.AggregationDocument, _ *dto.AggregationUnit) {
			d.AggregationUnits = nil
		}),
		"aggregation: no place": aggregation(func(d *dto.AggregationDocument, _ *dto.AggregationUnit) {
			d.BusinessPlaceID = 0
		}),
		"aggregation: no date": aggregation(func(d *dto.AggregationDocument, _ *dto.AggregationUnit) {
			d.DocumentDate = dto.DateTime{}
		}),
		"aggregation: no parent": aggregation(func(_ *dto.AggregationDocument, u *dto.AggregationUnit) {
			u.UnitSerialNumber = ""
		}),
		"aggregation: no children": aggregation(func(_ *dto.AggregationDocument, u *dto.AggregationUnit) {
			u.Codes = nil
		}),
		"aggregation: over capacity": aggregation(func(_ *dto.AggregationDocument, u *dto.AggregationUnit) {
			u.AggregationItemsCount = 3
		}),
		"aggregation: zero count": aggregation(func(_ *dto.AggregationDocument, u *dto.AggregationUnit) {
			u.AggregationItemsCount = 0
		}),
		"aggregation: 30003 codes in total": aggregation(tooManyUnits),

		"validation: no codes":   validation(),
		"validation: no quality": validation(dto.ValidationCode{Code: "c"}),
		"correction: no fields": func(e env) (any, error) {
			doc := dto.CorrectionDocument{BusinessDatetime: now, Codes: []string{"c"}}
			return e.asl.SendCorrection(e.ctx, doc)
		},
		"correction: no date": func(e env) (any, error) {
			return e.asl.SendCorrection(e.ctx, dto.CorrectionDocument{Codes: []string{"c"},
				UpdatedFields: dto.UpdatedFields{SeriesNumber: "s"}})
		},
		"disaggregation: no codes": func(e env) (any, error) {
			return e.asl.SendDisaggregation(e.ctx, dto.DisaggregationDocument{BusinessDatetime: now})
		},
		"transfer: no buyer": func(e env) (any, error) {
			doc := dto.TransferRequestDocument{OwnerID: 1, BusinessDatetime: now}
			return e.asl.SendTransferRequest(e.ctx, doc)
		},
		"transfer: zero quantity": func(e env) (any, error) {
			doc := dto.TransferRequestDocument{BuyerID: 1, OwnerID: 2, BusinessDatetime: now,
				Products: []dto.TransferProduct{{GTIN: "04605246003585"}}, Codes: []string{"c"}}
			return e.asl.SendTransferRequest(e.ctx, doc)
		},
		"transfer confirm: bad resolution": func(e env) (any, error) {
			doc := dto.TransferConfirmationDocument{TransferRequestDocID: testUUID, Resolution: "MAYBE"}
			return e.asl.SendTransferConfirmation(e.ctx, doc)
		},
		"transfer search: bad business dates": func(e env) (any, error) {
			filter := dto.TransferSearchFilter{BusinessDateFrom: later, BusinessDateTo: earlier}
			return e.asl.SearchTransferRequests(e.ctx, filter)
		},
		"customs: no code": func(e env) (any, error) {
			return e.asl.RegisterCustomsCode(e.ctx, dto.CustomsCodeDocument{GTIN: "04605246003585"})
		},
		"customs withdraw: no date": func(e env) (any, error) {
			return e.asl.WithdrawCustomsCode(e.ctx, dto.CustomsCodeWithdrawalDocument{Code: "a"})
		},
		"export: no gtin": func(e env) (any, error) { return e.asl.CreateExport(e.ctx, dto.ExportRequest{}) },
		"export: bad dates": func(e env) (any, error) {
			req := dto.ExportRequest{GTIN: "04605246003585", EmissionDateFrom: later, EmissionDateTo: earlier}
			return e.asl.CreateExport(e.ctx, req)
		},
		"export status: empty id": func(e env) (any, error) { return e.asl.GetExportStatus(e.ctx, "") },

		"public codes: none":      public(),
		"public codes: 1001":      public(slices.Repeat([]string{stageSSCC}, 1001)...),
		"public codes: too short": public("0104605246003585"),
		"public codes: cyrillic":  public("0104605246003585217Привет12"),
		"private codes: none": func(e env) (any, error) {
			return e.asl.GetPrivateCodes(e.ctx, dto.PrivateCodesRequest{})
		},
		"owner check: 101 codes": func(e env) (any, error) {
			codes := slices.Repeat([]string{stageSSCC}, 101)
			req := dto.OwnerCheckRequest{Codes: codes, OwnerTin: "312458894"}
			return e.asl.CheckCodeOwner(e.ctx, req)
		},
		"owner check: bad tin": func(e env) (any, error) {
			return e.asl.CheckCodeOwner(e.ctx, dto.OwnerCheckRequest{Codes: []string{stageSSCC}})
		},
		"verify: none": func(e env) (any, error) { return e.asl.VerifyCodes(e.ctx, nil) },

		"products: no group": func(e env) (any, error) {
			return e.asl.ListProducts(e.ctx, dto.ProductFilter{})
		},
		"products: bad dates": func(e env) (any, error) {
			filter := dto.ProductFilter{ProductGroup: dto.ProductGroupFertilizers,
				UpdatedAtFrom: later, UpdatedAtTo: earlier}
			return e.asl.ListProducts(e.ctx, filter)
		},
		"search by gtin: bad gtin": func(e env) (any, error) {
			return e.asl.SearchProductsByGTIN(e.ctx, dto.SearchByGTINFilter{GTIN: "abc"})
		},
		"product: empty id":      func(e env) (any, error) { return e.asl.GetProduct(e.ctx, "") },
		"product by gtin: bad":   func(e env) (any, error) { return e.asl.GetProductByGTIN(e.ctx, "1") },
		"product file: empty id": func(e env) (any, error) { return e.asl.GetProductFile(e.ctx, "") },

		"documents: receipt type": func(e env) (any, error) {
			filter := dto.DocumentFilter{Types: []dto.DocumentType{dto.DocumentTypeSalesReceipt}}
			return e.asl.SearchDocuments(e.ctx, filter)
		},
		"documents: bad dates": func(e env) (any, error) {
			return e.asl.SearchDocuments(e.ctx, dto.DocumentFilter{DateFrom: later, DateTo: earlier})
		},
		"document: empty id":      func(e env) (any, error) { return e.asl.GetDocument(e.ctx, "") },
		"document json: empty id": func(e env) (any, error) { return e.asl.GetDocumentJSON(e.ctx, "") },
		"document errors: empty id": func(e env) (any, error) {
			return e.asl.GetDocumentErrors(e.ctx, "", dto.DocumentErrorsFilter{})
		},
		"document codes: empty id": func(e env) (any, error) {
			return e.asl.GetDocumentCodes(e.ctx, "", dto.DocumentCodesFilter{})
		},
		"party: short tin": func(e env) (any, error) { return e.asl.GetPartyStatus(e.ctx, "123") },
	}
	mock := newMock(t)
	client := mock.client(t)
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := call(env{ctx: t.Context(), asl: client})
			requireCode(t, err, asl.CodeInvalidArgument)
		})
	}
	if calls := mock.requests(); len(calls) != 0 {
		t.Errorf("no request must reach ASL, got %d (first %s %s)",
			len(calls), calls[0].Method, calls[0].Path)
	}
}

func TestUnitCalcAdditionalUnit(t *testing.T) {
	cases := []struct {
		unit       dto.AdditionalUnit
		perPackage float64
		codes      int
		want       float64
	}{
		{dto.AdditionalUnitPiece, 12, 10, 120},
		{dto.AdditionalUnitThousand, 20, 50, 1},
		{dto.AdditionalUnitLiter, 0.5, 3, 1.5},
		{dto.AdditionalUnitLiter, 0.001, 1, 0.001},
	}
	for _, item := range cases {
		got, err := asl.CalcAdditionalUnitQuantity(item.unit, item.perPackage, item.codes)
		requireOK(t, err)
		if got != item.want {
			t.Errorf("unit %d: got %v, want %v", item.unit, got, item.want)
		}
	}
	_, err := asl.CalcAdditionalUnitQuantity(999, 1, 1)
	requireCode(t, err, asl.CodeInvalidArgument)
	_, err = asl.CalcAdditionalUnitQuantity(dto.AdditionalUnitPiece, -1, 1)
	requireCode(t, err, asl.CodeInvalidArgument)
}
