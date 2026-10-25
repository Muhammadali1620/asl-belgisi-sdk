package asl

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"asl/internal/dto"
)

// Code info and export paths.
const (
	pathPublicCodes  = "/public/api/cod/public/codes"
	pathPrivateCodes = "/public/api/cod/private/codes"
	pathOwnerCheck   = "/public/api/cod/nested-codes/owner-check"
	pathVerify       = "/public/api/v1/code-verification/verify"
	pathExports      = "/public/api/cod/exports"
)

func (c *Client) GetPublicCodes(
	ctx context.Context, req dto.PublicCodesRequest,
) ([]dto.PublicCodeInfo, error) {
	// POST /public/api/cod/public/codes. Send identity codes (see IdentityCode).
	// Unknown codes are skipped by ASL, so the answer can be shorter than the request.
	if err := requireIdentityCodes("codes", req.Codes, MaxInfoCodes); err != nil {
		return nil, err
	}
	var out []dto.PublicCodeInfo
	if err := c.post(ctx, pathPublicCodes, nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetPrivateCodes(
	ctx context.Context, req dto.PrivateCodesRequest,
) (*dto.PrivateCodesResponse, error) {
	// POST /public/api/cod/private/codes. When we have no access to any code ASL answers 307;
	// then public info is loaded and returned in PublicResults with Redirected=true.
	if err := requireIdentityCodes("codes", req.Codes, MaxInfoCodes); err != nil {
		return nil, err
	}
	var out dto.PrivateCodesResponse
	err := c.post(ctx, pathPrivateCodes, nil, req, &out)
	if HasCode(err, CodeRedirect) {
		return c.privateFallback(ctx, req)
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// privateFallback loads public info when ASL redirects the private codes request.
func (c *Client) privateFallback(
	ctx context.Context, req dto.PrivateCodesRequest,
) (*dto.PrivateCodesResponse, error) {
	// All requested codes are forbidden for private info in this case.
	publicReq := dto.PublicCodesRequest{Codes: req.Codes, AddCodeHistory: req.AddCodeHistory}
	public, err := c.GetPublicCodes(ctx, publicReq)
	if err != nil {
		return nil, err
	}
	return &dto.PrivateCodesResponse{
		Results:        []dto.PrivateCodeInfo{},
		ForbiddenCodes: append([]string(nil), req.Codes...),
		Redirected:     true,
		PublicResults:  public,
	}, nil
}

func (c *Client) CheckCodeOwner(
	ctx context.Context, req dto.OwnerCheckRequest,
) (*dto.OwnerCheckResponse, error) {
	// POST /public/api/cod/nested-codes/owner-check (max 100 codes). Data is returned only
	// when the TIN is the current owner; other codes go to ForbiddenCodes or MissingCodes.
	if err := requireIdentityCodes("codes", req.Codes, MaxOwnerCheckCodes); err != nil {
		return nil, err
	}
	if err := requireTIN("ownerTin", req.OwnerTin); err != nil {
		return nil, err
	}
	var out dto.OwnerCheckResponse
	if err := c.post(ctx, pathOwnerCheck, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) VerifyCodes(ctx context.Context, codes []string) ([]dto.VerifyResult, error) {
	// POST /public/api/v1/code-verification/verify with a JSON array of full codes (with GS).
	// Works without owner access. Transport codes (SSCC) are always not verified.
	if err := requireCodes("codes", codes, MaxInfoCodes); err != nil {
		return nil, err
	}
	var out []dto.VerifyResult
	if err := c.post(ctx, pathVerify, nil, codes, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateExport(ctx context.Context, req dto.ExportRequest) (*dto.ExportResponse, error) {
	// POST /public/api/cod/exports. ASL makes a zip file with codes we own (identity codes).
	// File is up to 10 MB and lives 60 minutes; a too big list ends with ERROR status.
	if err := requireGTIN("gtin", req.GTIN); err != nil {
		return nil, err
	}
	ranges := []struct {
		name     string
		from, to *dto.DateTime
	}{
		{"emissionDate", req.EmissionDateFrom, req.EmissionDateTo},
		{"productionDate", req.ProductionDateFrom, req.ProductionDateTo},
		{"expirationDate", req.ExpirationDateFrom, req.ExpirationDateTo},
	}
	for _, item := range ranges {
		if err := checkDateRange(item.name, item.from, item.to); err != nil {
			return nil, err
		}
	}
	var out dto.ExportResponse
	if err := c.post(ctx, pathExports, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetExportStatus(ctx context.Context, exportID string) (dto.ExportStatus, error) {
	// GET /public/api/cod/exports/{id}/status. The answer is a plain JSON string like "SUCCESS".
	id, err := pathID("exportId", exportID)
	if err != nil {
		return "", err
	}
	var status dto.ExportStatus
	if err := c.get(ctx, pathExports+"/"+id+"/status", nil, &status); err != nil {
		return "", err
	}
	return dto.ExportStatus(strings.TrimSpace(string(status))), nil
}

func (c *Client) WaitExport(
	ctx context.Context, exportID string, interval time.Duration,
) (dto.ExportStatus, error) {
	// Asks the status every interval (default 2s) until SUCCESS, ERROR or EXPIRED.
	// Stops with TIMEOUT or CANCELED error when ctx ends.
	interval = valueOr(interval, defaultPollInterval)
	for {
		status, err := c.GetExportStatus(ctx, exportID)
		if err != nil {
			return "", err
		}
		if status.IsFinal() {
			return status, nil
		}
		if waitErr := sleep(ctx, interval); waitErr != nil {
			return status, transportError(ctx, http.MethodGet, pathExports, waitErr)
		}
	}
}

func (c *Client) DownloadExport(ctx context.Context, exportID string) ([]byte, error) {
	// GET /public/api/cod/exports/{id}/result. Returns zip bytes; read them with ReadExportArchive.
	id, err := pathID("exportId", exportID)
	if err != nil {
		return nil, err
	}
	return c.download(ctx, pathExports+"/"+id+"/result")
}

// ReadExportArchive unpacks the export zip; each file has JSON in the private codes format.
func ReadExportArchive(archive []byte) ([]dto.ExportFile, error) {
	// Folders are skipped. Each file is limited by maxBodySize to protect from zip bombs.
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		message := "export is not a zip file: " + err.Error()
		return nil, &Error{Code: CodeDecode, Message: message, Err: err}
	}
	files := make([]dto.ExportFile, 0, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		data, err := readZipFile(file)
		if err != nil {
			message := "can not read " + file.Name + ": " + err.Error()
			return nil, &Error{Code: CodeDecode, Message: message, Err: err}
		}
		files = append(files, dto.ExportFile{Name: file.Name, Data: data})
	}
	return files, nil
}

// ReadExportCodes unpacks the export zip and reads codes from all JSON files.
func ReadExportCodes(archive []byte) ([]dto.PrivateCodeInfo, error) {
	// Each file is {"results": [...]} like the private codes answer.
	files, err := ReadExportArchive(archive)
	if err != nil {
		return nil, err
	}
	var codes []dto.PrivateCodeInfo
	for _, file := range files {
		var content dto.PrivateCodesResponse
		if err := json.Unmarshal(file.Data, &content); err != nil {
			message := "file " + file.Name + " is not valid JSON: " + err.Error()
			return nil, &Error{Code: CodeDecode, Message: message, Err: err}
		}
		codes = append(codes, content.Results...)
	}
	return codes, nil
}

// readZipFile reads one file from a zip archive.
func readZipFile(file *zip.File) ([]byte, error) {
	// The reader is closed in all cases.
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	return io.ReadAll(io.LimitReader(stream, maxBodySize))
}
