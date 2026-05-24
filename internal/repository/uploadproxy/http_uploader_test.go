package uploadproxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sugary/internal/config"
	"sugary/internal/domain"
)

func TestHTTPUploaderUpload(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("x-internal-upload-token") != "secret-token" {
			t.Fatalf("missing internal token header")
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if got := r.FormValue("folder"); got != "sugary" {
			t.Fatalf("expected folder sugary, got %q", got)
		}
		if got := r.FormValue("file_name"); got != "custom-name" {
			t.Fatalf("expected file_name custom-name, got %q", got)
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer file.Close()

		if header.Filename != "meal.jpg" {
			t.Fatalf("expected filename meal.jpg, got %q", header.Filename)
		}
		raw, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read uploaded file: %v", err)
		}
		if string(raw) != "image-bytes" {
			t.Fatalf("unexpected uploaded file body %q", string(raw))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(domain.UploadedFile{
			Message:   "File uploaded successfully",
			Folder:    "sugary",
			PublicURL: "https://example.com/file.jpg",
			PublicID:  "sugary/file",
			Format:    "jpg",
			Bytes:     11,
		})
	}))
	defer server.Close()

	uploader := NewHTTPUploader(config.UploadConfig{
		APIURL:        server.URL,
		InternalToken: "secret-token",
		Folder:        "sugary",
	})

	result, err := uploader.Upload(context.Background(), domain.UploadFileInput{
		File:              strings.NewReader("image-bytes"),
		Filename:          "meal.jpg",
		ContentType:       "image/jpeg",
		RequestedFileName: "custom-name",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Message != "File uploaded successfully" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
