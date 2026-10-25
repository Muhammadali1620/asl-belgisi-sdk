package test

import (
	"context"
	"strings"
	"testing"
	"time"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// AIK needs an E-IMZO signature; without a real key only the error paths are checked.
func TestLiveCustomsCode(t *testing.T) {
	client := live(t)
	ctx := testContext(t, liveTimeout)
	generated, err := client.GenerateCustomsCode(ctx)
	requireOK(t, err)
	code := generated.AggregatedCustomsCode
	if len(code) != 25 || strings.Trim(code, "0123456789") != "" {
		t.Fatalf("AIK must be 25 digits, got %q", code)
	}
	card, err := client.GetProductByGTIN(ctx, cfg.GTIN)
	requireOK(t, err)
	register := dto.CustomsCodeDocument{
		AggregatedCustomsCode: code,
		GTIN:                  cfg.GTIN,
		ProductID:             card.ID,
		ProductGroup:          cfg.ProductGroup,
		Tnved:                 card.Tnved.Code,
		Quantity:              1,
		Codes:                 []string{fakeUnit},
	}
	withdraw := dto.CustomsCodeWithdrawalDocument{
		BusinessDatetime: dto.NewDateTime(time.Now().UTC()), Code: code,
	}
	fakeSigner := asl.SignerFunc(func(context.Context, []byte) (string, error) {
		return "MIAGCSqGSIb3DQEHAqCAMIACAQEx", nil
	})
	signed, err := asl.New(asl.Config{BaseURL: cfg.BaseURL, Token: cfg.Token, Signer: fakeSigner})
	requireOK(t, err)
	registerWith := func(c *asl.Client) func(context.Context) error {
		return func(ctx context.Context) error { _, err := c.RegisterCustomsCode(ctx, register); return err }
	}
	withdrawWith := func(c *asl.Client) func(context.Context) error {
		return func(ctx context.Context) error { _, err := c.WithdrawCustomsCode(ctx, withdraw); return err }
	}
	cases := []struct {
		name  string
		call  func(context.Context) error
		code  asl.ErrorCode
		param string
	}{
		{"register without signature", registerWith(client), asl.CodeInvalidInput, "signature"},
		{"register with wrong signature", registerWith(signed), asl.CodeSignature, ""},
		{"withdraw without signature", withdrawWith(client), asl.CodeInvalidInput, "signature"},
		{"withdraw with wrong signature", withdrawWith(signed), asl.CodeSignature, ""},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			aslErr := requireCode(t, item.call(testContext(t, liveTimeout)), item.code)
			if aslErr.Parameter != item.param {
				t.Errorf("Parameter = %q, want %q", aslErr.Parameter, item.param)
			}
		})
	}
}
