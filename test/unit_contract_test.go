package test

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

const testUUID = "145dab62-fa73-4798-54c2-ec00c4783d0f"

// contractCase checks that one provider method sends the right request and reads the answer.
type contractCase struct {
	name   string
	method string
	path   string
	query  url.Values // expected query (exact); nil means "no query"
	body   string     // expected JSON body; for documents it is the decoded documentBody
	answer string     // mock answer body
	call   func(e env) (any, error)
	check  func(*testing.T, any) // optional check of the decoded result
}

func TestUnitContract(t *testing.T) {
	when := dto.NewDateTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	whenText := "2026-01-02T03:04:05.000Z"
	yes := true
	index := 0
	unitID := asl.IdentityCode(stageUnitCode)
	cases := []contractCase{
		{
			name: "CheckAPIKey", method: "GET",
			path:   "/public/api/v1/party/parties/312458894/api-keys/check",
			answer: `{"isTinCorrect":true,"expiresOn":"2026-11-22T23:59:59Z"}`,
			call:   func(e env) (any, error) { return e.asl.CheckAPIKey(e.ctx, "312458894") },
			check: func(t *testing.T, v any) {
				got := v.(*dto.APIKeyCheckResponse)
				if !got.IsTinCorrect || got.ExpiresOn.Year() != 2026 {
					t.Errorf("wrong result %+v", got)
				}
			},
		},
		{
			name: "RefreshAPIKey", method: "POST",
			path:   "/public/api/v1/party/parties/312458894/api-keys/refresh",
			body:   `{"id":"` + testUUID + `"}`,
			answer: `{"apiKey":"k2","id":"i2","expiresOn":"2025-11-30T15:51:28.071Z","label":"l"}`,
			call: func(e env) (any, error) {
				return e.asl.RefreshAPIKey(e.ctx, "312458894", dto.APIKeyRefreshRequest{ID: testUUID})
			},
		},
		{
			name: "AuthenticateTechUser", method: "POST", path: "/api/users/authenticate",
			body: `{"login":"u","password":"p"}`,
			answer: `{"accessToken":"a","accessTokenType":"BEARER","accessTokenExpiresIn":1800000,` +
				`"refreshToken":"r"}`,
			call: func(e env) (any, error) {
				return e.asl.AuthenticateTechUser(e.ctx, dto.TechAuthRequest{Login: "u", Password: "p"})
			},
		},
		{
			name: "CreateOrder", method: "POST", path: "/api/orders",
			body: `{"productGroup":"fertilizers","businessPlaceId":734,"releaseMethodType":"PRIMARY",` +
				`"products":[{"gtin":"04605246003585","quantity":1,"cisType":"UNIT",` +
				`"serialNumberType":"OPERATOR"}]}`,
			answer: `{"orderId":"` + testUUID + `"}`,
			call:   func(e env) (any, error) { return e.asl.CreateOrder(e.ctx, validOrder()) },
		},
		{
			name: "ListOrders", method: "GET", path: "/api/orders",
			query: url.Values{"orderId": {testUUID}, "dateFrom": {whenText}, "cursor": {"c1"}},
			answer: `{"orderInfos":[{"orderId":"o1","orderStatus":"READY",` +
				`"createDate":"2025-01-30T11:02:02.492Z"}]}`,
			call: func(e env) (any, error) {
				filter := dto.OrderFilter{OrderID: testUUID, DateFrom: &when, Cursor: "c1"}
				return e.asl.ListOrders(e.ctx, filter)
			},
		},
		{
			name: "ListSubOrders", method: "GET", path: "/api/orders/sub-orders",
			query:  url.Values{"orderId": {testUUID}, "cisType": {"GROUP"}, "status": {"ACTIVE"}},
			answer: `{"subOrderInfos":[]}`,
			call: func(e env) (any, error) {
				filter := dto.SubOrderFilter{OrderID: testUUID, CisType: dto.PackageGroup,
					Status: dto.BufferStatusActive}
				return e.asl.ListSubOrders(e.ctx, filter)
			},
		},
		{
			name: "GetCodes", method: "GET", path: "/api/codes",
			query: url.Values{"orderId": {testUUID}, "gtin": {"04605246003585"}, "quantity": {"5"},
				"lastPackId": {"p1"}},
			answer: `{"packId":"p2","codes":["0104605246003585217H3Je/ekvCq8M\u001d93At65"]}`,
			call: func(e env) (any, error) {
				req := dto.GetCodesRequest{OrderID: testUUID, GTIN: "04605246003585", Quantity: 5,
					LastPackID: "p1"}
				return e.asl.GetCodes(e.ctx, req)
			},
			check: func(t *testing.T, v any) {
				if got := v.(*dto.CodesPack); got.Codes[0] != stageUnitCode {
					t.Errorf("GS must be decoded as ASCII 29: %q", got.Codes[0])
				}
			},
		},
		{
			name: "ListCodePacks", method: "GET", path: "/api/codes/packs",
			query: url.Values{"orderId": {testUUID}, "gtin": {"04605246003585"}},
			answer: `{"orderId":"o","gtin":"g","packs":[{"packId":"p","quantity":2,` +
				`"packDateTime":"2025-01-15T09:49:18.834Z"}]}`,
			call: func(e env) (any, error) { return e.asl.ListCodePacks(e.ctx, testUUID, "04605246003585") },
		},
		{
			name: "CloseOrder", method: "POST", path: "/api/order/close",
			query:  url.Values{"orderId": {testUUID}, "gtin": {"04605246003585"}},
			answer: `{"orderId":"o","gtin":"g"}`,
			call:   func(e env) (any, error) { return e.asl.CloseOrder(e.ctx, testUUID, "04605246003585") },
		},
		{
			name: "GetUtilisationStatus", method: "GET", path: "/api/utilisation/" + testUUID,
			answer: `{"reportId":"r","reportStatus":"SUCCESS","createdTimestamp":"2026-08-18T03:09:06.965Z"}`,
			call:   func(e env) (any, error) { return e.asl.GetUtilisationStatus(e.ctx, testUUID) },
		},
		{
			name: "SendUtilisation", method: "POST", path: "/api/utilisation",
			query: url.Values{"productGroup": {"fertilizers"}},
			body: `{"sntins":["0104605246003585217H3Je/ekvCq8M\u001d93At65"],"businessPlaceId":734,` +
				`"releaseType":"PRODUCTION","manufacturerCountry":"UZ",` +
				`"productionDate":"` + whenText + `","seriesNumber":"S1"}`,
			answer: `{"reportId":"r1"}`,
			call: func(e env) (any, error) {
				return e.asl.SendUtilisation(e.ctx, dto.UtilisationRequest{
					ProductGroup: dto.ProductGroupFertilizers, Sntins: []string{stageUnitCode},
					BusinessPlaceID: 734, ReleaseType: dto.ReleaseProduction,
					ManufacturerCountry: dto.CountryUZ, ProductionDate: &when, SeriesNumber: "S1",
				})
			},
		},
		{
			name: "SendValidation", method: "POST", path: "/public/api/v1/doc/validation",
			body: `{"businessPlaceId":734,"codes":[{"code":"c1","printQualityClass":"A"}],` +
				`"documentDate":"` + whenText + `","packageType":"UNIT","productGroup":"fertilizers"}`,
			answer: `{"documentId":"d"}`,
			call: func(e env) (any, error) {
				return e.asl.SendValidation(e.ctx, dto.ValidationDocument{
					ProductGroup: dto.ProductGroupFertilizers, BusinessPlaceID: 734,
					PackageType: dto.PackageUnit, DocumentDate: when,
					Codes: []dto.ValidationCode{{Code: "c1", PrintQualityClass: dto.PrintQualityA}},
				})
			},
		},
		{
			name: "SendAggregation", method: "POST", path: "/public/api/v1/doc/aggregation",
			body: `{"aggregationUnits":[{"aggregationItemsCount":1,"aggregationUnitCapacity":12,` +
				`"codes":["c1"],"shouldBeUnbundled":false,"unitSerialNumber":"` + stageSSCC + `"}],` +
				`"businessPlaceId":734,"documentDate":"` + whenText + `","productionOrderId":"p1"}`,
			answer: `{"documentId":"d"}`,
			call: func(e env) (any, error) {
				unit := dto.AggregationUnit{UnitSerialNumber: stageSSCC, AggregationUnitCapacity: 12,
					AggregationItemsCount: 1, Codes: []string{"c1"}}
				return e.asl.SendAggregation(e.ctx, dto.AggregationDocument{BusinessPlaceID: 734,
					DocumentDate: when, ProductionOrderID: "p1",
					AggregationUnits: []dto.AggregationUnit{unit}})
			},
		},
		{
			name: "SendCorrection", method: "POST", path: "/public/api/v1/doc/correction",
			body: `{"businessDatetime":"` + whenText + `","codes":["c1"],` +
				`"updatedFields":{"expirationDatetime":"` + whenText + `"}}`,
			answer: `{"documentId":"d"}`,
			call: func(e env) (any, error) {
				return e.asl.SendCorrection(e.ctx, dto.CorrectionDocument{BusinessDatetime: when,
					Codes: []string{"c1"}, UpdatedFields: dto.UpdatedFields{ExpirationDatetime: &when}})
			},
		},
		{
			name: "SendDisaggregation", method: "POST",
			path:   "/public/api/v1/doc/transport-code-disaggregation",
			body:   `{"businessDatetime":"` + whenText + `","codes":["` + stageSSCC + `"]}`,
			answer: `{"documentId":"d"}`,
			call: func(e env) (any, error) {
				doc := dto.DisaggregationDocument{BusinessDatetime: when, Codes: []string{stageSSCC}}
				return e.asl.SendDisaggregation(e.ctx, doc)
			},
		},
		{
			name: "SearchTransferRequests", method: "GET",
			path: "/public/api/v1/doc/transfer/request/search",
			query: url.Values{"confirmationStatuses": {"AWAITING_CONFIRMATION", "REJECTED"},
				"buyerId": {"8"}, "limit": {"3"}},
			answer: `{"documentInfos":[]}`,
			call: func(e env) (any, error) {
				statuses := []dto.ConfirmationStatus{dto.ConfirmationAwaiting, dto.ConfirmationRejected}
				filter := dto.TransferSearchFilter{BuyerID: 8, Limit: 3, ConfirmationStatuses: statuses}
				return e.asl.SearchTransferRequests(e.ctx, filter)
			},
		},
		{
			name: "SendTransferRequest", method: "POST", path: "/public/api/v1/doc/transfer/request",
			body: `{"businessDatetime":"` + whenText + `","buyerId":8,"codes":["c1"],"ownerId":3,` +
				`"products":[{"gtin":"04605246003585","quantity":1}]}`,
			answer: `{"documentId":"d"}`,
			call: func(e env) (any, error) {
				return e.asl.SendTransferRequest(e.ctx, dto.TransferRequestDocument{
					BuyerID: 8, OwnerID: 3, BusinessDatetime: when, Codes: []string{"c1"},
					Products: []dto.TransferProduct{{GTIN: "04605246003585", Quantity: 1}},
				})
			},
		},
		{
			name: "SendTransferConfirmation", method: "POST",
			path:   "/public/api/v1/doc/transfer/confirmation",
			body:   `{"resolution":"REJECTED","transferRequestDocId":"` + testUUID + `"}`,
			answer: `{"documentId":"d"}`,
			call: func(e env) (any, error) {
				doc := dto.TransferConfirmationDocument{TransferRequestDocID: testUUID,
					Resolution: dto.ResolutionRejected}
				return e.asl.SendTransferConfirmation(e.ctx, doc)
			},
		},
		{
			name: "GenerateCustomsCode", method: "POST", path: "/public/api/v1/customs-registry/code",
			answer: `{"aggregatedCustomsCode":"0000031245889425102650000"}`,
			call:   func(e env) (any, error) { return e.asl.GenerateCustomsCode(e.ctx) },
		},
		{
			name: "RegisterCustomsCode", method: "POST", path: "/public/api/v1/doc/customs-code",
			body: `{"additionalUnit":null,"additionalUnitQuantity":null,"aggregatedCustomsCode":"a1",` +
				`"codes":["c1"],"gtin":"04605246003585","productGroup":"fertilizers","productId":"p1",` +
				`"quantity":1,"tnved":"2841300000"}`,
			answer: `{"documentId":"d"}`,
			call: func(e env) (any, error) {
				return e.asl.RegisterCustomsCode(e.ctx, dto.CustomsCodeDocument{
					AggregatedCustomsCode: "a1", GTIN: "04605246003585", ProductID: "p1",
					ProductGroup: dto.ProductGroupFertilizers, Tnved: "2841300000", Quantity: 1,
					Codes: []string{"c1"},
				})
			},
		},
		{
			name: "WithdrawCustomsCode", method: "POST", path: "/public/api/v1/doc/customs-code-withdrawal",
			body:   `{"businessDatetime":"` + whenText + `","code":"a1"}`,
			answer: `{"documentId":"d"}`,
			call: func(e env) (any, error) {
				doc := dto.CustomsCodeWithdrawalDocument{BusinessDatetime: when, Code: "a1"}
				return e.asl.WithdrawCustomsCode(e.ctx, doc)
			},
		},
		{
			name: "CreateExport", method: "POST", path: "/public/api/cod/exports",
			body: `{"gtin":"04605246003585","packageType":["UNIT"],"status":["INTRODUCED"],` +
				`"emptyPackage":true}`,
			answer: `{"id":"e1"}`,
			call: func(e env) (any, error) {
				return e.asl.CreateExport(e.ctx, dto.ExportRequest{
					GTIN: "04605246003585", EmptyPackage: &yes,
					PackageType: []dto.PackageType{dto.PackageUnit},
					Status:      []dto.CodeStatus{dto.CodeStatusIntroduced},
				})
			},
		},
		{
			name: "GetExportStatus", method: "GET", path: "/public/api/cod/exports/" + testUUID + "/status",
			answer: `"SUCCESS"`,
			call:   func(e env) (any, error) { return e.asl.GetExportStatus(e.ctx, testUUID) },
			check: func(t *testing.T, v any) {
				if v.(dto.ExportStatus) != dto.ExportStatusSuccess {
					t.Errorf("status = %v", v)
				}
			},
		},
		{
			name: "GetPublicCodes", method: "POST", path: "/public/api/cod/public/codes",
			body: `{"codes":["` + unitID + `"],"addCodeHistory":true}`,
			answer: `[{"code":"c","packageType":"UNIT","status":"APPLIED","isHadExtendedCode":false,` +
				`"issuerShortInfo":{"issuerTin":"t"}}]`,
			call: func(e env) (any, error) {
				req := dto.PublicCodesRequest{Codes: []string{unitID}, AddCodeHistory: &yes}
				return e.asl.GetPublicCodes(e.ctx, req)
			},
		},
		{
			name: "GetPrivateCodes", method: "POST", path: "/public/api/cod/private/codes",
			body:   `{"codes":["` + stageSSCC + `"],"addChildCodes":true}`,
			answer: privateAnswer,
			call: func(e env) (any, error) {
				req := dto.PrivateCodesRequest{Codes: []string{stageSSCC}, AddChildCodes: &yes}
				return e.asl.GetPrivateCodes(e.ctx, req)
			},
			check: func(t *testing.T, v any) {
				got := v.(*dto.PrivateCodesResponse)
				result := got.Results[0]
				event := result.CodeHistory[0].EventDescription.UpdateChildren
				if got.Redirected || result.PackageData.ActuallyPacked != 1 ||
					result.TurnoverData.OwnerInfo.OwnerTin != "312458894" || event.ActuallyPacked != 1 {
					t.Errorf("private answer is not decoded well: %+v", got)
				}
			},
		},
		{
			name: "CheckCodeOwner", method: "POST", path: "/public/api/cod/nested-codes/owner-check",
			body:   `{"codes":["` + stageSSCC + `"],"ownerTin":"312458894"}`,
			answer: `{"results":[],"forbiddenCodes":[],"missingCodes":["` + stageSSCC + `"]}`,
			call: func(e env) (any, error) {
				req := dto.OwnerCheckRequest{Codes: []string{stageSSCC}, OwnerTin: "312458894"}
				return e.asl.CheckCodeOwner(e.ctx, req)
			},
		},
		{
			name: "VerifyCodes", method: "POST", path: "/public/api/v1/code-verification/verify",
			body: `["0104605246003585217H3Je/ekvCq8M\u001d93At65"]`,
			answer: `[{"code":"c","verified":true,"productGroup":53},` +
				`{"code":"d","verified":false,"productGroup":null}]`,
			call: func(e env) (any, error) { return e.asl.VerifyCodes(e.ctx, []string{stageUnitCode}) },
		},
		{
			name: "ListProducts", method: "GET", path: "/public/api/v1/product-registry/product",
			query: url.Values{"productGroup": {"fertilizers"}, "gtin": {"04605246003585"},
				"status": {"PUBLISHED"}},
			answer: `[{"id":"p","gtin":"g","productGroup":{"code":"fertilizers"},` +
				`"status":{"code":"PUBLISHED"}}]`,
			call: func(e env) (any, error) {
				filter := dto.ProductFilter{ProductGroup: dto.ProductGroupFertilizers,
					GTIN: "04605246003585", Status: dto.ProductCardPublished}
				return e.asl.ListProducts(e.ctx, filter)
			},
		},
		{
			name: "SearchProductsByGTIN", method: "GET",
			path:   "/public/api/v1/product-registry/product/search-by-gtin",
			query:  url.Values{"gtin": {"04605246003585"}, "limit": {"10"}, "offset": {"20"}},
			answer: `{"products":[],"hasNextPage":false}`,
			call: func(e env) (any, error) {
				filter := dto.SearchByGTINFilter{GTIN: "04605246003585", Limit: 10, Offset: 20}
				return e.asl.SearchProductsByGTIN(e.ctx, filter)
			},
		},
		{
			name: "GetProduct", method: "GET", path: "/public/api/v1/product-registry/product/" + testUUID,
			answer: `{"id":"p","gtin":"g","scope":"GLOBAL","attributes":[{"meta":` +
				`{"code":"nested_products_count","type":"INTEGER"},"value":12}]}`,
			call: func(e env) (any, error) { return e.asl.GetProduct(e.ctx, testUUID) },
			check: func(t *testing.T, v any) {
				capacity, ok := v.(*dto.ProductCard).IntAttribute(dto.AttrNestedProductsCount)
				if !ok || capacity != 12 {
					t.Errorf("capacity = %d", capacity)
				}
			},
		},
		{
			name: "SearchDocuments", method: "GET", path: "/public/api/v1/doc/storage/docs/search",
			query: url.Values{"types": {"AGGREGATION", "UTILISATION"}, "status": {"ERROR"},
				"withWarning": {"true"}, "productGroups": {"fertilizers"}},
			answer: `{"documentInfos":[{"documentId":"d","type":"AGGREGATION","status":"ERROR",` +
				`"createDate":"2026-03-31T13:34:32.295Z"}]}`,
			call: func(e env) (any, error) {
				return e.asl.SearchDocuments(e.ctx, dto.DocumentFilter{
					WithWarning: &yes,
					Types: []dto.DocumentType{
						dto.DocumentTypeAggregation, dto.DocumentTypeUtilisation,
					},
					Statuses:      []dto.DocumentStatus{dto.DocumentStatusError},
					ProductGroups: []dto.ProductGroup{dto.ProductGroupFertilizers},
				})
			},
		},
		{
			name: "GetDocument", method: "GET", path: "/public/api/v1/doc/storage/docs/" + testUUID,
			answer: `{"documentId":"d","type":"UTILISATION","status":"SUCCESS",` +
				`"createDate":"2025-09-05T13:21:19.014Z","productGroup":null}`,
			call: func(e env) (any, error) { return e.asl.GetDocument(e.ctx, testUUID) },
		},
		{
			name: "GetDocumentJSON", method: "GET", path: "/public/api/v1/doc/storage/json/" + testUUID,
			answer: `{"sntins":["x"],"businessPlaceId":27}`,
			call:   func(e env) (any, error) { return e.asl.GetDocumentJSON(e.ctx, testUUID) },
			check: func(t *testing.T, v any) {
				if !strings.Contains(string(v.(json.RawMessage)), `"businessPlaceId":27`) {
					t.Errorf("raw JSON = %s", v)
				}
			},
		},
		{
			name: "GetDocumentErrors", method: "GET", path: "/public/api/v1/doc/storage/errors/" + testUUID,
			query: url.Values{"limit": {"5"}, "lastIndex": {"0"}, "propertyName": {"CODE"}},
			answer: `{"documentErrors":[{"propertyName":"CODE","index":0,` +
				`"errorCode":"invalid-code-status","errorTags":{"status":"INTRODUCED"}}]}`,
			call: func(e env) (any, error) {
				filter := dto.DocumentErrorsFilter{Limit: 5, LastIndex: &index, PropertyName: "CODE"}
				return e.asl.GetDocumentErrors(e.ctx, testUUID, filter)
			},
		},
		{
			name: "GetDocumentCodes", method: "GET",
			path:   "/public/api/v1/doc/storage/docs/" + testUUID + "/codes",
			query:  url.Values{"lastIndex": {"0"}},
			answer: `[{"index":1,"code":"c","state":"ERROR","result":"code-not-found"}]`,
			call: func(e env) (any, error) {
				return e.asl.GetDocumentCodes(e.ctx, testUUID, dto.DocumentCodesFilter{LastIndex: &index})
			},
		},
		{
			name: "GetPartyStatus", method: "GET", path: "/public/api/v1/party/parties/312458894/status",
			answer: `{"tin":"312458894","productGroups":["fertilizers"]}`,
			call:   func(e env) (any, error) { return e.asl.GetPartyStatus(e.ctx, "312458894") },
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			mock := newMock(t).on(item.method, item.path, reply{Body: item.answer})
			result, err := item.call(env{ctx: t.Context(), asl: mock.client(t)})
			requireOK(t, err)
			got := mock.last(t)
			if got.Method != item.method || got.Path != item.path || got.Proto != 2 {
				t.Fatalf("request = HTTP/%d %s %s, want %s %s",
					got.Proto, got.Method, got.Path, item.method, item.path)
			}
			if item.query != nil && got.Query.Encode() != item.query.Encode() {
				t.Errorf("query = %s\n want %s", got.Query.Encode(), item.query.Encode())
			}
			if item.query == nil && len(got.Query) != 0 {
				t.Errorf("unexpected query %s", got.Query.Encode())
			}
			if item.body != "" {
				requireSameJSON(t, requestJSON(t, got.Body), item.body)
			}
			if item.check != nil {
				item.check(t, result)
			}
		})
	}
}

// requestJSON returns the JSON of a request; for document envelopes it decodes documentBody.
func requestJSON(t *testing.T, body []byte) []byte {
	t.Helper()
	var envelope dto.DocumentEnvelope
	if json.Unmarshal(body, &envelope) == nil && envelope.DocumentBody != "" {
		raw, err := base64.StdEncoding.DecodeString(envelope.DocumentBody)
		requireOK(t, err)
		return raw
	}
	return body
}

// requireSameJSON compares two JSON texts without spaces and key order.
func requireSameJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var gotValue, wantValue any
	requireOK(t, json.Unmarshal(got, &gotValue))
	requireOK(t, json.Unmarshal([]byte(want), &wantValue))
	gotText, _ := json.Marshal(gotValue)
	wantText, _ := json.Marshal(wantValue)
	if string(gotText) != string(wantText) {
		t.Errorf("body = %s\n want %s", gotText, wantText)
	}
}

// privateAnswer is a private codes answer with nested data.
const privateAnswer = `{"results":[{` +
	`"codeData":{"code":"00031245889400000019","status":"INTRODUCED","template":"SSCC"},` +
	`"packageData":{"packageType":"BOX_LV_1","emptyPackage":false,` +
	`"children":[{"code":"g","status":"INTRODUCED","packageType":"GROUP"}],"actuallyPacked":1,` +
	`"aggregateProductGroups":[{"productGroupId":53,"unitsNumber":2}]},` +
	`"markingData":{"emissionDate":"2026-10-25T03:28:19.000Z",` +
	`"issuerInfo":{"issuerId":688,"issuerTin":"312458894"},"contractorInfo":{}},` +
	`"turnoverData":{"originalReleaseMethod":"CIRCULATION",` +
	`"ownerInfo":{"ownerId":688,"ownerTin":"312458894"}},` +
	`"codeHistory":[{"eventType":"UPDATE_CHILDREN","eventSourceId":"s","codeStateChanged":true,` +
	`"eventDescription":{"eventUpdateChildrenShortResponse":{"actuallyPacked":1,"actuallyRemoved":0}}}]` +
	`}],"forbiddenCodes":[]}`
