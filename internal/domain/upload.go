package domain

import (
	"context"
	"io"
)

type UploadFileInput struct {
	File              io.Reader
	Filename          string
	ContentType       string
	RequestedFileName string
}

type UploadedFile struct {
	Message   string `json:"message"`
	IssuedAt  string `json:"issued_at"`
	Folder    string `json:"folder"`
	PublicURL string `json:"public_url"`
	PublicID  string `json:"public_id"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Format    string `json:"format"`
	Bytes     int64  `json:"bytes"`
}

type FileUploader interface {
	Upload(ctx context.Context, input UploadFileInput) (UploadedFile, error)
}
