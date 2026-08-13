package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
)

type Code int

const (
	CodeOK Code = 0

	CodeInternal           Code = 10001
	CodeInvalidArgument    Code = 10002
	CodeNotFound           Code = 10003
	CodeServiceUnavailable Code = 10004
	CodeConflict           Code = 10005

	CodeUnauthenticated  Code = 20001
	CodePermissionDenied Code = 20002
	CodeRateLimited      Code = 20003

	CodeKubernetesUnavailable Code = 30001

	CodeConfigConflict  Code = 40001
	CodeReleaseFailed   Code = 50001
	CodeLogQueryFailed  Code = 60001
	CodeAuditFailed     Code = 70001
	CodeDiagnosisFailed Code = 80001
	CodeAssetFailed     Code = 90001
)

type AppError struct {
	Code       Code
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(code Code, message string, httpStatus int) *AppError {
	if httpStatus == 0 {
		httpStatus = http.StatusInternalServerError
	}
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func Wrap(err error, code Code, message string, httpStatus int) *AppError {
	if err == nil {
		return New(code, message, httpStatus)
	}
	appErr := New(code, message, httpStatus)
	appErr.Err = err
	return appErr
}

func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr
	}
	return Wrap(err, CodeInternal, "internal server error", http.StatusInternalServerError)
}

func Segment(code Code) string {
	switch {
	case code == CodeOK:
		return "ok"
	case code >= 10000 && code < 20000:
		return "common"
	case code >= 20000 && code < 30000:
		return "auth"
	case code >= 30000 && code < 40000:
		return "kubernetes"
	case code >= 40000 && code < 50000:
		return "configcenter"
	case code >= 50000 && code < 60000:
		return "release"
	case code >= 60000 && code < 70000:
		return "logquery"
	case code >= 70000 && code < 80000:
		return "audit"
	case code >= 80000 && code < 90000:
		return "ai"
	case code >= 90000 && code < 100000:
		return "asset"
	default:
		return "unknown"
	}
}
