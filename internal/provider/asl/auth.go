package asl

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"asl/internal/dto"
)

// Auth paths.
const (
	pathParties      = "/public/api/v1/party/parties/"
	pathAuthenticate = "/api/users/authenticate"
	pathTokenRefresh = "/api/users/tokens/refresh"
)

func (c *Client) CheckAPIKey(ctx context.Context, tin string) (*dto.APIKeyCheckResponse, error) {
	// GET /public/api/v1/party/parties/{tin}/api-keys/check. Limit: 100 calls in 24 hours.
	if err := requireTIN("tin", tin); err != nil {
		return nil, err
	}
	var out dto.APIKeyCheckResponse
	if err := c.get(ctx, pathParties+strings.TrimSpace(tin)+"/api-keys/check", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RefreshAPIKey(
	ctx context.Context, tin string, req dto.APIKeyRefreshRequest,
) (*dto.APIKeyRefreshResponse, error) {
	// POST /public/api/v1/party/parties/{tin}/api-keys/refresh. Needs "API keys manager" role.
	// Send only APIKey or only ID. If it is our own key, call SetToken with the new key after this.
	if err := requireTIN("tin", tin); err != nil {
		return nil, err
	}
	hasKey, hasID := strings.TrimSpace(req.APIKey) != "", strings.TrimSpace(req.ID) != ""
	if hasKey == hasID {
		return nil, invalidArgument("apiKey", "set exactly one of apiKey or id")
	}
	var out dto.APIKeyRefreshResponse
	path := pathParties + strings.TrimSpace(tin) + "/api-keys/refresh"
	if err := c.post(ctx, path, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) AuthenticateTechUser(
	ctx context.Context, req dto.TechAuthRequest,
) (*dto.TokenResponse, error) {
	// POST /api/users/authenticate without token. A new token makes the old access token invalid.
	// The token is not set automatically: call SetToken(resp.AccessToken) to use it.
	if err := requireText("login", req.Login); err != nil {
		return nil, err
	}
	if err := requireText("password", req.Password); err != nil {
		return nil, err
	}
	var out dto.TokenResponse
	call := request{method: http.MethodPost, path: pathAuthenticate, json: req, noAuth: true}
	if err := c.call(ctx, call, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RefreshTechToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	// POST /api/users/tokens/refresh with form body refreshToken=..., without token.
	if err := requireText("refreshToken", refreshToken); err != nil {
		return nil, err
	}
	form := url.Values{"refreshToken": {strings.TrimSpace(refreshToken)}}
	var out dto.TokenResponse
	call := request{method: http.MethodPost, path: pathTokenRefresh, form: form, noAuth: true}
	if err := c.call(ctx, call, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
