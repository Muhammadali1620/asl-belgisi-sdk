package test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

func TestUnitConfig(t *testing.T) {
	for _, base := range []string{"", "   ", "xtrace.aslbelgisi.uz", "ftp://x.uz", "https://"} {
		_, err := asl.New(asl.Config{BaseURL: base})
		requireCode(t, err, asl.CodeInvalidArgument)
	}
	client, err := asl.New(asl.Config{BaseURL: asl.ProdURL + "/", Token: "  key  "})
	requireOK(t, err)
	if client.Token() != "key" {
		t.Errorf("token must be trimmed, got %q", client.Token())
	}
}

func TestUnitHTTP2AndHeaders(t *testing.T) {
	mock := newMock(t).on(http.MethodGet, "/api/orders", reply{Body: `{"orderInfos":[]}`})
	client := mock.client(t, func(c *asl.Config) { c.UserAgent = "my-app/2" })
	_, err := client.ListOrders(t.Context(), dto.OrderFilter{Status: dto.OrderStatusReady, Limit: 5})
	requireOK(t, err)
	got := mock.last(t)
	if got.Proto != 2 {
		t.Errorf("request must use HTTP/2, got HTTP/%d", got.Proto)
	}
	checks := map[string]string{
		"Authorization": "Bearer test-token",
		"Accept":        "application/json",
		"User-Agent":    "my-app/2",
	}
	for header, want := range checks {
		if value := got.Header.Get(header); value != want {
			t.Errorf("%s = %q, want %q", header, value, want)
		}
	}
	if got.Header.Get("Content-Type") != "" || len(got.Body) != 0 {
		t.Error("GET must have no body and no Content-Type")
	}
	if got.Query.Get("status") != "READY" || got.Query.Get("limit") != "5" || got.Query.Has("orderId") {
		t.Errorf("wrong query: %v", got.Query)
	}
}

func TestUnitDefaultTransportUsesHTTP2(t *testing.T) {
	mock := newMock(t).on(http.MethodGet, "/public/api/v1/party/parties/312458894/status",
		reply{Body: `{"tin":"312458894"}`})
	// Our own transport is used; only the test certificate is added to it.
	transport := asl.NewTransport()
	testTransport := mock.server.Client().Transport.(*http.Transport)
	transport.TLSClientConfig.RootCAs = testTransport.TLSClientConfig.RootCAs
	client := mock.client(t, func(c *asl.Config) { c.HTTPClient = &http.Client{Transport: transport} })
	_, err := client.GetPartyStatus(t.Context(), "312458894")
	requireOK(t, err)
	if mock.last(t).Proto != 2 {
		t.Errorf("default transport must use HTTP/2, got HTTP/%d", mock.last(t).Proto)
	}
}

func TestUnitSetToken(t *testing.T) {
	mock := newMock(t).on(http.MethodGet, "/api/orders", reply{Body: `{"orderInfos":[]}`})
	client := mock.client(t)
	var group sync.WaitGroup
	for index := range 20 {
		group.Go(func() {
			if index%2 == 0 {
				client.SetToken("new-token")
				return
			}
			_, _ = client.ListOrders(t.Context(), dto.OrderFilter{})
		})
	}
	group.Wait()
	client.SetToken(" new-token ")
	_, err := client.ListOrders(t.Context(), dto.OrderFilter{})
	requireOK(t, err)
	if auth := mock.last(t).Header.Get("Authorization"); auth != "Bearer new-token" {
		t.Errorf("Authorization = %q", auth)
	}
	client.SetToken("")
	_, err = client.ListOrders(t.Context(), dto.OrderFilter{})
	requireOK(t, err)
	if mock.last(t).Header.Get("Authorization") != "" {
		t.Error("empty token must not send Authorization")
	}
}

func TestUnitRetry(t *testing.T) {
	t.Run("GET is retried on 503 then works", func(t *testing.T) {
		mock := newMock(t).on(http.MethodGet, "/api/orders", reply{Next: func(call int) reply {
			if call < 3 {
				return reply{Status: http.StatusServiceUnavailable}
			}
			return reply{Body: `{"orderInfos":[{"orderId":"1"}]}`}
		}})
		list, err := mock.client(t).ListOrders(t.Context(), dto.OrderFilter{})
		requireOK(t, err)
		if len(list.OrderInfos) != 1 || len(mock.requests()) != 3 {
			t.Errorf("want 3 tries and 1 order, got %d tries", len(mock.requests()))
		}
	})
	t.Run("GET gives up after MaxRetries", func(t *testing.T) {
		mock := newMock(t).on(http.MethodGet, "/api/orders", reply{Status: http.StatusTooManyRequests})
		client := mock.client(t, func(c *asl.Config) { c.MaxRetries = 1 })
		_, err := client.ListOrders(t.Context(), dto.OrderFilter{})
		requireCode(t, err, asl.CodeRateLimited)
		if len(mock.requests()) != 2 {
			t.Errorf("want 2 tries, got %d", len(mock.requests()))
		}
	})
	t.Run("Retry-After is used", func(t *testing.T) {
		mock := newMock(t).on(http.MethodGet, "/api/orders", reply{Next: func(call int) reply {
			if call == 1 {
				header := map[string]string{"Retry-After": "1"}
				return reply{Status: http.StatusTooManyRequests, Header: header}
			}
			return reply{Body: `{"orderInfos":[]}`}
		}})
		start := time.Now()
		_, err := mock.client(t).ListOrders(t.Context(), dto.OrderFilter{})
		requireOK(t, err)
		if time.Since(start) < 900*time.Millisecond {
			t.Errorf("Retry-After: 1 must wait about 1s, waited %s", time.Since(start))
		}
	})
	t.Run("POST is never retried", func(t *testing.T) {
		mock := newMock(t).on(http.MethodPost, "/api/orders", reply{Status: http.StatusServiceUnavailable})
		_, err := mock.client(t).CreateOrder(t.Context(), validOrder())
		requireCode(t, err, asl.CodeUnavailable)
		if len(mock.requests()) != 1 {
			t.Errorf("POST must be sent once, got %d", len(mock.requests()))
		}
	})
	t.Run("not retryable error is not retried", func(t *testing.T) {
		mock := newMock(t).on(http.MethodGet, "/api/orders", reply{Status: http.StatusBadRequest})
		_, err := mock.client(t).ListOrders(t.Context(), dto.OrderFilter{})
		requireCode(t, err, asl.CodeInvalidInput)
		if len(mock.requests()) != 1 {
			t.Errorf("400 must not be retried, got %d tries", len(mock.requests()))
		}
	})
}

func TestUnitTimeoutCancelNetwork(t *testing.T) {
	t.Run("timeout of one try", func(t *testing.T) {
		mock := newMock(t).on(http.MethodGet, "/api/orders", reply{Delay: time.Second, Body: `{}`})
		client := mock.client(t, func(c *asl.Config) { c.Timeout = 50 * time.Millisecond; c.MaxRetries = -1 })
		_, err := client.ListOrders(t.Context(), dto.OrderFilter{})
		aslErr := requireCode(t, err, asl.CodeTimeout)
		if aslErr.HTTPStatus != 0 || !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("timeout must wrap context.DeadlineExceeded and have no status: %+v", aslErr)
		}
	})
	t.Run("canceled context", func(t *testing.T) {
		mock := newMock(t).on(http.MethodGet, "/api/orders", reply{Delay: time.Second, Body: `{}`})
		ctx, cancel := context.WithCancel(t.Context())
		time.AfterFunc(30*time.Millisecond, cancel)
		_, err := mock.client(t).ListOrders(ctx, dto.OrderFilter{})
		requireCode(t, err, asl.CodeCanceled)
	})
	t.Run("server is down", func(t *testing.T) {
		mock := newMock(t)
		client := mock.client(t)
		mock.server.Close()
		_, err := client.ListOrders(t.Context(), dto.OrderFilter{})
		requireCode(t, err, asl.CodeNetwork)
	})
	t.Run("wait methods stop with context", func(t *testing.T) {
		mock := newMock(t).on(http.MethodGet, "/api/orders/sub-orders", reply{Body: `{"subOrderInfos":[
			{"parentOrderId":"o1","gtin":"04605246003585","bufferStatus":"PENDING"}]}`})
		ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
		defer cancel()
		_, err := mock.client(t).WaitSubOrder(ctx, "o1", "04605246003585", 20*time.Millisecond)
		requireCode(t, err, asl.CodeTimeout)
		if len(mock.requests()) < 3 {
			t.Errorf("wait must poll many times, got %d", len(mock.requests()))
		}
	})
}

func TestUnitDecodeError(t *testing.T) {
	mock := newMock(t).on(http.MethodGet, "/api/orders", reply{Body: `{"orderInfos": "not a list"}`})
	_, err := mock.client(t).ListOrders(t.Context(), dto.OrderFilter{})
	aslErr := requireCode(t, err, asl.CodeDecode)
	if aslErr.HTTPStatus != http.StatusOK || len(aslErr.Body) == 0 {
		t.Errorf("decode error must keep status and body: %+v", aslErr)
	}
}

func TestUnitPrivateCodesRedirect(t *testing.T) {
	mock := newMock(t).
		on(http.MethodPost, "/public/api/cod/private/codes", reply{Status: http.StatusTemporaryRedirect,
			Header: map[string]string{"Location": "/public/api/cod/public/codes"},
			Body:   `[{"code":"access-denied","service":"cod-reader"}]`}).
		on(http.MethodPost, "/public/api/cod/public/codes", reply{Body: `[{
		"code":"0104605246003585217H3Je/ekvCq8M",
			"packageType":"UNIT","status":"INTRODUCED","issuerShortInfo":{"issuerTin":"307797292"}}]`})
	codes := []string{"0104605246003585217H3Je/ekvCq8M"}
	history := true
	result, err := mock.client(t).GetPrivateCodes(t.Context(), dto.PrivateCodesRequest{Codes: codes,
		AddCodeHistory: &history})
	requireOK(t, err)
	if !result.Redirected || len(result.PublicResults) != 1 || len(result.ForbiddenCodes) != 1 {
		t.Fatalf("redirect fallback is wrong: %+v", result)
	}
	var publicBody map[string]any
	_ = json.Unmarshal(mock.last(t).Body, &publicBody)
	if publicBody["addCodeHistory"] != true {
		t.Errorf("history flag must go to the public request, body %s", mock.last(t).Body)
	}
}

func TestUnitDocumentEnvelope(t *testing.T) {
	var signed []byte
	signer := asl.SignerFunc(func(_ context.Context, body []byte) (string, error) {
		signed = body
		return "SIGNATURE-1", nil
	})
	mock := newMock(t).on(http.MethodPost, "/public/api/v1/doc/correction",
		reply{Body: `{"documentId":"d1"}`})
	client := mock.client(t, func(c *asl.Config) { c.Signer = signer })
	when := time.Date(2026, 7, 5, 4, 36, 29, 785_000_000, time.UTC)
	result, err := client.SendCorrection(t.Context(), dto.CorrectionDocument{
		BusinessDatetime: dto.NewDateTime(when),
		Codes:            []string{"0104605246003585217<b>&x"},
		UpdatedFields:    dto.UpdatedFields{SeriesNumber: "S-1", ManufacturerCountry: dto.CountryUZ},
	})
	requireOK(t, err)
	if result.DocumentID != "d1" {
		t.Errorf("DocumentID = %q", result.DocumentID)
	}
	var envelope dto.DocumentEnvelope
	requireOK(t, json.Unmarshal(mock.last(t).Body, &envelope))
	raw, err := base64.StdEncoding.DecodeString(envelope.DocumentBody)
	requireOK(t, err)
	want := `{"businessDatetime":"2026-07-05T04:36:29.785Z","codes":["0104605246003585217<b>&x"],` +
		`"updatedFields":{"manufacturerCountry":"UZ","seriesNumber":"S-1"}}`
	if string(raw) != want {
		t.Errorf("document JSON must have A-Z keys and no HTML escape:\n got %s\nwant %s", raw, want)
	}
	if envelope.Signature != "SIGNATURE-1" || string(signed) != want {
		t.Errorf("signer must get the same JSON, signature %q", envelope.Signature)
	}
}

func TestUnitSignerError(t *testing.T) {
	mock := newMock(t)
	signer := asl.SignerFunc(func(context.Context, []byte) (string, error) {
		return "", errors.New("no token")
	})
	client := mock.client(t, func(c *asl.Config) { c.Signer = signer })
	_, err := client.WithdrawCustomsCode(t.Context(), dto.CustomsCodeWithdrawalDocument{
		BusinessDatetime: dto.NewDateTime(time.Now()), Code: "0000031245889425102650000",
	})
	requireCode(t, err, asl.CodeSignature)
	if len(mock.requests()) != 0 {
		t.Error("request must not be sent when signing fails")
	}
}

func TestUnitUnsignedDocumentHasNoSignature(t *testing.T) {
	mock := newMock(t).on(http.MethodPost, "/public/api/v1/doc/aggregation",
		reply{Body: `{"documentId":"d2"}`})
	_, err := mock.client(t).SendAggregation(t.Context(), validAggregation())
	requireOK(t, err)
	if strings.Contains(string(mock.last(t).Body), "signature") {
		t.Errorf("signature must be omitted without signer: %s", mock.last(t).Body)
	}
}

func TestUnitFormBodyForTokenRefresh(t *testing.T) {
	mock := newMock(t).on(http.MethodPost, "/api/users/tokens/refresh", reply{Body: `{"accessToken":"a",
		"accessTokenType":"BEARER","accessTokenExpiresIn":1800000,"refreshToken":"r"}`})
	tokens, err := mock.client(t).RefreshTechToken(t.Context(), " refresh-1 ")
	requireOK(t, err)
	got := mock.last(t)
	if got.Header.Get("Content-Type") != "application/x-www-form-urlencoded" ||
		string(got.Body) != "refreshToken=refresh-1" || got.Header.Get("Authorization") != "" {
		t.Errorf("wrong refresh request: %s %v", got.Body, got.Header)
	}
	if tokens.AccessToken != "a" || tokens.AccessTokenExpiresIn != 1800000 {
		t.Errorf("wrong tokens: %+v", tokens)
	}
}

func TestUnitBinaryDownload(t *testing.T) {
	mock := newMock(t).on(http.MethodGet, "/public/api/v1/product-registry/file/f1",
		reply{Body: "\x89PNG-bytes", Header: map[string]string{"Content-Type": "image/png"}})
	data, err := mock.client(t).GetProductFile(t.Context(), "f1")
	requireOK(t, err)
	if string(data) != "\x89PNG-bytes" || mock.last(t).Header.Get("Accept") != "*/*" {
		t.Errorf("wrong download: %q", data)
	}
}

func TestUnitPathEscape(t *testing.T) {
	mock := newMock(t)
	_, err := mock.client(t).GetDocument(t.Context(), "../../api/orders?x=1")
	requireCode(t, err, asl.CodeNotFound)
	if got := mock.last(t).Path; !strings.HasPrefix(got, "/public/api/v1/doc/storage/docs/") {
		t.Errorf("ID must not change the URL path, got %q", got)
	}
}

// validOrder returns a correct order request for mock tests.
func validOrder() dto.CreateOrderRequest {
	return dto.CreateOrderRequest{
		ProductGroup: dto.ProductGroupFertilizers, BusinessPlaceID: 734,
		ReleaseMethodType: dto.ReleaseMethodPrimary,
		Products: []dto.OrderProduct{{GTIN: "04605246003585", Quantity: 1, CisType: dto.PackageUnit,
			SerialNumberType: dto.SerialNumberOperator}},
	}
}

// validAggregation returns a correct aggregation document for mock tests.
func validAggregation() dto.AggregationDocument {
	return dto.AggregationDocument{
		BusinessPlaceID: 734, DocumentDate: dto.NewDateTime(time.Now()),
		AggregationUnits: []dto.AggregationUnit{{UnitSerialNumber: stageSSCC, AggregationUnitCapacity: 2,
			AggregationItemsCount: 1, Codes: []string{"0104605246003585217H3Je/ekvCq8M"}}},
	}
}
