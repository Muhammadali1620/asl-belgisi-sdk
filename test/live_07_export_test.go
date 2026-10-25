package test

import (
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

func TestLiveExport(t *testing.T) {
	client := live(t)
	runCases(t, client, []testCase{
		fail("export of unknown gtin", func(e env) (any, error) {
			return e.asl.CreateExport(e.ctx, dto.ExportRequest{GTIN: "04600000000006"})
		}, asl.CodeNotFound),
		fail("status of unknown export",
			func(e env) (any, error) { return e.asl.GetExportStatus(e.ctx, zeroUUID) },
			asl.CodeNotFound),
		fail("status with bad id",
			func(e env) (any, error) { return e.asl.GetExportStatus(e.ctx, "abc") },
			asl.CodeInvalidInput),
		fail("download unknown export",
			func(e env) (any, error) { return e.asl.DownloadExport(e.ctx, zeroUUID) },
			asl.CodeNotFound),
	})
	t.Run("export, wait and read codes", func(t *testing.T) {
		ctx := testContext(t, liveTimeout)
		// Stage export data is not updated for new codes, so no date filter is used.
		req := dto.ExportRequest{GTIN: cfg.GTIN, PackageType: []dto.PackageType{dto.PackageUnit}}
		export, err := client.CreateExport(ctx, req)
		requireOK(t, err)
		status, err := client.WaitExport(ctx, export.ID, time.Second)
		requireOK(t, err)
		if status != dto.ExportStatusSuccess {
			t.Fatalf("export status = %s", status)
		}
		archive, err := client.DownloadExport(ctx, export.ID)
		requireOK(t, err)
		codes, err := asl.ReadExportCodes(archive)
		requireOK(t, err)
		if len(codes) == 0 {
			t.Fatal("export of our unit codes must not be empty")
		}
		for _, code := range codes {
			if code.PackageData.PackageType != dto.PackageUnit || code.CodeData.Code == "" {
				t.Fatalf("filter by package type is not applied: %+v", code)
			}
		}
		t.Logf("export has %d codes", len(codes))
	})
}
