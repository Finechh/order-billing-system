package errorsx

import "net/http"

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *AppError) Error() string {
	return e.Message
}

func ErrNotFound(msg string) *AppError {
	if msg == "" {
		msg = "resource not found"
	}
	return &AppError{Code: "ERR_NOT_FOUND", Message: msg, Status: http.StatusNotFound}
}

func ErrInvalidInput(msg string) *AppError {
	if msg == "" {
		msg = "invalid input"
	}
	return &AppError{Code: "ERR_INVALID_INPUT", Message: msg, Status: http.StatusBadRequest}
}

func ErrInternalError(msg string) *AppError {
	if msg == "" {
		msg = "internal server error"
	}
	return &AppError{Code: "ERR_INTERNAL", Message: msg, Status: http.StatusInternalServerError}
}

func ErrInvalidOrderState(msg string) *AppError {
	if msg == "" {
		msg = "invalid order state"
	}
	return &AppError{Code: "ERR_INVALID_ORDER_STATE", Message: msg, Status: http.StatusConflict}
}
