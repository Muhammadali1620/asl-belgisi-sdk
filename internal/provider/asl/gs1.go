package asl

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// GS is the ASCII 29 group separator used inside full marking codes.
const GS = "\x1d"

// SSCC sizes: 18 digits of SSCC itself and 20 chars with the "00" application identifier.
const (
	ssccDigits    = 18
	ssccWithAI    = 20
	ssccAI        = "00"
	ssccPayloadSz = ssccDigits - 1 // extension + company prefix + serial, without check digit
)

// Code is a parsed GS1 DataMatrix marking code.
type Code struct {
	Raw       string            // normalized input (GS is ASCII 29)
	GTIN      string            // AI 01
	Serial    string            // AI 21
	CryptoKey string            // AI 91, key ID
	Crypto    string            // AI 92, crypto signature
	CheckCode string            // AI 93, short check code
	Other     map[string]string // other AIs, e.g. 17 (expiry), 10 (batch)
}

// Identity returns the identity code (01 + GTIN + 21 + serial) used by info and aggregation methods.
func (c Code) Identity() string {
	// Crypto tail (91, 92, 93) is not part of the identity code.
	return "01" + c.GTIN + "21" + c.Serial
}

// fixedAIs are GS1 application identifiers with fixed length (value length without AI).
var fixedAIs = map[string]int{"01": 14, "02": 14, "11": 6, "13": 6, "15": 6, "17": 6, "8005": 6}

// knownVarAIs are variable length AIs used in ASL codes; other 2 digit AIs are read as variable too.
var knownVarAIs = []string{"240", "21", "10", "91", "92", "93"}

// CheckDigit returns the GS1 mod-10 check digit for digits without the check digit.
func CheckDigit(digits string) (int, error) {
	// From the right: weights 3, 1, 3, 1... Same rule for GTIN and SSCC.
	if digits == "" {
		return 0, invalidArgument("digits", "digits are empty")
	}
	sum := 0
	for i := len(digits) - 1; i >= 0; i-- {
		char := digits[i]
		if char < '0' || char > '9' {
			return 0, invalidArgument("digits", "%q has a non digit char", digits)
		}
		weight := 1
		if (len(digits)-1-i)%2 == 0 {
			weight = 3
		}
		sum += int(char-'0') * weight
	}
	return (10 - sum%10) % 10, nil
}

// IsValidGTIN reports whether the GTIN (8, 12, 13 or 14 digits) has a correct check digit.
func IsValidGTIN(gtin string) bool {
	// ASL needs 14 digits; shorter GTINs are valid GS1 but must be padded with zeros.
	switch len(gtin) {
	case 8, 12, 13, 14:
	default:
		return false
	}
	return hasValidCheckDigit(gtin)
}

// NormalizeGTIN pads a GTIN with leading zeros to 14 digits and checks it.
func NormalizeGTIN(gtin string) (string, error) {
	// ASL searches GTIN by full match, so leading zeros are important.
	gtin = strings.TrimSpace(gtin)
	if !IsValidGTIN(gtin) {
		return "", invalidArgument("gtin", "%q is not a valid GTIN", gtin)
	}
	return strings.Repeat("0", 14-len(gtin)) + gtin, nil
}

// GenerateSSCC builds a 20 char SSCC: "00" + extension + company prefix + serial + check digit.
func GenerateSSCC(extension int, companyPrefix string, serial uint64) (string, error) {
	// companyPrefix is the GS1 company prefix or the 9 digit INN (as ASL stage accepts).
	// serial fills the rest of 16 digits; it must fit, or an error is returned.
	if extension < 0 || extension > 9 {
		return "", invalidArgument("extension", "extension must be 0..9, got %d", extension)
	}
	companyPrefix = strings.TrimSpace(companyPrefix)
	if !digitsOnly.MatchString(companyPrefix) || len(companyPrefix) < 6 || len(companyPrefix) > 12 {
		return "", invalidArgument("companyPrefix", "company prefix must be 6..12 digits, got %q",
			companyPrefix)
	}
	serialLength := ssccPayloadSz - 1 - len(companyPrefix)
	serialText := fmt.Sprintf("%0*d", serialLength, serial)
	if len(serialText) > serialLength {
		return "", invalidArgument("serial", "serial %d does not fit in %d digits", serial, serialLength)
	}
	payload := fmt.Sprintf("%d%s%s", extension, companyPrefix, serialText)
	check, err := CheckDigit(payload)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s%d", ssccAI, payload, check), nil
}

// NewSSCC builds an SSCC with a random serial (crypto/rand); check uniqueness in your database.
func NewSSCC(extension int, companyPrefix string) (string, error) {
	// The random serial uses all free digits after the company prefix.
	serialLength := ssccPayloadSz - 1 - len(strings.TrimSpace(companyPrefix))
	if serialLength < 1 {
		return "", invalidArgument("companyPrefix", "company prefix %q is too long", companyPrefix)
	}
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(serialLength)), nil)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", &Error{Code: CodeUnknown, Message: "can not make random serial: " + err.Error(),
			Err: err}
	}
	return GenerateSSCC(extension, companyPrefix, serial.Uint64())
}

// SSCCSerial returns the serial part of an SSCC made with the given company prefix.
func SSCCSerial(sscc, companyPrefix string) (uint64, error) {
	// Useful to continue a counter from the last SSCC saved in the database.
	if !IsValidSSCC(sscc) {
		return 0, invalidArgument("sscc", "%q is not a valid SSCC", sscc)
	}
	digits := strings.TrimPrefix(sscc, ssccAI)
	if len(digits) != ssccDigits || !strings.HasPrefix(digits[1:], companyPrefix) {
		return 0, invalidArgument("companyPrefix", "SSCC %q is not made with prefix %q",
			sscc, companyPrefix)
	}
	var serial uint64
	_, err := fmt.Sscanf(digits[1+len(companyPrefix):ssccDigits-1], "%d", &serial)
	if err != nil {
		return 0, invalidArgument("sscc", "can not read serial of %q", sscc)
	}
	return serial, nil
}

// IsValidSSCC reports whether the code is an SSCC (18 digits or 20 with "00") with a right check digit.
func IsValidSSCC(code string) bool {
	// ASL uses the 20 char form with leading "00".
	switch {
	case len(code) == ssccWithAI && strings.HasPrefix(code, ssccAI):
		return hasValidCheckDigit(code[len(ssccAI):])
	case len(code) == ssccDigits:
		return hasValidCheckDigit(code)
	default:
		return false
	}
}

// IsSSCC reports whether the code looks like an ASL transport code (20 digits starting with 00).
func IsSSCC(code string) bool {
	// Only the format is checked here; IsValidSSCC also checks the digit.
	return len(code) == ssccWithAI && strings.HasPrefix(code, ssccAI) && digitsOnly.MatchString(code)
}

// hasValidCheckDigit checks the last digit of a digits only text.
func hasValidCheckDigit(digits string) bool {
	// The last char is the check digit of all other chars.
	if !digitsOnly.MatchString(digits) || len(digits) < 2 {
		return false
	}
	check, err := CheckDigit(digits[:len(digits)-1])
	return err == nil && int(digits[len(digits)-1]-'0') == check
}

// NormalizeCode makes the GS separator one ASCII 29 char and removes scanner prefixes.
func NormalizeCode(raw string) string {
	// Scanners and logs show GS as "<GS>", "\u001d", "\x1d" or "\\u001D"; all become ASCII 29.
	code := strings.TrimSpace(raw)
	for _, prefix := range []string{"]d2", "]C1", "]Q3", "\xe8"} {
		code = strings.TrimPrefix(code, prefix)
	}
	replacer := strings.NewReplacer(
		"<GS>", GS, "<gs>", GS, `\u001d`, GS, `\u001D`, GS, `\x1d`, GS, `\x1D`, GS, "␝", GS,
	)
	code = replacer.Replace(code)
	return strings.TrimPrefix(code, GS)
}

// IdentityCode returns the code without crypto tail: SSCC as is, GS1 code cut at the first GS.
func IdentityCode(raw string) string {
	// Info, aggregation and owner check methods need identity codes, utilisation needs full codes.
	code := NormalizeCode(raw)
	if index := strings.Index(code, GS); index >= 0 {
		return code[:index]
	}
	return code
}

// IdentityCodes applies IdentityCode to each code and removes repeated codes, keeping the order.
func IdentityCodes(raw []string) []string {
	// Empty codes are skipped.
	seen := make(map[string]struct{}, len(raw))
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		code := IdentityCode(item)
		if _, ok := seen[code]; ok || code == "" {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	return result
}

// ParseCode reads a GS1 DataMatrix code like 01<GTIN>21<serial><GS>93<check>.
func ParseCode(raw string) (Code, error) {
	// Groups after variable length AIs end with GS or with the end of the code.
	code := NormalizeCode(raw)
	result := Code{Raw: code, Other: map[string]string{}}
	rest := code
	for rest != "" {
		ai, value, next, err := readAI(rest)
		if err != nil {
			return Code{}, invalidArgument("code", "can not parse %q: %v", raw, err)
		}
		result.set(ai, value)
		rest = next
	}
	if result.GTIN == "" || result.Serial == "" {
		return Code{}, invalidArgument("code", "%q has no GTIN (01) or serial (21)", raw)
	}
	return result, nil
}

// readAI reads one application identifier and its value from the start of text.
func readAI(text string) (ai, value, rest string, err error) {
	// Fixed AIs are checked first, then known variable AIs, then any 2 digit AI.
	text = strings.TrimPrefix(text, GS)
	for candidate, size := range fixedAIs {
		if strings.HasPrefix(text, candidate) {
			end := len(candidate) + size
			if len(text) < end {
				return "", "", "", fmt.Errorf("AI %s needs %d chars", candidate, size)
			}
			return candidate, text[len(candidate):end], text[end:], nil
		}
	}
	ai = ""
	for _, candidate := range knownVarAIs {
		if strings.HasPrefix(text, candidate) {
			ai = candidate
			break
		}
	}
	if ai == "" {
		if len(text) < 2 || !digitsOnly.MatchString(text[:2]) {
			return "", "", "", fmt.Errorf("unknown data at %q", text)
		}
		ai = text[:2]
	}
	body := text[len(ai):]
	if index := strings.Index(body, GS); index >= 0 {
		return ai, body[:index], body[index:], nil
	}
	return ai, body, "", nil
}

// set stores an AI value in the matching field.
func (c *Code) set(ai, value string) {
	// Unknown AIs go to Other, so no data is lost.
	switch ai {
	case "01":
		c.GTIN = value
	case "21":
		c.Serial = value
	case "91":
		c.CryptoKey = value
	case "92":
		c.Crypto = value
	case "93":
		c.CheckCode = value
	default:
		c.Other[ai] = value
	}
}
