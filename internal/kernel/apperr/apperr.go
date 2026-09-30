// Package apperr is EduPilot's application error model.
//
// Every failure that can reach a user interface carries a stable machine code,
// never a free-text message. The frontend maps codes to localised text, so a
// translated interface never depends on the wording of a backend string.
package apperr

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Code is a stable, machine-readable error classification.
type Code string

// Application error codes. The set is closed: adding a code is a deliberate act
// because every code has a localised message in the frontend.
const (
	CodeInsufficientPermission Code = "INSUFFICIENT_PERMISSION"
	CodeUnauthenticated        Code = "UNAUTHENTICATED"
	CodePeriodClosed           Code = "PERIOD_CLOSED"
	CodeInvoiceAlreadyPosted   Code = "INVOICE_ALREADY_POSTED"
	CodePaymentAlreadyReversed Code = "PAYMENT_ALREADY_REVERSED"
	CodeInvalidAllocation      Code = "INVALID_ALLOCATION"
	CodeUnbalancedJournal      Code = "UNBALANCED_JOURNAL"
	CodeStockInsufficient      Code = "STOCK_INSUFFICIENT"
	CodePayrollLocked          Code = "PAYROLL_LOCKED"
	CodeValidationFailed       Code = "VALIDATION_FAILED"
	CodeNotFound               Code = "NOT_FOUND"
	CodeConflict               Code = "CONFLICT"
	CodeAlreadyExists          Code = "ALREADY_EXISTS"
	CodeNetwork                Code = "NETWORK"
	CodeInternal               Code = "INTERNAL"
)

// Error is an application error with a code, a developer-facing message and
// structured details safe to serialise to a client.
type Error struct {
	code    Code
	message string
	details map[string]any
	cause   error
}

// New returns an application error with the given code and message.
func New(code Code, message string) *Error {
	return &Error{code: code, message: message}
}

// Wrap returns an application error that carries an underlying cause. The
// cause is preserved for errors.Is and errors.As but is never serialised.
func Wrap(cause error, code Code, message string) *Error {
	return &Error{code: code, message: message, cause: cause}
}

// WithDetail attaches a structured detail and returns the receiver.
func (e *Error) WithDetail(key string, value any) *Error {
	if e.details == nil {
		e.details = make(map[string]any, 2)
	}
	e.details[key] = value
	return e
}

// Code returns the machine-readable classification.
func (e *Error) Code() Code { return e.code }

// Message returns the developer-facing message.
func (e *Error) Message() string { return e.message }

// Details returns the structured details, which may be nil.
func (e *Error) Details() map[string]any { return e.details }

// Unwrap exposes the underlying cause to errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.cause }

// Error implements the error interface.
func (e *Error) Error() string {
	if e.cause == nil {
		return fmt.Sprintf("%s: %s", e.code, e.message)
	}
	return fmt.Sprintf("%s: %s: %v", e.code, e.message, e.cause)
}

// Is reports whether the error carries the given code, so callers can write
// errors.Is(err, apperr.Is(apperr.CodeNotFound)) or use the CodeOf helper.
func (e *Error) Is(target error) bool {
	other, isAppError := target.(*Error)
	return isAppError && other.code == e.code
}

// wireError is the transport shape shared with the frontend AppError type.
type wireError struct {
	Code    Code           `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// MarshalJSON emits the transport shape, never the cause.
func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(wireError{Code: e.code, Message: e.message, Details: e.details})
}

// FromError converts any error into the transport shape. Errors that are not
// application errors are reported as INTERNAL without leaking their text,
// because an unexpected message may contain a SQL fragment or a file path.
func FromError(err error) wireError {
	if err == nil {
		return wireError{Code: CodeInternal, Message: "unknown error"}
	}

	var appError *Error
	if errors.As(err, &appError) {
		return wireError{Code: appError.code, Message: appError.message, Details: appError.details}
	}

	return wireError{Code: CodeInternal, Message: "an unexpected error occurred"}
}

// CodeOf returns the code carried by an error, or CodeInternal when the error
// did not originate in the application layer.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var appError *Error
	if errors.As(err, &appError) {
		return appError.code
	}
	return CodeInternal
}

// Is reports whether err carries the given code.
func Is(err error, code Code) bool {
	return CodeOf(err) == code
}

// ValidationFailed builds the standard validation error.
func ValidationFailed(message string) *Error {
	return New(CodeValidationFailed, message)
}

// NotFound builds the standard not-found error for an entity type and id.
func NotFound(entity string, id string) *Error {
	return New(CodeNotFound, fmt.Sprintf("%s %q was not found", entity, id)).
		WithDetail("entity", entity).
		WithDetail("id", id)
}

// AlreadyExists builds the standard uniqueness violation error.
func AlreadyExists(entity string, field string, value string) *Error {
	return New(CodeAlreadyExists, fmt.Sprintf("%s with %s %q already exists", entity, field, value)).
		WithDetail("entity", entity).
		WithDetail("field", field)
}

// InsufficientPermission builds the standard authorisation failure.
func InsufficientPermission(action string, permission string) *Error {
	return New(CodeInsufficientPermission, fmt.Sprintf("permission %q is required to %s", permission, action)).
		WithDetail("action", action).
		WithDetail("permission", permission)
}

// Unauthenticated builds the standard authentication failure.
func Unauthenticated(reason string) *Error {
	return New(CodeUnauthenticated, reason)
}

// Internal builds an internal error from a cause. The cause is retained for
// logging and never returned to the client.
func Internal(cause error, message string) *Error {
	return Wrap(cause, CodeInternal, message)
}

// Describe renders an error for a log line: code, message and cause chain.
func Describe(err error) string {
	if err == nil {
		return "<nil>"
	}
	if errors.Is(err, err) {
		return strings.TrimSpace(err.Error())
	}
	return err.Error()
}
