package dto

// DocumentFilter is the search filter for documents; all fields are optional.
type DocumentFilter struct {
	DocumentID    string
	ProductGroups []ProductGroup
	Statuses      []DocumentStatus
	Types         []DocumentType // SALES_RECEIPT and REFUND_RECEIPT are not allowed
	WithWarning   *bool
	DateFrom      *DateTime
	DateTo        *DateTime
	Limit         int    // ASL default is 100
	Cursor        string // ID of the last document from the previous page
}

// DocumentInfo is a short description of a document.
type DocumentInfo struct {
	DocumentID  string         `json:"documentId"`
	Type        DocumentType   `json:"type"`
	Status      DocumentStatus `json:"status"`
	WithWarning *bool          `json:"withWarning,omitempty"` // some codes did not change state
	CreateDate  DateTime       `json:"createDate"`
}

// DocumentList is a page of documents.
type DocumentList struct {
	DocumentInfos []DocumentInfo `json:"documentInfos"`
}

// DocumentCard is the main data of one document.
type DocumentCard struct {
	DocumentInfo
	OriginalDocID string `json:"originalDocId,omitempty"` // external ID, used in invoices
	// For example customs declaration number
	DocumentNumber string       `json:"documentNumber,omitempty"`
	ProductGroup   ProductGroup `json:"productGroup,omitempty"`
}

// DocumentErrorsFilter is the page filter for document errors.
type DocumentErrorsFilter struct {
	PropertyName string // return only errors of this property, e.g. CODE
	Limit        int    // required by ASL; provider sends 100 when it is 0
	LastIndex    *int   // index of the last error from the previous page
}

// DocumentErrorList is the list of errors found in a document.
type DocumentErrorList struct {
	DocumentErrors []DocumentError `json:"documentErrors"`
}

// DocumentError is one error in a document.
type DocumentError struct {
	PropertyName string         `json:"propertyName,omitempty"`
	Index        int            `json:"index"`     // line (code) number in the document
	ErrorCode    string         `json:"errorCode"` // e.g. invalid-code-status
	ErrorTags    map[string]any `json:"errorTags,omitempty"`
}

// DocumentCodesFilter is the page filter for code states of a document.
type DocumentCodesFilter struct {
	Limit     int  // ASL default is 30 000
	LastIndex *int // index of the last code from the previous page
}

// DocumentCodeState is the processing result of one code in a document.
type DocumentCodeState struct {
	Index  int       `json:"index"`
	Code   string    `json:"code"`
	State  CodeState `json:"state"`
	Result string    `json:"result,omitempty"` // reason of ERROR or WARNING
}

// PartyStatus is info about a participant; all fields are empty when it is not found.
type PartyStatus struct {
	Tin           string         `json:"tin,omitempty"`
	Name          LocalizedText  `json:"name,omitempty"`
	FullName      LocalizedText  `json:"fullName,omitempty"`
	ProductGroups []ProductGroup `json:"productGroups,omitempty"`
}

// Found reports whether ASL knows this participant.
func (p PartyStatus) Found() bool {
	return p.Tin != ""
}
