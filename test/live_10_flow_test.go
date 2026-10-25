package test

import (
	"slices"
	"testing"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// TestLiveFlow walks the full code life step by step and checks the code in ASL after each step.
// Steps use the shared fixtures, so nothing is made twice when other live tests ran before.
func TestLiveFlow(t *testing.T) {
	// A printed code is APPLIED, or INTRODUCED at once when goods are made in Uzbekistan.
	printed := []dto.CodeStatus{dto.CodeStatusApplied, dto.CodeStatusIntroduced}
	received := []dto.CodeStatus{dto.CodeStatusReceived}
	steps := []struct {
		name   string
		code   func(t *testing.T) string // identity code to check after the step
		want   []dto.CodeStatus
		parent func(t *testing.T) string // expected parent code; nil means "do not check"
	}{
		{
			name: "1 order and download codes",
			code: func(t *testing.T) string { return asl.IdentityCode(fxCodes(t).SpareReceived()[0]) },
			want: received,
		},
		{
			name: "2 utilisation (print codes)",
			code: func(t *testing.T) string { return asl.IdentityCode(fxApplied(t).SpareApplied()[0]) },
			want: printed,
		},
		{
			name:   "3 pack units into the group",
			code:   func(t *testing.T) string { return asl.IdentityCode(fxGroup(t).Packed()[0]) },
			want:   printed,
			parent: func(t *testing.T) string { return fxGroup(t).GroupID },
		},
		{
			name:   "4 pack the group into a box",
			code:   func(t *testing.T) string { return fxBox(t).GroupID },
			want:   printed,
			parent: func(t *testing.T) string { return fxBox(t).SSCC },
		},
		{
			name: "5 pack the box into a pallet",
			code: func(t *testing.T) string { return fxPallet(t).packedBox.SSCC },
			want: printed,
		},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			code := step.code(t)
			req := dto.PrivateCodesRequest{Codes: []string{code}}
			result, err := live(t).GetPrivateCodes(testContext(t, liveTimeout), req)
			requireOK(t, err)
			if len(result.Results) != 1 {
				t.Fatalf("code %s is not found", code)
			}
			info := result.Results[0]
			t.Logf("%s: status %s, package %s, parent %q", code, info.CodeData.Status,
				info.PackageData.PackageType, info.PackageData.ParentCode)
			if !slices.Contains(step.want, info.CodeData.Status) {
				t.Errorf("status = %s, want one of %v", info.CodeData.Status, step.want)
			}
			if step.parent != nil && info.PackageData.ParentCode != step.parent(t) {
				t.Errorf("parent = %q, want %q", info.PackageData.ParentCode, step.parent(t))
			}
		})
	}
}
