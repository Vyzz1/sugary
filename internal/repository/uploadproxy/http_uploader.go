package uploadproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"

	"sugary/internal/config"
	"sugary/internal/domain"
)

type HTTPUploader struct {
	apiURL        string
	internalToken string
	folder        string
	client        *http.Client
}

func NewHTTPUploader(cfg config.UploadConfig) HTTPUploader {
	return HTTPUploader{
		apiURL:        strings.TrimSpace(cfg.APIURL),
		internalToken: strings.TrimSpace(cfg.InternalToken),
		folder:        strings.TrimSpace(cfg.Folder),
		client:        &http.Client{Timeout: 30 * time.Second},
	}
}

func (u HTTPUploader) Upload(ctx context.Context, input domain.UploadFileInput) (domain.UploadedFile, error) {
	if u.apiURL == "" || u.internalToken == "" || u.folder == "" {
		return domain.UploadedFile{}, fmt.Errorf("upload proxy is not configured")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("folder", u.folder); err != nil {
		return domain.UploadedFile{}, err
	}
	if input.RequestedFileName != "" {
		if err := writer.WriteField("file_name", input.RequestedFileName); err != nil {
			return domain.UploadedFile{}, err
		}
	}

	partHeaders := make(textproto.MIMEHeader)
	partHeaders.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeQuotes(input.Filename)))
	if strings.TrimSpace(input.ContentType) != "" {
		partHeaders.Set("Content-Type", input.ContentType)
	}
	part, err := writer.CreatePart(partHeaders)
	if err != nil {
		return domain.UploadedFile{}, err
	}
	if _, err := io.Copy(part, input.File); err != nil {
		return domain.UploadedFile{}, err
	}
	if err := writer.Close(); err != nil {
		return domain.UploadedFile{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.apiURL, &body)
	if err != nil {
		return domain.UploadedFile{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-internal-upload-token", u.internalToken)

	resp, err := u.client.Do(req)
	if err != nil {
		return domain.UploadedFile{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.UploadedFile{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.UploadedFile{}, fmt.Errorf("upload API failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var result domain.UploadedFile
	if err := json.Unmarshal(raw, &result); err != nil {
		return domain.UploadedFile{}, err
	}
	return result, nil
}

func escapeQuotes(value string) string {
	name := filepath.Base(strings.TrimSpace(value))
	name = strings.ReplaceAll(name, `"`, "")
	if name == "" {
		return "upload.bin"
	}
	return name
}
