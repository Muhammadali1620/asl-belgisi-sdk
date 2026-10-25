package test

import (
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// Code transfer works only for pharma and medicals, so for other groups only errors are checked.
func TestLiveTransfer(t *testing.T) {
	search := func(filter dto.TransferSearchFilter) func(e env) (*dto.TransferList, error) {
		return func(e env) (*dto.TransferList, error) { return e.asl.SearchTransferRequests(e.ctx, filter) }
	}
	request := dto.TransferRequestDocument{
		BuyerID:                   688,
		OwnerID:                   3,
		BusinessDatetime:          dto.NewDateTime(time.Now().UTC()),
		ExternalDocumentReference: "test",
		Products:                  []dto.TransferProduct{{GTIN: cfg.GTIN, Quantity: 1}},
		Codes:                     []string{fakeUnit},
	}
	runCases(t, live(t), []testCase{
		pass("search transfer requests",
			search(dto.TransferSearchFilter{Limit: 2,
				Statuses: []dto.DocumentStatus{dto.DocumentStatusError}}),
			func(t *testing.T, got *dto.TransferList) {
				for _, info := range got.DocumentInfos {
					if info.Status != dto.DocumentStatusError {
						t.Errorf("status filter is not applied: %+v", info)
					}
				}
			}),
		fail("search with unknown status",
			search(dto.TransferSearchFilter{Statuses: []dto.DocumentStatus{"FOO"}}),
			asl.CodeInvalidInput),
		asyncFail("request transfer of unknown code",
			sendDocument(func(e env) (*dto.DocumentIDResponse, error) {
				return e.asl.SendTransferRequest(e.ctx, request)
			}),
			asl.CodeNotFound),
		fail("confirm unknown transfer request", func(e env) (any, error) {
			doc := dto.TransferConfirmationDocument{TransferRequestDocID: zeroUUID,
				Resolution: dto.ResolutionConfirmed}
			return e.asl.SendTransferConfirmation(e.ctx, doc)
		}, asl.CodeNotFound),
	})
}
