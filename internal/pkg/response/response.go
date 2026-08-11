package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "ops-platform/internal/pkg/errors"
)

type Envelope struct {
	Code    apperrors.Code `json:"code"`
	Message string         `json:"message"`
	Data    any            `json:"data"`
}

func JSON(c *gin.Context, status int, code apperrors.Code, message string, data any) {
	c.JSON(status, Envelope{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func Success(c *gin.Context, data any) {
	JSON(c, http.StatusOK, apperrors.CodeOK, "ok", data)
}

func Created(c *gin.Context, data any) {
	JSON(c, http.StatusCreated, apperrors.CodeOK, "ok", data)
}

func Error(c *gin.Context, err error) {
	appErr := apperrors.From(err)
	if appErr == nil {
		JSON(c, http.StatusOK, apperrors.CodeOK, "ok", nil)
		return
	}
	JSON(c, appErr.HTTPStatus, appErr.Code, appErr.Message, nil)
}

func ErrorWithData(c *gin.Context, err error, data any) {
	appErr := apperrors.From(err)
	if appErr == nil {
		JSON(c, http.StatusOK, apperrors.CodeOK, "ok", data)
		return
	}
	JSON(c, appErr.HTTPStatus, appErr.Code, appErr.Message, data)
}
