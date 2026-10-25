package test

import (
	"testing"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

func TestLiveAuth(t *testing.T) {
	client := live(t)
	otherTIN := "307797292"
	refresh := func(tin string, req dto.APIKeyRefreshRequest) func(e env) (any, error) {
		return func(e env) (any, error) { return e.asl.RefreshAPIKey(e.ctx, tin, req) }
	}
	runCases(t, client, []testCase{
		pass("check key of own tin",
			func(e env) (*dto.APIKeyCheckResponse, error) { return e.asl.CheckAPIKey(e.ctx, cfg.TIN) },
			func(t *testing.T, got *dto.APIKeyCheckResponse) {
				if !got.IsTinCorrect || got.ExpiresOn == nil || got.ExpiresOn.IsZero() {
					t.Errorf("own key must be correct and have expire date: %+v", got)
				}
			}),
		pass("check key of other tin",
			func(e env) (*dto.APIKeyCheckResponse, error) { return e.asl.CheckAPIKey(e.ctx, otherTIN) },
			func(t *testing.T, got *dto.APIKeyCheckResponse) {
				if got.IsTinCorrect || got.ExpiresOn != nil {
					t.Errorf("other tin must be false without date: %+v", got)
				}
			}),
		fail("check key with bad tin format",
			func(e env) (any, error) { return e.asl.CheckAPIKey(e.ctx, "12345") },
			asl.CodeInvalidArgument),
		fail("refresh key of other tin",
			refresh(otherTIN, dto.APIKeyRefreshRequest{ID: zeroUUID}), asl.CodeInvalidInput),
		fail("refresh unknown key id",
			refresh(cfg.TIN, dto.APIKeyRefreshRequest{ID: zeroUUID}), asl.CodeNotFound),
		fail("refresh unknown key value",
			refresh(cfg.TIN, dto.APIKeyRefreshRequest{APIKey: zeroUUID}), asl.CodeNotFound),
		fail("refresh with key and id",
			refresh(cfg.TIN, dto.APIKeyRefreshRequest{APIKey: zeroUUID, ID: zeroUUID}),
			asl.CodeInvalidArgument),
		fail("tech user wrong password", func(e env) (any, error) {
			login := dto.TechAuthRequest{Login: "no-such-user", Password: "wrong"}
			return e.asl.AuthenticateTechUser(e.ctx, login)
		}, asl.CodeUnauthorized),
		fail("tech user empty login", func(e env) (any, error) {
			return e.asl.AuthenticateTechUser(e.ctx, dto.TechAuthRequest{Password: "x"})
		}, asl.CodeInvalidArgument),
		fail("tech token unknown",
			func(e env) (any, error) { return e.asl.RefreshTechToken(e.ctx, zeroUUID) },
			asl.CodeInvalidInput),
	})
	t.Run("bad token", func(t *testing.T) {
		_, err := liveWithToken(t, zeroUUID).CheckAPIKey(testContext(t, liveTimeout), cfg.TIN)
		aslErr := requireCode(t, err, asl.CodeUnauthorized)
		if aslErr.HTTPStatus != 401 || aslErr.ASLCode != "invalid-token" {
			t.Errorf("want 401 invalid-token, got %d %s", aslErr.HTTPStatus, aslErr.ASLCode)
		}
	})
	t.Run("tech user login and refresh", func(t *testing.T) {
		if cfg.TechLogin == "" || cfg.TechPassword == "" {
			t.Skip("ASL_TECH_LOGIN and ASL_TECH_PASSWORD are not set")
		}
		ctx := testContext(t, liveTimeout)
		tech, err := asl.New(asl.Config{BaseURL: cfg.BaseURL})
		requireOK(t, err)
		login := dto.TechAuthRequest{Login: cfg.TechLogin, Password: cfg.TechPassword}
		tokens, err := tech.AuthenticateTechUser(ctx, login)
		requireOK(t, err)
		tech.SetToken(tokens.AccessToken)
		_, err = tech.ListOrders(ctx, dto.OrderFilter{Limit: 1})
		requireOK(t, err)
		refreshed, err := tech.RefreshTechToken(ctx, tokens.RefreshToken)
		requireOK(t, err)
		if refreshed.AccessToken == "" || refreshed.AccessToken == tokens.AccessToken {
			t.Errorf("refresh must give a new access token")
		}
	})
	t.Run("refresh own api key", func(t *testing.T) {
		if !cfg.AllowKeyRefresh {
			t.Skip("this test REPLACES the real API key; set ASL_ALLOW_KEY_REFRESH=1 to run it")
		}
		ctx := testContext(t, liveTimeout)
		req := dto.APIKeyRefreshRequest{APIKey: client.Token()}
		fresh, err := client.RefreshAPIKey(ctx, cfg.TIN, req)
		requireOK(t, err)
		client.SetToken(fresh.APIKey)
		t.Logf("NEW API KEY (save it in test/.env): %s, expires %s", fresh.APIKey, fresh.ExpiresOn)
		check, err := client.CheckAPIKey(ctx, cfg.TIN)
		requireOK(t, err)
		if !check.IsTinCorrect {
			t.Error("new key must belong to the tin")
		}
	})
}

func TestLiveParty(t *testing.T) {
	runCases(t, live(t), []testCase{
		pass("own company",
			func(e env) (*dto.PartyStatus, error) { return e.asl.GetPartyStatus(e.ctx, cfg.TIN) },
			func(t *testing.T, got *dto.PartyStatus) {
				if got.Tin != cfg.TIN || got.Name.Get("ru") == "" || len(got.ProductGroups) == 0 {
					t.Errorf("wrong party: %+v", got)
				}
			}),
		fail("unknown tin",
			func(e env) (any, error) { return e.asl.GetPartyStatus(e.ctx, "999999999") },
			asl.CodeNotFound),
		fail("bad tin format",
			func(e env) (any, error) { return e.asl.GetPartyStatus(e.ctx, "abc") },
			asl.CodeInvalidArgument),
	})
	t.Run("bad token", func(t *testing.T) {
		_, err := liveWithToken(t, zeroUUID).GetPartyStatus(testContext(t, liveTimeout), cfg.TIN)
		requireCode(t, err, asl.CodeUnauthorized)
	})
}
