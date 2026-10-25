package dto

// APIKeyCheckResponse tells if the API key belongs to the given TIN and when the key expires.
type APIKeyCheckResponse struct {
	IsTinCorrect bool      `json:"isTinCorrect"`        // true when key owner has this TIN
	ExpiresOn    *DateTime `json:"expiresOn,omitempty"` // only when IsTinCorrect is true
}

// APIKeyRefreshRequest selects the key to replace; send only one field: APIKey or ID.
type APIKeyRefreshRequest struct {
	APIKey string `json:"apiKey,omitempty"` // value of the key to replace
	ID     string `json:"id,omitempty"`     // ID of the key to replace
}

// APIKeyRefreshResponse is the new API key; the old key stops working.
type APIKeyRefreshResponse struct {
	APIKey    string   `json:"apiKey"`
	ID        string   `json:"id"`
	ExpiresOn DateTime `json:"expiresOn"`
	Label     string   `json:"label"`
}

// TechAuthRequest is the login and password of a technical user.
type TechAuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TokenResponse is the access and refresh token of a technical user.
type TokenResponse struct {
	AccessToken          string `json:"accessToken"`
	AccessTokenType      string `json:"accessTokenType"`      // always BEARER
	AccessTokenExpiresIn int64  `json:"accessTokenExpiresIn"` // lifetime in milliseconds
	RefreshToken         string `json:"refreshToken"`         // lives 24 hours
}
