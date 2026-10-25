// Package test checks the ASL provider.
//
// Offline tests (TestUnit*) use a local HTTP/2 mock server and need no network.
// Live tests (TestLive*) call the ASL stage server; they are skipped when ASL_API_KEY is empty.
//
//	go test ./test/ -run TestUnit -v                 # offline only
//	go test ./test/ -run TestLive -v -count=1        # all live tests in file order
//	go test ./test/ -v -count=1                      # everything
//	go test ./test/ -run 'TestLiveOrders/close' -v   # one case, fixtures are made on demand
package test

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"

	"asl/internal/dto"
	"asl/internal/provider/asl"
)

// testConfig holds settings for live tests.
type testConfig struct {
	BaseURL         string
	Token           string
	TIN             string
	ProductGroup    dto.ProductGroup
	BusinessPlaceID int
	GTIN            string // GTIN of a consumer (UNIT) product
	GroupGTIN       string // GTIN of a group (GROUP) product that holds GTIN
	TechLogin       string
	TechPassword    string
	AllowKeyRefresh bool // allows the test that REPLACES the real API key
}

// cfg is filled once in TestMain.
var cfg testConfig

// TestMain loads test/.env (if it exists) and the environment, then runs the tests.
func TestMain(m *testing.M) {
	loadEnvFile(".env")
	cfg = testConfig{
		BaseURL:         envOr("ASL_BASE_URL", asl.StageURL),
		Token:           os.Getenv("ASL_API_KEY"),
		TIN:             envOr("ASL_TIN", "312458894"),
		ProductGroup:    dto.ProductGroup(envOr("ASL_PRODUCT_GROUP", string(dto.ProductGroupFertilizers))),
		BusinessPlaceID: envInt("ASL_BUSINESS_PLACE_ID", 734),
		GTIN:            envOr("ASL_GTIN", "04605246003585"),
		GroupGTIN:       envOr("ASL_GROUP_GTIN", "08001090970459"),
		TechLogin:       os.Getenv("ASL_TECH_LOGIN"),
		TechPassword:    os.Getenv("ASL_TECH_PASSWORD"),
		AllowKeyRefresh: os.Getenv("ASL_ALLOW_KEY_REFRESH") == "1",
	}
	os.Exit(m.Run())
}

// loadEnvFile reads KEY=VALUE lines; variables that are already set are not changed.
func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key, value = strings.TrimSpace(key), strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

// envOr returns the variable value or the fallback.
func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// envInt returns the variable as number or the fallback.
func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
