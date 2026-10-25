package dto

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// Known product card attribute codes.
const (
	AttrNestedProductsCount = "nested_products_count" // capacity of a group package
	AttrNestedProduct       = "nested_product"        // GTIN inside a group package
	AttrExpirationDate      = "expiration_date"       // shelf life, unit is in Attribute.Unit
	AttrNominalQuantity     = "nominal_quantity"      // net amount of product in package
	AttrProductName         = "product_name"
	AttrCountry             = "country"
)

// Product card attribute types.
const (
	AttrTypeString          = "STRING"
	AttrTypeStringLocalized = "STRING_LOCALIZED"
	AttrTypeInteger         = "INTEGER"
	AttrTypeDouble          = "DOUBLE"
	AttrTypeBoolean         = "BOOLEAN"
	AttrTypeDate            = "DATE"
	AttrTypeEnum            = "ENUM"
	AttrTypeEnumList        = "ENUM_LIST"
	AttrTypeFileList        = "FILE_LIST"
	AttrTypeInnerGTIN       = "INNER_GTIN"
	AttrTypeLocalizedList   = "LOCALIZED_LIST"
)

// ProductFilter is the filter for the product card list; ProductGroup is required.
type ProductFilter struct {
	ProductGroup  ProductGroup
	Inn           string
	GTIN          string
	Status        ProductCardStatus
	Tnved         string
	UpdatedAtFrom *DateTime
	UpdatedAtTo   *DateTime
	PackageType   PackageType
}

// CodeName is a value code and its name in many languages.
type CodeName struct {
	Code string        `json:"code"`
	Name LocalizedText `json:"name,omitempty"`
}

// ValueName is a value code and its name in many languages (search by GTIN format).
type ValueName struct {
	Value string        `json:"value"`
	Name  LocalizedText `json:"name,omitempty"`
}

// ProductSummary is a product card in the list answer.
type ProductSummary struct {
	ID              string        `json:"id"`
	GTIN            string        `json:"gtin"`
	ProductName     LocalizedText `json:"productName,omitempty"`
	Inn             string        `json:"inn"`
	ProductGroup    CodeName      `json:"productGroup"`
	ProductCategory CodeName      `json:"productCategory"`
	Tnved           CodeName      `json:"tnved"`
	PackageType     CodeName      `json:"packageType"`
	Status          CodeName      `json:"status"`
	CreatedAt       *DateTime     `json:"createdAt,omitempty"`
	UpdatedAt       *DateTime     `json:"updatedAt,omitempty"`
}

// SearchByGTINFilter is the filter for search by GTIN; GTIN is required.
type SearchByGTINFilter struct {
	GTIN         string
	ProductGroup ProductGroup
	Status       ProductCardStatus
	Limit        int // ASL default is 20
	Offset       int
}

// SearchByGTINResponse is a page of product cards with the given GTIN.
type SearchByGTINResponse struct {
	Products    []ProductSearchItem `json:"products"`
	HasNextPage bool                `json:"hasNextPage"`
}

// ProductSearchItem is a product card in the search by GTIN answer.
type ProductSearchItem struct {
	ID                string          `json:"id"`
	GTIN              string          `json:"gtin"`
	Tnved             string          `json:"tnved"`
	ProductName       LocalizedText   `json:"productName,omitempty"`
	Inn               string          `json:"inn"`
	ParticipantName   LocalizedText   `json:"participantName,omitempty"`
	ProductGroup      ValueName       `json:"pg"`
	ProductCategory   ValueName       `json:"productCategory"`
	PackageType       ValueName       `json:"packageType"` // value is lower case here: "unit", "group"
	Status            ValueName       `json:"status"`
	LanguagesComplete map[string]bool `json:"languagesComplete,omitempty"`
}

// ProductCard is the full product card with all attributes.
type ProductCard struct {
	ProductSummary
	Scope      ProductScope       `json:"scope"`
	Attributes []ProductAttribute `json:"attributes"`
}

// ProductAttribute is one attribute of a product card; value format depends on Meta.Type.
type ProductAttribute struct {
	Meta  AttributeMeta   `json:"meta"`
	Value json.RawMessage `json:"value"`
	Unit  *CodeName       `json:"unit,omitempty"` // unit of measure, e.g. MON or GRM
}

// AttributeMeta is the code, name and type of an attribute.
type AttributeMeta struct {
	Code string        `json:"code"`
	Name LocalizedText `json:"name,omitempty"`
	Type string        `json:"type"`
}

// Attribute returns the first attribute with this code; ok is false when it is missing or null.
func (p ProductCard) Attribute(code string) (ProductAttribute, bool) {
	for _, attr := range p.Attributes {
		if attr.Meta.Code == code && !attr.IsNull() {
			return attr, true
		}
	}
	return ProductAttribute{}, false
}

// IntAttribute returns an integer attribute (for example nested_products_count).
func (p ProductCard) IntAttribute(code string) (int, bool) {
	attr, ok := p.Attribute(code)
	if !ok {
		return 0, false
	}
	return attr.Int()
}

// StringAttribute returns a text attribute (for example nested_product).
func (p ProductCard) StringAttribute(code string) (string, bool) {
	attr, ok := p.Attribute(code)
	if !ok {
		return "", false
	}
	return attr.String()
}

// ProductFile is one file from a FILE_LIST attribute.
type ProductFile struct {
	FileName    string `json:"fileName"` // name with extension, e.g. "075bd75e-...-4353.jpg"
	HashType    string `json:"hashType,omitempty"`
	Signature   string `json:"signature,omitempty"` // file hash
	Description string `json:"description,omitempty"`
}

// ID returns the file ID for GetProductFile: the file name without extension.
func (f ProductFile) ID() string {
	name := f.FileName
	if dot := strings.LastIndex(name, "."); dot > 0 {
		name = name[:dot]
	}
	return name
}

// Files returns files of a FILE_LIST attribute; it is empty for other types.
func (a ProductAttribute) Files() []ProductFile {
	var value struct {
		Files []ProductFile `json:"files"`
	}
	if a.IsNull() || json.Unmarshal(a.Value, &value) != nil {
		return nil
	}
	return value.Files
}

// AllFiles returns files of all FILE_LIST attributes of the card.
func (p ProductCard) AllFiles() []ProductFile {
	var files []ProductFile
	for _, attr := range p.Attributes {
		if attr.Meta.Type == AttrTypeFileList {
			files = append(files, attr.Files()...)
		}
	}
	return files
}

// IsNull reports whether the attribute has no value.
func (a ProductAttribute) IsNull() bool {
	text := strings.TrimSpace(string(a.Value))
	return text == "" || text == "null"
}

// Int returns the value as integer; ok is false when the value is not a whole number.
func (a ProductAttribute) Int() (int, bool) {
	var number float64
	if err := json.Unmarshal(a.Value, &number); err != nil {
		text, ok := a.String()
		if !ok {
			return 0, false
		}
		parsed, err := strconv.Atoi(text)
		return parsed, err == nil
	}
	if number != math.Trunc(number) {
		return 0, false
	}
	return int(number), true
}

// Float returns the value as number.
func (a ProductAttribute) Float() (float64, bool) {
	var number float64
	if err := json.Unmarshal(a.Value, &number); err != nil {
		return 0, false
	}
	return number, true
}

// String returns the value as text; ok is false when the value is not a JSON string.
func (a ProductAttribute) String() (string, bool) {
	var text string
	if err := json.Unmarshal(a.Value, &text); err != nil {
		return "", false
	}
	return text, true
}

// Localized returns the value as localized text (for STRING_LOCALIZED and ENUM types).
func (a ProductAttribute) Localized() (LocalizedText, bool) {
	var text LocalizedText
	if err := json.Unmarshal(a.Value, &text); err != nil {
		return nil, false
	}
	return text, true
}
