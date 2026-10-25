package test

import (
	"strings"
	"testing"

	"asl/internal/provider/asl"
)

// Real codes from the ASL stage server.
const (
	stageUnitCode  = "0104605246003585217H3Je/ekvCq8M\x1d93At65"
	stageGroupCode = "0108001090970459217G_M/6\"R!!(>-\x1d93q602"
	stageSSCC      = "00031245889400000019"
)

func TestUnitCheckDigit(t *testing.T) {
	cases := []struct {
		digits string
		want   int
	}{
		{"0460524600358", 5},     // GTIN 04605246003585
		{"0800109097045", 9},     // GTIN 08001090970459
		{"03124588940000001", 9}, // SSCC 00031245889400000019 without AI and check digit
		{"0", 0},
	}
	for _, item := range cases {
		got, err := asl.CheckDigit(item.digits)
		requireOK(t, err)
		if got != item.want {
			t.Errorf("CheckDigit(%q) = %d, want %d", item.digits, got, item.want)
		}
	}
	for _, bad := range []string{"", "12a4", " 123"} {
		_, err := asl.CheckDigit(bad)
		requireCode(t, err, asl.CodeInvalidArgument)
	}
}

func TestUnitGTIN(t *testing.T) {
	valid := []string{"04605246003585", "08001090970459", "4605246003585", "96385074"}
	for _, gtin := range valid {
		if !asl.IsValidGTIN(gtin) {
			t.Errorf("IsValidGTIN(%q) = false, want true", gtin)
		}
	}
	invalid := []string{"", "04605246003586", "0460524600358", "0460524600358X", "123456789012345"}
	for _, gtin := range invalid {
		if asl.IsValidGTIN(gtin) {
			t.Errorf("IsValidGTIN(%q) = true, want false", gtin)
		}
	}
	padded, err := asl.NormalizeGTIN(" 4605246003585 ")
	requireOK(t, err)
	if padded != "04605246003585" {
		t.Errorf("NormalizeGTIN = %q", padded)
	}
	_, err = asl.NormalizeGTIN("04605246003586")
	requireCode(t, err, asl.CodeInvalidArgument)
}

func TestUnitSSCC(t *testing.T) {
	got, err := asl.GenerateSSCC(0, "312458894", 1)
	requireOK(t, err)
	if got != stageSSCC {
		t.Fatalf("GenerateSSCC = %q, want %q (real stage SSCC)", got, stageSSCC)
	}
	if !asl.IsValidSSCC(got) || !asl.IsSSCC(got) || !asl.IsValidSSCC(strings.TrimPrefix(got, "00")) {
		t.Errorf("SSCC %q must be valid in 20 and 18 digit form", got)
	}
	serial, err := asl.SSCCSerial(got, "312458894")
	requireOK(t, err)
	if serial != 1 {
		t.Errorf("SSCCSerial = %d, want 1", serial)
	}
	maxSerial, err := asl.GenerateSSCC(9, "312458894", 9_999_999)
	requireOK(t, err)
	if !asl.IsValidSSCC(maxSerial) || !strings.HasPrefix(maxSerial, "009312458894") {
		t.Errorf("max serial SSCC is wrong: %q", maxSerial)
	}
	seen := map[string]bool{}
	for range 50 {
		random, err := asl.NewSSCC(0, "312458894")
		requireOK(t, err)
		if !asl.IsValidSSCC(random) || len(random) != 20 || !strings.HasPrefix(random, "000312458894") {
			t.Fatalf("NewSSCC gave bad code %q", random)
		}
		seen[random] = true
	}
	if len(seen) < 45 {
		t.Errorf("NewSSCC gives too many equal codes: %d unique of 50", len(seen))
	}
}

func TestUnitSSCCErrors(t *testing.T) {
	errorCases := []struct {
		name      string
		extension int
		prefix    string
		serial    uint64
	}{
		{"extension too big", 10, "312458894", 1},
		{"negative extension", -1, "312458894", 1},
		{"prefix with letters", 0, "31245889A", 1},
		{"prefix too short", 0, "123", 1},
		{"prefix too long", 0, "1234567890123", 1},
		{"serial does not fit", 0, "312458894", 10_000_000},
	}
	for _, item := range errorCases {
		t.Run(item.name, func(t *testing.T) {
			_, err := asl.GenerateSSCC(item.extension, item.prefix, item.serial)
			requireCode(t, err, asl.CodeInvalidArgument)
		})
	}
	for _, bad := range []string{"", "00031245889400000018", "0003124588940000001X", "1003124588940000001"} {
		if asl.IsValidSSCC(bad) {
			t.Errorf("IsValidSSCC(%q) = true, want false", bad)
		}
	}
	_, err := asl.SSCCSerial(stageSSCC, "307797292")
	requireCode(t, err, asl.CodeInvalidArgument)
	_, err = asl.SSCCSerial("00031245889400000018", "312458894")
	requireCode(t, err, asl.CodeInvalidArgument)
}

func TestUnitParseCode(t *testing.T) {
	variants := map[string]string{
		"raw GS":         stageUnitCode,
		"text <GS>":      strings.ReplaceAll(stageUnitCode, "\x1d", "<GS>"),
		"text \\u001d":   strings.ReplaceAll(stageUnitCode, "\x1d", `\u001d`),
		"scanner prefix": "]d2" + stageUnitCode,
		"spaces":         "  " + stageUnitCode + "\n",
	}
	for name, raw := range variants {
		t.Run(name, func(t *testing.T) {
			code, err := asl.ParseCode(raw)
			requireOK(t, err)
			if code.GTIN != "04605246003585" || code.Serial != "7H3Je/ekvCq8M" || code.CheckCode != "At65" {
				t.Fatalf("wrong parse: %+v", code)
			}
			if code.Identity() != "0104605246003585217H3Je/ekvCq8M" {
				t.Fatalf("wrong identity %q", code.Identity())
			}
			if code.Raw != stageUnitCode {
				t.Fatalf("Raw must be normalized, got %q", code.Raw)
			}
		})
	}
	group, err := asl.ParseCode(stageGroupCode)
	requireOK(t, err)
	if group.GTIN != "08001090970459" || group.Serial != `7G_M/6"R!!(>-` {
		t.Errorf("wrong group parse: %+v", group)
	}
	long, err := asl.ParseCode("010460524600358521ABC123\x1d91EE07\x1d92dGVzdA==\x1d17270101")
	requireOK(t, err)
	if long.CryptoKey != "EE07" || long.Crypto != "dGVzdA==" || long.Other["17"] != "270101" {
		t.Errorf("wrong AI parse: %+v", long)
	}
	for _, bad := range []string{"", "abc", "0104605246", "2112345", stageSSCC} {
		_, err := asl.ParseCode(bad)
		requireCode(t, err, asl.CodeInvalidArgument)
	}
}

func TestUnitIdentityCode(t *testing.T) {
	cases := map[string]string{
		stageUnitCode:  "0104605246003585217H3Je/ekvCq8M",
		stageGroupCode: `0108001090970459217G_M/6"R!!(>-`,
		stageSSCC:      stageSSCC,
		"0104605246003585217H3Je/ekvCq8M<GS>93At65": "0104605246003585217H3Je/ekvCq8M",
	}
	for raw, want := range cases {
		if got := asl.IdentityCode(raw); got != want {
			t.Errorf("IdentityCode(%q) = %q, want %q", raw, got, want)
		}
	}
	list := asl.IdentityCodes([]string{stageUnitCode, "0104605246003585217H3Je/ekvCq8M", "", stageSSCC})
	if len(list) != 2 || list[0] != "0104605246003585217H3Je/ekvCq8M" || list[1] != stageSSCC {
		t.Errorf("IdentityCodes must remove repeats and keep order, got %q", list)
	}
}
