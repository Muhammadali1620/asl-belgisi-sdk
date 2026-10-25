package test

import (
	"slices"
	"testing"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// otherOwnerSSCC is an SSCC from the ASL doc example that belongs to another company.
const otherOwnerSSCC = "00030779729277777889"

func TestLiveCodeInfo(t *testing.T) {
	box := fxBox(t)
	unitID := asl.IdentityCode(box.Packed()[0])
	history := true
	noChildren := false
	public := func(req dto.PublicCodesRequest) func(e env) ([]dto.PublicCodeInfo, error) {
		return func(e env) ([]dto.PublicCodeInfo, error) { return e.asl.GetPublicCodes(e.ctx, req) }
	}
	private := func(req dto.PrivateCodesRequest) func(e env) (*dto.PrivateCodesResponse, error) {
		return func(e env) (*dto.PrivateCodesResponse, error) { return e.asl.GetPrivateCodes(e.ctx, req) }
	}
	owner := func(tin string, codes ...string) func(e env) (*dto.OwnerCheckResponse, error) {
		req := dto.OwnerCheckRequest{Codes: codes, OwnerTin: tin}
		return func(e env) (*dto.OwnerCheckResponse, error) { return e.asl.CheckCodeOwner(e.ctx, req) }
	}
	runCases(t, live(t), []testCase{
		pass("public info of box, group and unit",
			public(dto.PublicCodesRequest{Codes: []string{box.SSCC, box.GroupID, unitID},
				AddCodeHistory: &history}),
			func(t *testing.T, got []dto.PublicCodeInfo) {
				types := map[string]dto.PackageType{}
				for _, info := range got {
					types[info.Code] = info.PackageType
					if info.IssuerShortInfo.IssuerTin != cfg.TIN || info.Status != dto.CodeStatusIntroduced {
						t.Errorf("wrong public info: %+v", info)
					}
				}
				want := map[string]dto.PackageType{
					box.SSCC: dto.PackageBoxLv1, box.GroupID: dto.PackageGroup, unitID: dto.PackageUnit,
				}
				for code, packageType := range want {
					if types[code] != packageType {
						t.Errorf("package of %s = %s, want %s", code, types[code], packageType)
					}
				}
			}),
		pass("public info skips unknown codes",
			public(dto.PublicCodesRequest{Codes: []string{fakeUnit, fakeSSCC(t)}}),
			func(t *testing.T, got []dto.PublicCodeInfo) {
				if len(got) != 0 {
					t.Errorf("unknown codes must be skipped, got %d", len(got))
				}
			}),
		pass("private info of the box with children and history",
			private(dto.PrivateCodesRequest{Codes: []string{box.SSCC}, AddCodeHistory: &history}),
			func(t *testing.T, got *dto.PrivateCodesResponse) {
				if got.Redirected || len(got.Results) != 1 {
					t.Fatalf("want one private result: %+v", got)
				}
				info := got.Results[0]
				pack := info.PackageData
				if pack.PackageType != dto.PackageBoxLv1 || pack.ActuallyPacked != 1 ||
					len(pack.Children) != 1 || pack.Children[0].Code != box.GroupID {
					t.Errorf("box must hold the group: %+v", pack)
				}
				if info.TurnoverData == nil || info.TurnoverData.OwnerInfo.OwnerTin != cfg.TIN ||
					len(info.CodeHistory) == 0 {
					t.Errorf("owner or history is missing: %+v", info)
				}
			}),
		pass("private info of a unit without children",
			private(dto.PrivateCodesRequest{Codes: []string{unitID}, AddChildCodes: &noChildren}),
			func(t *testing.T, got *dto.PrivateCodesResponse) {
				if len(got.Results) != 1 {
					t.Fatalf("want one result: %+v", got)
				}
				info := got.Results[0]
				if info.PackageData.ParentCode != box.GroupID || info.ProductData == nil ||
					info.ProductData.GTIN != cfg.GTIN || info.MarkingData.UtilisationDate == nil {
					t.Errorf("unit data is wrong: %+v", info)
				}
			}),
		pass("private info of unknown code is empty",
			private(dto.PrivateCodesRequest{Codes: []string{fakeUnit}}),
			func(t *testing.T, got *dto.PrivateCodesResponse) {
				if len(got.Results) != 0 || got.Redirected {
					t.Errorf("want empty answer: %+v", got)
				}
			}),
		pass("private info of other owner codes falls back to public",
			private(dto.PrivateCodesRequest{Codes: []string{otherOwnerSSCC}}),
			func(t *testing.T, got *dto.PrivateCodesResponse) {
				if !got.Redirected || !slices.Contains(got.ForbiddenCodes, otherOwnerSSCC) {
					t.Errorf("ASL 307 must give Redirected=true: %+v", got)
				}
			}),
		pass("owner check of own codes gives children",
			owner(cfg.TIN, box.SSCC, box.GroupID, fakeUnit),
			func(t *testing.T, got *dto.OwnerCheckResponse) {
				children := map[string]int{}
				for _, item := range got.Results {
					children[item.Code] = len(item.Children)
				}
				if children[box.SSCC] != 1 || children[box.GroupID] != box.GroupSize ||
					!slices.Contains(got.MissingCodes, fakeUnit) {
					t.Errorf("wrong owner check: %+v", got)
				}
			}),
		pass("owner check with other tin",
			owner("307797292", unitID),
			func(t *testing.T, got *dto.OwnerCheckResponse) {
				if len(got.Results) != 0 || !slices.Contains(got.ForbiddenCodes, unitID) {
					t.Errorf("code of other owner must be forbidden: %+v", got)
				}
			}),
		pass("verify full codes",
			func(e env) ([]dto.VerifyResult, error) {
				return e.asl.VerifyCodes(e.ctx, []string{box.Packed()[0], box.Group, fakeUnitFull, box.SSCC})
			},
			func(t *testing.T, got []dto.VerifyResult) {
				verified := map[string]bool{}
				for _, result := range got {
					verified[result.Code] = result.Verified
					if result.Verified && result.ProductGroup != cfg.ProductGroup.ID() {
						t.Errorf("verified code must have group %d: %+v", cfg.ProductGroup.ID(), result)
					}
				}
				want := map[string]bool{
					box.Packed()[0]: true, box.Group: true, fakeUnitFull: false, box.SSCC: false,
				}
				for code, value := range want {
					if verified[code] != value {
						t.Errorf("verified[%q] = %v, want %v", code, verified[code], value)
					}
				}
			}),
		fail("public info with short code",
			public(dto.PublicCodesRequest{Codes: []string{"0104605246003585"}}),
			asl.CodeInvalidArgument),
	})
}
