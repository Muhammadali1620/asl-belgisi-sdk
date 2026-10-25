package test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

func TestUnitDateTime(t *testing.T) {
	inputs := map[string]time.Time{
		`"2026-10-25T03:28:06.898Z"`:       time.Date(2026, 10, 25, 3, 28, 6, 898_000_000, time.UTC),
		`"2025-01-30T13:08:06.47Z"`:        time.Date(2025, 1, 30, 13, 8, 6, 470_000_000, time.UTC),
		`"2026-10-25T02:45:28.826720791Z"`: time.Date(2026, 10, 25, 2, 45, 28, 826_720_791, time.UTC),
		`"2028-01-01T00:00:00+00:00"`:      time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
		`"2024-12-03T10:15:30+01:00"`:      time.Date(2024, 12, 3, 9, 15, 30, 0, time.UTC),
		`"2026-02-24T07:46:23"`:            time.Date(2026, 2, 24, 7, 46, 23, 0, time.UTC),
		`"2026-02-24"`:                     time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC),
	}
	for input, want := range inputs {
		var got dto.DateTime
		requireOK(t, json.Unmarshal([]byte(input), &got))
		if !got.Equal(want) {
			t.Errorf("%s -> %s, want %s", input, got.Time, want)
		}
	}
	for _, empty := range []string{`null`, `""`} {
		var got dto.DateTime
		requireOK(t, json.Unmarshal([]byte(empty), &got))
		if !got.IsZero() {
			t.Errorf("%s must give zero time", empty)
		}
	}
	var bad dto.DateTime
	if json.Unmarshal([]byte(`"yesterday"`), &bad) == nil {
		t.Error("bad date must give an error")
	}
	local := time.Date(2026, 1, 2, 8, 4, 5, 123_456_789, time.FixedZone("UZ", 5*3600))
	raw, err := json.Marshal(struct {
		At    dto.DateTime  `json:"at"`
		Empty dto.DateTime  `json:"empty"`
		Ptr   *dto.DateTime `json:"ptr,omitempty"`
		Day   dto.Date      `json:"day"`
	}{At: dto.NewDateTime(local), Day: dto.Date{Time: local}})
	requireOK(t, err)
	want := `{"at":"2026-01-02T03:04:05.123Z","empty":null,"day":"2026-01-02"}`
	if string(raw) != want {
		t.Errorf("marshal = %s, want %s", raw, want)
	}
}

func TestUnitSmallTypes(t *testing.T) {
	var flex struct {
		A dto.FlexString `json:"a"`
		B dto.FlexString `json:"b"`
		C dto.FlexString `json:"c"`
	}
	requireOK(t, json.Unmarshal([]byte(`{"a":603,"b":"text","c":null}`), &flex))
	if flex.A != "603" || flex.B != "text" || flex.C != "" {
		t.Errorf("FlexString = %+v", flex)
	}
	text := dto.LocalizedText{"ru": "Привет", "uz": "Salom"}
	empty := dto.LocalizedText{}
	if text.Get("uz") != "Salom" || text.Get("en") != "Привет" || empty.Get("ru") != "" {
		t.Error("LocalizedText.Get is wrong")
	}
	if dto.ProductGroupFertilizers.ID() != 53 || dto.ProductGroup("tobacco").ID() != 0 {
		t.Error("ProductGroup.ID is wrong")
	}
	if group, ok := dto.ProductGroupByID(7); !ok || group != dto.ProductGroupPharma {
		t.Error("ProductGroupByID is wrong")
	}
	if _, ok := dto.ProductGroupByID(3); ok {
		t.Error("tobacco must not be in product groups")
	}
	if !dto.CountryUZ.IsValid() || dto.CountryCode("XX").IsValid() {
		t.Error("CountryCode.IsValid is wrong")
	}
	if !dto.DocumentStatusPartiallyProcessed.IsFinal() || dto.DocumentStatusInProcess.IsFinal() {
		t.Error("DocumentStatus.IsFinal is wrong")
	}
	if !dto.ExportStatusExpired.IsFinal() || dto.ExportStatusCreated.IsFinal() {
		t.Error("ExportStatus.IsFinal is wrong")
	}
	sub := dto.SubOrderInfo{RejectionReason: "321: The product code was not found"}
	if sub.RejectionCode() != "321" || (dto.SubOrderInfo{}).RejectionCode() != "" {
		t.Error("RejectionCode is wrong")
	}
	if !(dto.PartyStatus{Tin: "1"}).Found() || (dto.PartyStatus{}).Found() {
		t.Error("PartyStatus.Found is wrong")
	}
}

func TestUnitProductCard(t *testing.T) {
	raw := `{"id":"p","gtin":"08001090970459","scope":"GLOBAL","attributes":[
		{"meta":{"code":"nested_product","type":"INNER_GTIN"},"value":"04605246003585"},
		{"meta":{"code":"nested_products_count","type":"INTEGER"},"value":12},
		{"meta":{"code":"expiration_date","type":"INTEGER"},"value":24,"unit":{"code":"MON"}},
		{"meta":{"code":"nominal_quantity","type":"DOUBLE"},"value":2000.5},
		{"meta":{"code":"product_name","type":"STRING_LOCALIZED"},"value":{"ru":"Natriy"}},
		{"meta":{"code":"right_side","type":"FILE_LIST"},"value":null},
		{"meta":{"code":"front_side","type":"FILE_LIST"},
			"value":{"files":[{"fileName":"075bd75e.jpg","hashType":"SHA3_256"}]}},
		{"meta":{"code":"text_number","type":"STRING"},"value":"15"}]}`
	var card dto.ProductCard
	requireOK(t, json.Unmarshal([]byte(raw), &card))
	if capacity, ok := card.IntAttribute(dto.AttrNestedProductsCount); !ok || capacity != 12 {
		t.Errorf("capacity = %d %v", capacity, ok)
	}
	if inner, ok := card.StringAttribute(dto.AttrNestedProduct); !ok || inner != "04605246003585" {
		t.Errorf("nested product = %q", inner)
	}
	shelf, ok := card.Attribute(dto.AttrExpirationDate)
	if months, _ := shelf.Int(); !ok || months != 24 || shelf.Unit.Code != "MON" {
		t.Errorf("shelf life = %+v", shelf)
	}
	nominal, _ := card.Attribute(dto.AttrNominalQuantity)
	if value, ok := nominal.Float(); !ok || value != 2000.5 {
		t.Errorf("nominal = %v", value)
	}
	if _, ok := nominal.Int(); ok {
		t.Error("2000.5 is not an integer")
	}
	name, _ := card.Attribute(dto.AttrProductName)
	if text, ok := name.Localized(); !ok || text.Get("ru") != "Natriy" {
		t.Error("localized attribute is wrong")
	}
	if number, ok := card.IntAttribute("text_number"); !ok || number != 15 {
		t.Error("number in text must be read")
	}
	if _, ok := card.Attribute("right_side"); ok {
		t.Error("null attribute must be missing")
	}
	files := card.AllFiles()
	if len(files) != 1 || files[0].ID() != "075bd75e" {
		t.Errorf("files = %+v", files)
	}
}

func TestUnitExportArchive(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	_, _ = writer.Create("folder/")
	file, _ := writer.Create("export.json")
	_, _ = file.Write([]byte(`{"results":[{"codeData":{"code":"c1","status":"INTRODUCED"},` +
		`"packageData":{"packageType":"UNIT"}}]}`))
	requireOK(t, writer.Close())
	files, err := asl.ReadExportArchive(buffer.Bytes())
	requireOK(t, err)
	if len(files) != 1 || files[0].Name != "export.json" {
		t.Fatalf("files = %+v", files)
	}
	codes, err := asl.ReadExportCodes(buffer.Bytes())
	requireOK(t, err)
	if len(codes) != 1 || codes[0].CodeData.Code != "c1" {
		t.Errorf("codes = %+v", codes)
	}
	_, err = asl.ReadExportArchive([]byte("not a zip"))
	requireCode(t, err, asl.CodeDecode)
}
