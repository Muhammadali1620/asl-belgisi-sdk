package test

import (
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// documentCall sends a document and returns its ID.
type documentCall func(e env) (string, error)

// asyncFail makes a case where ASL accepts the document but its check fails with one of the codes.
func asyncFail(name string, send documentCall, codes ...asl.ErrorCode) testCase {
	return testCase{name: name, run: func(t *testing.T, e env) {
		documentID, err := send(e)
		requireOK(t, err)
		card, err := e.asl.WaitDocumentResult(e.ctx, documentID, pollInterval)
		aslErr := requireCode(t, err, codes...)
		if card == nil || !card.Status.IsFinal() || card.Status == dto.DocumentStatusSuccess {
			t.Errorf("document must be finished with errors: %+v", card)
		}
		if len(aslErr.Details) == 0 {
			t.Error("document errors must be in Details")
		}
	}}
}

// asyncPass makes a case where ASL must process the document with SUCCESS.
func asyncPass(name string, send documentCall) testCase {
	return testCase{name: name, run: func(t *testing.T, e env) {
		documentID, err := send(e)
		requireOK(t, err)
		card, err := e.asl.WaitDocumentResult(e.ctx, documentID, pollInterval)
		requireOK(t, err)
		if card.Status != dto.DocumentStatusSuccess {
			t.Errorf("status = %s, want SUCCESS", card.Status)
		}
	}}
}

// sendUtilisation sends a utilisation report and returns its ID.
func sendUtilisation(req dto.UtilisationRequest) documentCall {
	return func(e env) (string, error) {
		report, err := e.asl.SendUtilisation(e.ctx, req)
		if err != nil {
			return "", err
		}
		return report.ReportID, nil
	}
}

// sendDocument sends a document with one provider method and returns its ID.
func sendDocument(send func(e env) (*dto.DocumentIDResponse, error)) documentCall {
	return func(e env) (string, error) {
		result, err := send(e)
		if err != nil {
			return "", err
		}
		return result.DocumentID, nil
	}
}

func TestLiveUtilisation(t *testing.T) {
	applied := fxApplied(t)
	produced := time.Now().UTC().Add(-time.Minute)
	report := func(change func(*dto.UtilisationRequest)) dto.UtilisationRequest {
		req := utilisationRequest(applied.SpareReceived(), produced, applied.ShelfLife)
		change(&req)
		return req
	}
	send := func(change func(*dto.UtilisationRequest)) func(e env) (any, error) {
		return func(e env) (any, error) { return e.asl.SendUtilisation(e.ctx, report(change)) }
	}
	runCases(t, live(t), []testCase{
		pass("fixture report is SUCCESS (old status method)",
			func(e env) (*dto.UtilisationStatus, error) {
				return e.asl.GetUtilisationStatus(e.ctx, applied.ReportID)
			},
			func(t *testing.T, got *dto.UtilisationStatus) {
				if got.ReportStatus != dto.ReportStatusSuccess {
					t.Errorf("report status = %s", got.ReportStatus)
				}
			}),
		fail("no series for fertilizers",
			send(func(r *dto.UtilisationRequest) { r.SeriesNumber = "" }),
			asl.CodeInvalidInput),
		fail("expiration date in the past",
			send(func(r *dto.UtilisationRequest) {
				r.ExpirationDate = dto.NewDateTimePtr(produced.AddDate(-1, 0, 0))
			}),
			asl.CodeInvalidInput),
		fail("unknown product group",
			send(func(r *dto.UtilisationRequest) { r.ProductGroup = "foo" }),
			asl.CodeInvalidInput),
		fail("unknown business place",
			send(func(r *dto.UtilisationRequest) { r.BusinessPlaceID = 999999 }),
			asl.CodeInvalidInput),
		asyncFail("code that does not exist",
			sendUtilisation(report(func(r *dto.UtilisationRequest) { r.Sntins = []string{fakeUnitFull} })),
			asl.CodeNotFound),
		asyncFail("codes that are already applied",
			sendUtilisation(report(func(r *dto.UtilisationRequest) { r.Sntins = applied.Packed()[:1] })),
			asl.CodeInvalidInput), // ASL: invalid-code-properties
	})
}

func TestLiveValidation(t *testing.T) {
	codes := fxCodes(t)
	validation := func(code string) documentCall {
		doc := dto.ValidationDocument{
			ProductGroup:    cfg.ProductGroup,
			BusinessPlaceID: cfg.BusinessPlaceID,
			PackageType:     dto.PackageUnit,
			DocumentDate:    dto.NewDateTime(time.Now().UTC()),
			Codes:           []dto.ValidationCode{{Code: code, PrintQualityClass: dto.PrintQualityA}},
		}
		return sendDocument(func(e env) (*dto.DocumentIDResponse, error) {
			return e.asl.SendValidation(e.ctx, doc)
		})
	}
	runCases(t, live(t), []testCase{
		asyncPass("print quality of a received code",
			validation(asl.IdentityCode(codes.SpareReceived()[0]))),
		asyncFail("code that does not exist", validation(fakeUnit), asl.CodeNotFound),
	})
}

// aggregation sends one package with the children.
func aggregation(parent string, capacity int, children ...string) documentCall {
	doc := dto.AggregationDocument{
		BusinessPlaceID: cfg.BusinessPlaceID,
		DocumentDate:    dto.NewDateTime(time.Now().UTC()),
		AggregationUnits: []dto.AggregationUnit{{
			UnitSerialNumber:        parent,
			AggregationUnitCapacity: capacity,
			AggregationItemsCount:   len(children),
			Codes:                   children,
		}},
	}
	return sendDocument(func(e env) (*dto.DocumentIDResponse, error) {
		return e.asl.SendAggregation(e.ctx, doc)
	})
}

func TestLiveAggregation(t *testing.T) {
	box := fxBox(t)
	runCases(t, live(t), []testCase{
		pass("group package has its units",
			func(e env) ([]dto.DocumentCodeState, error) {
				return e.asl.GetDocumentCodes(e.ctx, box.packedGroup.AggregationID, dto.DocumentCodesFilter{})
			},
			func(t *testing.T, got []dto.DocumentCodeState) {
				if len(got) != box.GroupSize+1 {
					t.Errorf("want parent and %d children, got %d", box.GroupSize, len(got))
				}
				for _, state := range got {
					if state.State != dto.CodeStateSuccess {
						t.Errorf("code %s is %s: %s", state.Code, state.State, state.Result)
					}
				}
			}),
		asyncFail("child that does not exist",
			aggregation(fakeSSCC(t), 1, fakeUnit), asl.CodeNotFound),
		asyncFail("group that is already in a box",
			aggregation(mustSSCC(t), 2, box.GroupID),
			asl.CodeInvalidInput), // ASL: invalid-nested-codes
	})
}

func TestLiveCorrection(t *testing.T) {
	applied := fxApplied(t)
	correction := func(codes []string, fields dto.UpdatedFields) documentCall {
		doc := dto.CorrectionDocument{BusinessDatetime: dto.NewDateTime(time.Now().UTC()),
			Codes: codes, UpdatedFields: fields}
		return sendDocument(func(e env) (*dto.DocumentIDResponse, error) {
			return e.asl.SendCorrection(e.ctx, doc)
		})
	}
	spare := asl.IdentityCodes(applied.SpareApplied())
	runCases(t, live(t), []testCase{
		asyncPass("change series of an applied code",
			correction(spare, dto.UpdatedFields{SeriesNumber: "GO-FIXED"})),
		asyncFail("code that does not exist",
			correction([]string{fakeUnit}, dto.UpdatedFields{SeriesNumber: "X1"}), asl.CodeNotFound),
	})
	t.Run("new series is in code info", func(t *testing.T) {
		req := dto.PrivateCodesRequest{Codes: spare}
		result, err := live(t).GetPrivateCodes(testContext(t, liveTimeout), req)
		requireOK(t, err)
		if len(result.Results) != 1 || result.Results[0].ProductData.ProductSeries != "GO-FIXED" {
			t.Errorf("series must be GO-FIXED after correction: %+v", result.Results)
		}
	})
}

func TestLiveDisaggregation(t *testing.T) {
	pallet := fxPallet(t)
	disaggregation := func(code string) documentCall {
		doc := dto.DisaggregationDocument{BusinessDatetime: dto.NewDateTime(time.Now().UTC()),
			Codes: []string{code}}
		return sendDocument(func(e env) (*dto.DocumentIDResponse, error) {
			return e.asl.SendDisaggregation(e.ctx, doc)
		})
	}
	runCases(t, live(t), []testCase{
		asyncFail("package that does not exist", disaggregation(fakeSSCC(t)), asl.CodeNotFound),
		asyncPass("break the pallet", disaggregation(pallet.SSCC)),
	})
	t.Run("box keeps its group after pallet is broken", func(t *testing.T) {
		req := dto.OwnerCheckRequest{Codes: []string{pallet.SSCC, pallet.packedBox.SSCC}, OwnerTin: cfg.TIN}
		result, err := live(t).CheckCodeOwner(testContext(t, liveTimeout), req)
		requireOK(t, err)
		children := map[string][]string{}
		for _, item := range result.Results {
			children[item.Code] = item.Children
		}
		if len(children[pallet.SSCC]) != 0 || len(children[pallet.packedBox.SSCC]) != 1 {
			t.Errorf("pallet must be empty and box must keep 1 group: %+v", children)
		}
	})
}

// mustSSCC returns a new random SSCC of the test company.
func mustSSCC(t *testing.T) string {
	t.Helper()
	sscc, err := asl.NewSSCC(2, cfg.TIN)
	requireOK(t, err)
	return sscc
}
