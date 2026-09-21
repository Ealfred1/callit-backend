// Package respond is the one place every handler goes through to write a
// response, so every route returns the same two envelope shapes instead of
// each handler inventing its own {"error": "..."} vs {"message","token"} vs
// raw-object shape (that was mollah-backend's ad hoc style; Callit's client
// only has to learn one error vocabulary).
package respond

import (
	"errors"
	"net/http"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/labstack/echo/v4"
)

type envelope struct {
	Data any `json:"data,omitempty"`
}

type errorBody struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Fields  map[string][]string `json:"fields,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

func OK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, envelope{Data: data})
}

func Created(c echo.Context, data any) error {
	return c.JSON(http.StatusCreated, envelope{Data: data})
}

func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Err writes the unified error envelope directly.
func Err(c echo.Context, status int, code, message string, fields map[string][]string) error {
	return c.JSON(status, errorEnvelope{Error: errorBody{Code: code, Message: message, Fields: fields}})
}

// AppError is for validation/business errors raised inside our own handlers
// and services (as opposed to domain.PantaError, which is a passthrough of
// Panta's own failure).
type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string][]string
}

func (e *AppError) Error() string { return e.Message }

func NewAppError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func BadRequest(code, message string) *AppError {
	return NewAppError(http.StatusBadRequest, code, message)
}

func NotFound(code, message string) *AppError {
	return NewAppError(http.StatusNotFound, code, message)
}

func Unauthorized(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// FromError inspects the error's concrete type and writes the matching
// envelope: a domain.PantaError keeps Panta's own code/status (so a
// QUOTE_STALE from Panta reaches the client as QUOTE_STALE, not a generic
// 500), an *AppError keeps ours, anything else becomes a 500 INTERNAL_ERROR.
func FromError(c echo.Context, err error) error {
	var pantaErr *domain.PantaError
	if errors.As(err, &pantaErr) {
		return Err(c, pantaErr.HTTPStatus, pantaErr.Code, pantaErr.Message, pantaErr.Fields)
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return Err(c, appErr.Status, appErr.Code, appErr.Message, appErr.Fields)
	}

	var echoErr *echo.HTTPError
	if errors.As(err, &echoErr) {
		msg, _ := echoErr.Message.(string)
		if msg == "" {
			msg = http.StatusText(echoErr.Code)
		}
		return Err(c, echoErr.Code, "HTTP_ERROR", msg, nil)
	}

	return Err(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
}

// GlobalErrorHandler is registered as echo.Echo.HTTPErrorHandler (see
// cmd/api/main.go) as a safety net: any error that reaches Echo without a
// handler having already called FromError/Err itself (a route panic Recover
// catches, Echo's own router 404/405, a future handler that forgets to go
// through this package) still comes out as the same envelope instead of
// Echo's bare default `{"message": "..."}` shape. This is what caught (and
// now prevents regressing) the JWT middleware returning a bare *AppError
// instead of writing it — see git history on auth_middleware.go.
func GlobalErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	if writeErr := FromError(c, err); writeErr != nil {
		c.Logger().Error(writeErr)
	}
}
