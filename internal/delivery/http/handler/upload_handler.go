package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	httpresponse "sugary/internal/delivery/http/response"
	"sugary/internal/domain"
)

type uploadFileUseCase interface {
	Execute(ctx context.Context, input domain.UploadFileInput) (domain.UploadedFile, error)
}

type UploadHandler struct {
	uploadFile uploadFileUseCase
}

func NewUploadHandler(uploadFile uploadFileUseCase) UploadHandler {
	return UploadHandler{uploadFile: uploadFile}
}

func (h UploadHandler) Upload(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "missing_file", "file is required"))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_file", "unable to read file"))
		return
	}
	defer file.Close()

	result, err := h.uploadFile.Execute(ctx.Request.Context(), domain.UploadFileInput{
		File:              file,
		Filename:          fileHeader.Filename,
		ContentType:       fileHeader.Header.Get("Content-Type"),
		RequestedFileName: ctx.PostForm("file_name"),
	})
	if err != nil {
		status := http.StatusInternalServerError
		code := "upload_failed"
		if errors.Is(err, domain.ErrInvalidUpload) {
			status = http.StatusBadRequest
			code = "invalid_upload_input"
		}
		ctx.JSON(status, httpresponse.Fail(ctx, code, err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, result))
}
