package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// Fixtures create real data on the stage server once per test run.
// Each fixture builds the fixtures it needs, so any single test can run alone.
// The chain is: codes -> applied codes -> group package -> box (SSCC) -> pallet (SSCC).

// fixtureTimeout is the time limit for building one fixture.
const fixtureTimeout = 5 * time.Minute

// pollInterval is how often fixtures ask ASL about statuses.
const pollInterval = 2 * time.Second

// Units in the fixture order: GroupSize go to the group, then spare codes.
const (
	spareApplied  = 1 // applied but not packed: used by correction
	spareReceived = 1 // not applied: used by validation and negative utilisation
)

// fixture runs build once and keeps its result or error for all tests.
type fixture[T any] struct {
	once  sync.Once
	value T
	err   error
}

// get returns the fixture value, or fails the test when building failed (now or before).
func (f *fixture[T]) get(t *testing.T, build func(ctx context.Context, c *asl.Client) (T, error)) T {
	t.Helper()
	client := live(t)
	f.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), fixtureTimeout)
		defer cancel()
		f.value, f.err = build(ctx, client)
	})
	if f.err != nil {
		t.Fatalf("fixture is not ready: %v", f.err)
	}
	return f.value
}

// orderedCodes is a closed order with downloaded codes.
type orderedCodes struct {
	OrderID   string
	GroupSize int      // capacity of the group package from the product card
	ShelfLife int      // shelf life in months from the product card
	Units     []string // full unit codes: GroupSize for the group + spare codes
	Group     string   // full group code
}

// Packed returns unit codes that go into the group package.
func (o orderedCodes) Packed() []string { return o.Units[:o.GroupSize] }

// SpareApplied returns unit codes that are applied but not packed.
func (o orderedCodes) SpareApplied() []string {
	return o.Units[o.GroupSize : o.GroupSize+spareApplied]
}

// SpareReceived returns unit codes that stay in RECEIVED status.
func (o orderedCodes) SpareReceived() []string { return o.Units[o.GroupSize+spareApplied:] }

var codesFixture fixture[orderedCodes]

// fxCodes orders unit and group codes, waits for them and downloads them.
func fxCodes(t *testing.T) orderedCodes {
	t.Helper()
	return codesFixture.get(t, func(ctx context.Context, c *asl.Client) (orderedCodes, error) {
		result := orderedCodes{}
		groupCard, err := c.GetProductByGTIN(ctx, cfg.GroupGTIN)
		if err != nil {
			return result, fmt.Errorf("group card: %w", err)
		}
		unitCard, err := c.GetProductByGTIN(ctx, cfg.GTIN)
		if err != nil {
			return result, fmt.Errorf("unit card: %w", err)
		}
		var ok bool
		if result.GroupSize, ok = groupCard.IntAttribute(dto.AttrNestedProductsCount); !ok {
			return result, fmt.Errorf("group card has no %s", dto.AttrNestedProductsCount)
		}
		if result.ShelfLife, ok = unitCard.IntAttribute(dto.AttrExpirationDate); !ok {
			result.ShelfLife = 24
		}
		unitCount := result.GroupSize + spareApplied + spareReceived
		order, err := c.CreateOrder(ctx, dto.CreateOrderRequest{
			ProductGroup: cfg.ProductGroup, BusinessPlaceID: cfg.BusinessPlaceID,
			ReleaseMethodType: dto.ReleaseMethodPrimary, PoNumber: "go-provider-test",
			Products: []dto.OrderProduct{
				{GTIN: cfg.GTIN, Quantity: unitCount, CisType: dto.PackageUnit,
					SerialNumberType: dto.SerialNumberOperator},
				{GTIN: cfg.GroupGTIN, Quantity: 1, CisType: dto.PackageGroup,
					SerialNumberType: dto.SerialNumberOperator},
			},
		})
		if err != nil {
			return result, fmt.Errorf("create order: %w", err)
		}
		result.OrderID = order.OrderID
		units, err := downloadCodes(ctx, c, order.OrderID, cfg.GTIN, unitCount)
		if err != nil {
			return result, err
		}
		groups, err := downloadCodes(ctx, c, order.OrderID, cfg.GroupGTIN, 1)
		if err != nil {
			return result, err
		}
		result.Units, result.Group = units, groups[0]
		_, _ = c.CloseOrder(ctx, order.OrderID, "") // ASL closes it itself when all codes are downloaded
		return result, nil
	})
}

// downloadCodes waits for a sub-order and downloads all its codes in one pack.
func downloadCodes(ctx context.Context, c *asl.Client, orderID, gtin string, count int) ([]string, error) {
	if _, err := c.WaitSubOrder(ctx, orderID, gtin, time.Second); err != nil {
		return nil, fmt.Errorf("wait sub-order %s: %w", gtin, err)
	}
	pack, err := c.GetCodes(ctx, dto.GetCodesRequest{OrderID: orderID, GTIN: gtin, Quantity: count})
	if err != nil {
		return nil, fmt.Errorf("get codes %s: %w", gtin, err)
	}
	if len(pack.Codes) != count {
		return nil, fmt.Errorf("got %d codes of %s, want %d", len(pack.Codes), gtin, count)
	}
	return pack.Codes, nil
}

// appliedCodes is the result of a successful utilisation report.
type appliedCodes struct {
	orderedCodes
	ReportID       string
	ProductionDate time.Time
}

var appliedFixture fixture[appliedCodes]

// fxApplied sends utilisation for packed units, the spare applied unit and the group code.
func fxApplied(t *testing.T) appliedCodes {
	t.Helper()
	codes := fxCodes(t)
	return appliedFixture.get(t, func(ctx context.Context, c *asl.Client) (appliedCodes, error) {
		produced := time.Now().UTC().Add(-time.Minute)
		sntins := append(append(append([]string{}, codes.Packed()...), codes.SpareApplied()...), codes.Group)
		report, err := c.SendUtilisation(ctx, utilisationRequest(sntins, produced, codes.ShelfLife))
		if err != nil {
			return appliedCodes{}, fmt.Errorf("utilisation: %w", err)
		}
		if _, err := c.WaitDocumentResult(ctx, report.ReportID, pollInterval); err != nil {
			return appliedCodes{}, fmt.Errorf("utilisation result: %w", err)
		}
		return appliedCodes{orderedCodes: codes, ReportID: report.ReportID, ProductionDate: produced}, nil
	})
}

// utilisationRequest builds a utilisation report for the test product.
func utilisationRequest(sntins []string, produced time.Time, shelfLifeMonths int) dto.UtilisationRequest {
	return dto.UtilisationRequest{
		ProductGroup: cfg.ProductGroup, Sntins: sntins, BusinessPlaceID: cfg.BusinessPlaceID,
		ReleaseType: dto.ReleaseProduction, ManufacturerCountry: dto.CountryUZ,
		ProductionDate: dto.NewDateTimePtr(produced),
		ExpirationDate: dto.NewDateTimePtr(produced.AddDate(0, shelfLifeMonths, 0)),
		SeriesNumber:   "GO-TEST-" + produced.Format("0102"),
	}
}

// packedGroup is a group package with applied units inside.
type packedGroup struct {
	appliedCodes
	GroupID       string // identity code of the group
	AggregationID string
}

var groupFixture fixture[packedGroup]

// fxGroup packs units into the group code.
func fxGroup(t *testing.T) packedGroup {
	t.Helper()
	applied := fxApplied(t)
	return groupFixture.get(t, func(ctx context.Context, c *asl.Client) (packedGroup, error) {
		groupID := asl.IdentityCode(applied.Group)
		children := asl.IdentityCodes(applied.Packed())
		docID, err := aggregate(ctx, c, groupID, applied.GroupSize, children)
		if err != nil {
			return packedGroup{}, fmt.Errorf("group aggregation: %w", err)
		}
		return packedGroup{appliedCodes: applied, GroupID: groupID, AggregationID: docID}, nil
	})
}

// aggregate sends one package and waits for the result.
func aggregate(
	ctx context.Context, c *asl.Client, parent string, capacity int, children []string,
) (string, error) {
	doc, err := c.SendAggregation(ctx, dto.AggregationDocument{
		BusinessPlaceID: cfg.BusinessPlaceID, DocumentDate: dto.NewDateTime(time.Now().UTC()),
		AggregationUnits: []dto.AggregationUnit{{UnitSerialNumber: parent, AggregationUnitCapacity: capacity,
			AggregationItemsCount: len(children), Codes: children}},
	})
	if err != nil {
		return "", err
	}
	if _, err := c.WaitDocumentResult(ctx, doc.DocumentID, pollInterval); err != nil {
		return doc.DocumentID, err
	}
	return doc.DocumentID, nil
}

// packedBox is a transport box (BOX_LV_1) with the group inside.
type packedBox struct {
	packedGroup
	SSCC          string
	AggregationID string
}

var boxFixture fixture[packedBox]

// fxBox packs the group into a new SSCC box.
func fxBox(t *testing.T) packedBox {
	t.Helper()
	group := fxGroup(t)
	return boxFixture.get(t, func(ctx context.Context, c *asl.Client) (packedBox, error) {
		sscc, err := asl.NewSSCC(0, cfg.TIN)
		if err != nil {
			return packedBox{}, err
		}
		docID, err := aggregate(ctx, c, sscc, 10, []string{group.GroupID})
		if err != nil {
			return packedBox{}, fmt.Errorf("box aggregation: %w", err)
		}
		return packedBox{packedGroup: group, SSCC: sscc, AggregationID: docID}, nil
	})
}

// packedPallet is a pallet (BOX_LV_2) with the box inside; only disaggregation tests break it.
type packedPallet struct {
	packedBox
	SSCC string
}

var palletFixture fixture[packedPallet]

// fxPallet packs the box into a new SSCC pallet.
func fxPallet(t *testing.T) packedPallet {
	t.Helper()
	box := fxBox(t)
	return palletFixture.get(t, func(ctx context.Context, c *asl.Client) (packedPallet, error) {
		sscc, err := asl.NewSSCC(1, cfg.TIN)
		if err != nil {
			return packedPallet{}, err
		}
		if _, err := aggregate(ctx, c, sscc, 5, []string{box.SSCC}); err != nil {
			return packedPallet{}, fmt.Errorf("pallet aggregation: %w", err)
		}
		return packedPallet{packedBox: box, SSCC: sscc}, nil
	})
}

// fakeUnit is a unit code with the real GTIN and a serial that does not exist.
const fakeUnit = "0104605246003585217ZZZZZZZZZZZZ"

// fakeUnitFull is fakeUnit with a fake check code.
const fakeUnitFull = fakeUnit + "\x1d93ZZZZ"

// fakeSSCC returns a valid SSCC that is never used (serial 9999999).
func fakeSSCC(t *testing.T) string {
	t.Helper()
	sscc, err := asl.GenerateSSCC(9, cfg.TIN, 9_999_999)
	requireOK(t, err)
	return sscc
}

// zeroUUID is a valid UUID that ASL does not know.
const zeroUUID = "00000000-0000-0000-0000-000000000000"
