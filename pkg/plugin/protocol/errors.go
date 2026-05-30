package protocol

import (
	"errors"
	"fmt"
)

// Error codes are part of the wire contract. Hosts and plugins must agree on
// these strings.
const (
	ErrorAuthRequired           = "auth_required"
	ErrorNotApplicable          = "not_applicable"
	ErrorUnsupportedCapability  = "unsupported_capability"
	ErrorInvalidRequest         = "invalid_request"
	ErrorRemoteRateLimited      = "remote_rate_limited"
	ErrorRemoteValidationFailed = "remote_validation_failed"
	ErrorNetwork                = "network_error"
	ErrorPartialPublishUnknown  = "partial_publish_unknown"
	ErrorInternal               = "internal_error"
)

// Error is the structured protocol error sent over JSON-lines.
type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Message
}

// NewError creates a protocol error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// NewErrorf creates a protocol error with a formatted message.
func NewErrorf(code, format string, args ...any) *Error {
	return NewError(code, fmt.Sprintf(format, args...))
}

// AsError unwraps err to a *protocol.Error, returning nil if it is not a
// protocol error. This is used by the server loop to preserve typed errors
// across the method dispatch boundary.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var pe *Error
	if errors.As(err, &pe) {
		return pe
	}
	return nil
}
