package dto

// ExportRequest asks ASL to make a file with our codes; only GTIN is required.
type ExportRequest struct {
	GTIN                  string         `json:"gtin"` // full GTIN with leading zeros
	ProductID             string         `json:"productId,omitempty"`
	ProductName           string         `json:"productName,omitempty"` // exact full name
	ProductGroupID        []int          `json:"productGroupId,omitempty"`
	PackageType           []PackageType  `json:"packageType,omitempty"`
	EmissionDateFrom      *DateTime      `json:"emissionDateFrom,omitempty"`
	EmissionDateTo        *DateTime      `json:"emissionDateTo,omitempty"`
	ProductionDateFrom    *DateTime      `json:"productionDateFrom,omitempty"`
	ProductionDateTo      *DateTime      `json:"productionDateTo,omitempty"`
	ExpirationDateFrom    *DateTime      `json:"expirationDateFrom,omitempty"`
	ExpirationDateTo      *DateTime      `json:"expirationDateTo,omitempty"`
	Status                []CodeStatus   `json:"status,omitempty"`
	ProductStatus         string         `json:"productStatus,omitempty"`
	EmissionType          []EmissionType `json:"emissionType,omitempty"`
	OriginalReleaseMethod []ReleaseType  `json:"originalReleaseMethod,omitempty"`
	// For group and transport packages
	EmptyPackage  *bool  `json:"emptyPackage,omitempty"`
	ProductSeries string `json:"productSeries,omitempty"`
}

// ExportResponse is the ID of the export task.
type ExportResponse struct {
	ID string `json:"id"`
}

// ExportFile is one file from the export zip archive.
type ExportFile struct {
	Name string // file name inside the archive
	Data []byte // raw file content (JSON in the private codes format)
}
