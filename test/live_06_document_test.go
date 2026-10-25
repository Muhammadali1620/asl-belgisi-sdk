package test

import (
	"encoding/json"
	"testing"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

func TestLiveDocuments(t *testing.T) {
	group := fxGroup(t)
	lastIndex := 0
	search := func(filter dto.DocumentFilter) func(e env) (*dto.DocumentList, error) {
		return func(e env) (*dto.DocumentList, error) { return e.asl.SearchDocuments(e.ctx, filter) }
	}
	runCases(t, live(t), []testCase{
		pass("search by type and status",
			search(dto.DocumentFilter{
				Limit:    2,
				Types:    []dto.DocumentType{dto.DocumentTypeUtilisation},
				Statuses: []dto.DocumentStatus{dto.DocumentStatusSuccess},
			}),
			func(t *testing.T, got *dto.DocumentList) {
				for _, info := range got.DocumentInfos {
					if info.Type != dto.DocumentTypeUtilisation || info.Status != dto.DocumentStatusSuccess {
						t.Errorf("filter is not applied: %+v", info)
					}
				}
				if len(got.DocumentInfos) == 0 || len(got.DocumentInfos) > 2 {
					t.Errorf("want 1..2 documents, got %d", len(got.DocumentInfos))
				}
			}),
		pass("search unknown id is empty",
			search(dto.DocumentFilter{DocumentID: zeroUUID}),
			func(t *testing.T, got *dto.DocumentList) {
				if len(got.DocumentInfos) != 0 {
					t.Errorf("want nothing, got %d", len(got.DocumentInfos))
				}
			}),
		fail("search with unknown type",
			search(dto.DocumentFilter{Types: []dto.DocumentType{"FOO"}}), asl.CodeInvalidInput),
		pass("utilisation document card",
			func(e env) (*dto.DocumentCard, error) { return e.asl.GetDocument(e.ctx, group.ReportID) },
			func(t *testing.T, got *dto.DocumentCard) {
				if got.Type != dto.DocumentTypeUtilisation || got.Status != dto.DocumentStatusSuccess ||
					got.ProductGroup != cfg.ProductGroup {
					t.Errorf("wrong card: %+v", got)
				}
			}),
		pass("document json is the sent document",
			func(e env) (json.RawMessage, error) { return e.asl.GetDocumentJSON(e.ctx, group.AggregationID) },
			func(t *testing.T, got json.RawMessage) {
				var body dto.AggregationDocument
				requireOK(t, json.Unmarshal(got, &body))
				units := body.AggregationUnits
				if len(units) != 1 || units[0].UnitSerialNumber != group.GroupID {
					t.Errorf("wrong document json: %s", got)
				}
			}),
		pass("successful document has no errors",
			func(e env) (*dto.DocumentErrorList, error) {
				return e.asl.GetDocumentErrors(e.ctx, group.ReportID, dto.DocumentErrorsFilter{})
			},
			func(t *testing.T, got *dto.DocumentErrorList) {
				if len(got.DocumentErrors) != 0 {
					t.Errorf("want no errors, got %+v", got.DocumentErrors)
				}
			}),
		pass("document codes with page",
			func(e env) ([]dto.DocumentCodeState, error) {
				filter := dto.DocumentCodesFilter{Limit: 3, LastIndex: &lastIndex}
				return e.asl.GetDocumentCodes(e.ctx, group.ReportID, filter)
			},
			func(t *testing.T, got []dto.DocumentCodeState) {
				if len(got) == 0 || len(got) > 3 || got[0].Index <= lastIndex {
					t.Errorf("page after index %d with limit 3 is wrong: %+v", lastIndex, got)
				}
			}),
		fail("unknown document card",
			func(e env) (any, error) { return e.asl.GetDocument(e.ctx, zeroUUID) },
			asl.CodeNotFound),
		fail("unknown document json",
			func(e env) (any, error) { return e.asl.GetDocumentJSON(e.ctx, zeroUUID) },
			asl.CodeNotFound),
		fail("unknown document errors", func(e env) (any, error) {
			return e.asl.GetDocumentErrors(e.ctx, zeroUUID, dto.DocumentErrorsFilter{})
		}, asl.CodeNotFound),
		fail("unknown document codes", func(e env) (any, error) {
			return e.asl.GetDocumentCodes(e.ctx, zeroUUID, dto.DocumentCodesFilter{})
		}, asl.CodeNotFound),
		fail("document with bad id", func(e env) (any, error) {
			return e.asl.GetDocumentCodes(e.ctx, "abc", dto.DocumentCodesFilter{})
		}, asl.CodeInvalidInput),
	})
	t.Run("search by document id", func(t *testing.T) {
		// Stage search index is not updated for new documents, so an ID from the search is used.
		ctx := testContext(t, liveTimeout)
		page, err := live(t).SearchDocuments(ctx, dto.DocumentFilter{Limit: 1})
		requireOK(t, err)
		if len(page.DocumentInfos) != 1 {
			t.Fatalf("want one document in the page, got %d", len(page.DocumentInfos))
		}
		first := page.DocumentInfos[0]
		found, err := live(t).SearchDocuments(ctx, dto.DocumentFilter{DocumentID: first.DocumentID})
		requireOK(t, err)
		if len(found.DocumentInfos) != 1 || found.DocumentInfos[0].DocumentID != first.DocumentID ||
			found.DocumentInfos[0].Type != first.Type {
			t.Errorf("search by id must give the same document: %+v", found)
		}
	})
}
