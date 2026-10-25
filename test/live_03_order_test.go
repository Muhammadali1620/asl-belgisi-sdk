package test

import (
	"context"
	"slices"
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// liveOrder returns an order request for the test product; change edits it.
func liveOrder(change func(*dto.CreateOrderRequest)) dto.CreateOrderRequest {
	req := dto.CreateOrderRequest{
		ProductGroup:      cfg.ProductGroup,
		BusinessPlaceID:   cfg.BusinessPlaceID,
		ReleaseMethodType: dto.ReleaseMethodPrimary,
		Products: []dto.OrderProduct{{
			GTIN: cfg.GTIN, Quantity: 1, CisType: dto.PackageUnit,
			SerialNumberType: dto.SerialNumberOperator,
		}},
	}
	if change != nil {
		change(&req)
	}
	return req
}

func TestLiveOrders(t *testing.T) {
	client := live(t)
	create := func(change func(*dto.CreateOrderRequest)) func(e env) (any, error) {
		return func(e env) (any, error) { return e.asl.CreateOrder(e.ctx, liveOrder(change)) }
	}
	orders := func(filter dto.OrderFilter) func(e env) (*dto.OrderList, error) {
		return func(e env) (*dto.OrderList, error) { return e.asl.ListOrders(e.ctx, filter) }
	}
	codes := func(req dto.GetCodesRequest) func(e env) (any, error) {
		return func(e env) (any, error) { return e.asl.GetCodes(e.ctx, req) }
	}
	runCases(t, client, []testCase{
		fail("create with unknown business place",
			create(func(r *dto.CreateOrderRequest) { r.BusinessPlaceID = 999999 }),
			asl.CodeInvalidInput),
		fail("create in not own product group",
			create(func(r *dto.CreateOrderRequest) { r.ProductGroup = dto.ProductGroupPharma }),
			asl.CodeInvalidInput),
		fail("create with unknown enums",
			create(func(r *dto.CreateOrderRequest) {
				r.ReleaseMethodType, r.Products[0].CisType = "FOO", "BAR"
			}),
			asl.CodeInvalidInput),
		fail("create with 11 products",
			create(func(r *dto.CreateOrderRequest) { r.Products = slices.Repeat(r.Products, 11) }),
			asl.CodeInvalidArgument),
		pass("list orders",
			orders(dto.OrderFilter{ProductGroup: cfg.ProductGroup, Limit: 3}),
			func(t *testing.T, got *dto.OrderList) {
				if len(got.OrderInfos) == 0 || len(got.OrderInfos) > 3 {
					t.Errorf("want 1..3 orders, got %d", len(got.OrderInfos))
				}
			}),
		pass("list orders by date and status", orders(dto.OrderFilter{
			DateFrom: dto.NewDateTimePtr(time.Now().AddDate(0, 0, -30)),
			Status:   dto.OrderStatusClosed,
			Limit:    2,
		})),
		fail("list orders with bad id", orders(dto.OrderFilter{OrderID: "abc"}), asl.CodeInvalidInput),
		fail("list orders with bad status", orders(dto.OrderFilter{Status: "FOO"}), asl.CodeInvalidInput),
		fail("get unknown order",
			func(e env) (any, error) { return e.asl.GetOrder(e.ctx, zeroUUID) },
			asl.CodeNotFound),
		fail("list sub-orders with bad status", func(e env) (any, error) {
			return e.asl.ListSubOrders(e.ctx, dto.SubOrderFilter{Status: "FOO"})
		}, asl.CodeInvalidInput),
		fail("get unknown sub-order",
			func(e env) (any, error) { return e.asl.GetSubOrder(e.ctx, zeroUUID, cfg.GTIN) },
			asl.CodeNotFound),
		fail("codes of unknown order",
			codes(dto.GetCodesRequest{OrderID: zeroUUID, GTIN: cfg.GTIN, Quantity: 1}),
			asl.CodeNotFound),
		fail("codes with bad order id",
			codes(dto.GetCodesRequest{OrderID: "abc", GTIN: cfg.GTIN, Quantity: 1}),
			asl.CodeInvalidInput),
		pass("packs of unknown order are empty",
			func(e env) (*dto.CodePackList, error) { return e.asl.ListCodePacks(e.ctx, zeroUUID, cfg.GTIN) },
			func(t *testing.T, got *dto.CodePackList) {
				if len(got.Packs) != 0 {
					t.Errorf("want no packs, got %d", len(got.Packs))
				}
			}),
		fail("packs with bad order id",
			func(e env) (any, error) { return e.asl.ListCodePacks(e.ctx, "abc", cfg.GTIN) },
			asl.CodeInvalidInput),
		fail("close unknown order",
			func(e env) (any, error) { return e.asl.CloseOrder(e.ctx, zeroUUID, "") },
			asl.CodeNotFound),
		fail("close with bad order id",
			func(e env) (any, error) { return e.asl.CloseOrder(e.ctx, "abc", "") },
			asl.CodeInvalidInput),
		fail("utilisation status of unknown report",
			func(e env) (any, error) { return e.asl.GetUtilisationStatus(e.ctx, zeroUUID) },
			asl.CodeNotFound),
		fail("utilisation status with bad id",
			func(e env) (any, error) { return e.asl.GetUtilisationStatus(e.ctx, "abc") },
			asl.CodeInvalidInput),
	})
	t.Run("bad token", func(t *testing.T) {
		badClient := liveWithToken(t, zeroUUID)
		_, err := badClient.CreateOrder(testContext(t, liveTimeout), liveOrder(nil))
		requireCode(t, err, asl.CodeUnauthorized)
	})
	t.Run("order with unknown gtin is rejected", func(t *testing.T) {
		req := liveOrder(func(r *dto.CreateOrderRequest) { r.Products[0].GTIN = "04600000000006" })
		requireRejected(t, client, req, asl.CodeNotFound, "321")
	})
	t.Run("order with wrong package type is rejected", func(t *testing.T) {
		req := liveOrder(func(r *dto.CreateOrderRequest) { r.Products[0].CisType = dto.PackageGroup })
		requireRejected(t, client, req, asl.CodeInvalidInput, "311")
	})
}

// requireRejected creates an order that ASL must reject and checks the error of WaitSubOrder.
func requireRejected(
	t *testing.T, c *asl.Client, req dto.CreateOrderRequest, code asl.ErrorCode, aslCode string,
) {
	t.Helper()
	ctx := testContext(t, liveTimeout)
	order, err := c.CreateOrder(ctx, req)
	requireOK(t, err)
	sub, err := c.WaitSubOrder(ctx, order.OrderID, req.Products[0].GTIN, time.Second)
	aslErr := requireCode(t, err, code)
	if sub == nil || sub.BufferStatus != dto.BufferStatusRejected || aslErr.ASLCode != aslCode {
		t.Errorf("want REJECTED with ASL code %s, got %+v / %s", aslCode, sub, aslErr.ASLCode)
	}
	info, err := c.GetOrder(ctx, order.OrderID)
	requireOK(t, err)
	if info.OrderStatus != dto.OrderStatusRejected {
		t.Errorf("order status = %s, want REJECTED", info.OrderStatus)
	}
}

// TestLiveOrderPacks downloads codes in packs, checks pack rules and closes the sub-order.
func TestLiveOrderPacks(t *testing.T) {
	client := live(t)
	ctx := testContext(t, liveTimeout)
	req := liveOrder(func(r *dto.CreateOrderRequest) { r.Products[0].Quantity = 3 })
	order, err := client.CreateOrder(ctx, req)
	requireOK(t, err)
	t.Cleanup(func() { _, _ = client.CloseOrder(context.Background(), order.OrderID, "") })
	next := func(lastPackID string, quantity int) dto.GetCodesRequest {
		return dto.GetCodesRequest{OrderID: order.OrderID, GTIN: cfg.GTIN, Quantity: quantity,
			LastPackID: lastPackID}
	}

	sub, err := client.WaitSubOrder(ctx, order.OrderID, cfg.GTIN, time.Second)
	requireOK(t, err)
	if sub.BufferStatus != dto.BufferStatusActive || sub.AvailableCodes != 3 || sub.LeftInBuffer != 3 {
		t.Fatalf("sub-order must be ACTIVE with 3 codes: %+v", sub)
	}
	first, err := client.GetCodes(ctx, next("", 1))
	requireOK(t, err)
	second, err := client.GetCodes(ctx, next(first.PackID, 1))
	requireOK(t, err)
	if len(first.Codes) != 1 || len(second.Codes) != 1 || first.Codes[0] == second.Codes[0] {
		t.Fatalf("next pack must have new codes: %q / %q", first.Codes, second.Codes)
	}
	for _, code := range append(first.Codes, second.Codes...) {
		parsed, err := asl.ParseCode(code)
		requireOK(t, err)
		if parsed.GTIN != cfg.GTIN || parsed.CheckCode == "" {
			t.Errorf("downloaded code is not a full code: %+v", parsed)
		}
	}
	again, err := client.GetCodes(ctx, next("", 1))
	requireOK(t, err)
	if again.PackID != first.PackID || again.Codes[0] != first.Codes[0] {
		t.Errorf("without lastPackId ASL must give the first pack again: %+v", again)
	}
	packs, err := client.ListCodePacks(ctx, order.OrderID, cfg.GTIN)
	requireOK(t, err)
	if len(packs.Packs) != 2 || packs.Packs[0].PackDateTime.IsZero() {
		t.Errorf("want 2 packs, got %+v", packs.Packs)
	}
	_, err = client.GetCodes(ctx, next(second.PackID, 5))
	requireCode(t, err, asl.CodeInvalidInput) // ASL 603: quantity is more than codes left

	closed, err := client.CloseOrder(ctx, order.OrderID, cfg.GTIN)
	requireOK(t, err)
	if closed.OrderID != order.OrderID {
		t.Errorf("closed order = %+v", closed)
	}
	sub, err = client.GetSubOrder(ctx, order.OrderID, cfg.GTIN)
	requireOK(t, err)
	if sub.BufferStatus != dto.BufferStatusClosed {
		t.Errorf("sub-order status = %s, want CLOSED", sub.BufferStatus)
	}
	_, err = client.CloseOrder(ctx, order.OrderID, "")
	requireCode(t, err, asl.CodeConflict)
}

// TestLiveOrderFixture checks the order made by the shared fixture.
func TestLiveOrderFixture(t *testing.T) {
	codes := fxCodes(t)
	runCases(t, live(t), []testCase{
		pass("get order",
			func(e env) (*dto.OrderInfo, error) { return e.asl.GetOrder(e.ctx, codes.OrderID) },
			func(t *testing.T, got *dto.OrderInfo) {
				if got.OrderStatus != dto.OrderStatusClosed || got.PoNumber != "go-provider-test" {
					t.Errorf("order with all codes downloaded must be CLOSED: %+v", got)
				}
			}),
		pass("unit sub-order is exhausted",
			func(e env) (*dto.SubOrderInfo, error) {
				return e.asl.GetSubOrder(e.ctx, codes.OrderID, cfg.GTIN)
			},
			func(t *testing.T, got *dto.SubOrderInfo) {
				if got.LeftInBuffer != 0 || got.TotalPassed != len(codes.Units) ||
					got.CisType != dto.PackageUnit {
					t.Errorf("wrong sub-order: %+v", got)
				}
			}),
		pass("list sub-orders of order",
			func(e env) (*dto.SubOrderList, error) {
				return e.asl.ListSubOrders(e.ctx, dto.SubOrderFilter{OrderID: codes.OrderID})
			},
			func(t *testing.T, got *dto.SubOrderList) {
				if len(got.SubOrderInfos) != 2 {
					t.Errorf("want 2 sub-orders (unit and group), got %d", len(got.SubOrderInfos))
				}
			}),
		fail("codes of exhausted sub-order", func(e env) (any, error) {
			req := dto.GetCodesRequest{OrderID: codes.OrderID, GTIN: cfg.GTIN, Quantity: 1,
				LastPackID: zeroUUID}
			return e.asl.GetCodes(e.ctx, req)
		}, asl.CodeInvalidInput), // ASL 603: quantity is more than codes left
		fail("close closed order",
			func(e env) (any, error) { return e.asl.CloseOrder(e.ctx, codes.OrderID, "") },
			asl.CodeConflict),
	})
}
