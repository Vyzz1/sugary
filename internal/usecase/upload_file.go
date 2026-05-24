package usecase

import (
	"context"
	"strings"

	"sugary/internal/domain"
)

type UploadFile struct {
	uploader domain.FileUploader
}

func NewUploadFile(uploader domain.FileUploader) UploadFile {
	return UploadFile{uploader: uploader}
}

func (uc UploadFile) Execute(ctx context.Context, input domain.UploadFileInput) (domain.UploadedFile, error) {
	if input.File == nil {
		return domain.UploadedFile{}, domain.ErrInvalidUpload
	}
	input.Filename = strings.TrimSpace(input.Filename)
	if input.Filename == "" {
		return domain.UploadedFile{}, domain.ErrInvalidUpload
	}
	input.RequestedFileName = strings.TrimSpace(input.RequestedFileName)

	return uc.uploader.Upload(ctx, input)
}
