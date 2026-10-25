package asl

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"asl/internal/dto"
)

// ASL limits (doc 1.4). The provider checks them before sending a request.
const (
	MaxOrderProducts     = 10      // sub-orders in one order
	MaxCodesPerProduct   = 150_000 // codes in one sub-order
	MaxReportCodes       = 30_000  // codes in one report or document
	MaxInfoCodes         = 1000    // codes in public, private and verify requests
	MaxOwnerCheckCodes   = 100     // codes in owner check request
	MinCodeLength        = 20      // shortest identity code ASL accepts
	MaxSeriesNumberLen   = 20      // longest series (batch) number
	MaxGroupItems        = 200     // items directly inside a GROUP package
	MaxBoxLv1Items       = 1500    // items directly inside a BOX_LV_1 package
	MaxBoxLv2Items       = 500     // items directly inside a BOX_LV_2 package
	MaxSetItems          = 200     // items directly inside a SET package
	DefaultDocErrorLimit = 100     // limit for document errors when the filter has 0
)

// digitsOnly matches a text of digits.
var digitsOnly = regexp.MustCompile(`^\d+$`)

// requireText returns an error when the value is empty.
func requireText(param, value string) error {
	// Spaces only is also empty.
	if strings.TrimSpace(value) == "" {
		return invalidArgument(param, "%s is required", param)
	}
	return nil
}

// requireTIN checks a TIN (9 digits) or PINFL (14 digits).
func requireTIN(param, tin string) error {
	// ASL uses INN for companies and PINFL for persons.
	tin = strings.TrimSpace(tin)
	if !digitsOnly.MatchString(tin) || (len(tin) != 9 && len(tin) != 14) {
		return invalidArgument(param, "%s must be 9 digits (INN) or 14 digits (PINFL), got %q",
			param, tin)
	}
	return nil
}

// requireGTIN checks that the GTIN has 14 digits.
func requireGTIN(param, gtin string) error {
	// The check digit is not tested here, ASL does it; use IsValidGTIN for a full check.
	if !digitsOnly.MatchString(gtin) || len(gtin) != 14 {
		return invalidArgument(param, "%s must be 14 digits with leading zeros, got %q", param, gtin)
	}
	return nil
}

// requireCodes checks that the list is not empty, not too long and has no empty items.
func requireCodes(param string, codes []string, maxCount int) error {
	// Codes are checked one by one, so the error shows the bad index.
	if len(codes) == 0 {
		return invalidArgument(param, "%s must have at least one code", param)
	}
	if len(codes) > maxCount {
		return invalidArgument(param, "%s has %d codes, max is %d", param, len(codes), maxCount)
	}
	for index, code := range codes {
		if strings.TrimSpace(code) == "" {
			return invalidArgument(param, "%s[%d] is empty", param, index)
		}
		if !utf8.ValidString(code) {
			return invalidArgument(param, "%s[%d] is not valid UTF-8", param, index)
		}
	}
	return nil
}

// requireIdentityCodes checks codes for info methods: ASL rejects short and non ASCII codes.
func requireIdentityCodes(param string, codes []string, maxCount int) error {
	// The whole request fails on ASL side for one bad code, so we stop it early.
	if err := requireCodes(param, codes, maxCount); err != nil {
		return err
	}
	for index, code := range codes {
		if len(code) < MinCodeLength {
			return invalidArgument(param, "%s[%d] is shorter than %d chars", param, index, MinCodeLength)
		}
		for _, char := range code {
			if char > 127 {
				return invalidArgument(param, "%s[%d] has non ASCII char %q", param, index, char)
			}
		}
	}
	return nil
}

// checkDateRange returns an error when "to" is before "from".
func checkDateRange(param string, from, to *dto.DateTime) error {
	// ASL also rejects it, but the provider gives a clearer message.
	if from == nil || to == nil || from.IsZero() || to.IsZero() {
		return nil
	}
	if to.Before(from.Time) {
		return invalidArgument(param, "%sTo must not be before %sFrom", param, param)
	}
	return nil
}

// pathID checks an ID used inside the URL path and escapes it.
func pathID(param, id string) (string, error) {
	// Escaping keeps "/" or "?" inside the ID from changing the URL.
	id = strings.TrimSpace(id)
	if id == "" {
		return "", invalidArgument(param, "%s is required", param)
	}
	return url.PathEscape(id), nil
}
