// Package apperr defines the domain error model shared by all modules.
// Business code classifies errors by Kind; only the HTTP boundary maps a
// Kind to a status code (see internal/platform/httpx), so services and
// repositories stay transport-agnostic. Each module exposes its domain
// error variables built with New (e.g. his.ErrVisitNotFound) and adds
// context with Wrapf, which never changes the classification.
package apperr

import (
	"errors"
	"fmt"
)

// Kind classifies an error for transport mapping and control flow.
type Kind int

const (
	// KindInvalid is malformed input or a validation failure (HTTP 400).
	KindInvalid Kind = iota + 1
	// KindUnauthorized is a missing or invalid credential (HTTP 401).
	KindUnauthorized
	// KindForbidden is an authenticated caller lacking permission (HTTP 403).
	KindForbidden
	// KindNotFound is a resource that does not exist (HTTP 404).
	KindNotFound
	// KindConflict is a state conflict such as a duplicate (HTTP 409).
	KindConflict
	// KindUpstream is a dependent external system failure (HTTP 502).
	KindUpstream
	// KindInternal is an unexpected internal failure (HTTP 500).
	KindInternal
)

func (k Kind) String() string {
	switch k {
	case KindInvalid:
		return "invalid"
	case KindUnauthorized:
		return "unauthorized"
	case KindForbidden:
		return "forbidden"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindUpstream:
		return "upstream"
	case KindInternal:
		return "internal"
	default:
		return "unknown"
	}
}

// HTTPStatus maps a Kind to the status the HTTP transport must return.
// It lives here so every module's handler maps errors identically.
func (k Kind) HTTPStatus() int {
	switch k {
	case KindInvalid:
		return 400
	case KindUnauthorized:
		return 401
	case KindForbidden:
		return 403
	case KindNotFound:
		return 404
	case KindConflict:
		return 409
	case KindUpstream:
		return 502
	default:
		return 500
	}
}

// Error is a classified domain error. Msg is safe to expose to clients for
// all kinds except KindInternal; Err is the optional cause kept for logs and
// unwrapping.
type Error struct {
	Kind Kind
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	return e.Msg + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// New creates a domain error variable, e.g.
//
//	var ErrVisitNotFound = apperr.New(apperr.KindNotFound, "visit not found")
func New(kind Kind, msg string) *Error {
	return &Error{Kind: kind, Msg: msg}
}

// Wrapf creates a classified error carrying err as its cause. The kind must
// be stated explicitly so classification is visible at the call site even
// when the cause carries no classification of its own.
func Wrapf(kind Kind, err error, format string, args ...any) *Error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...), Err: err}
}

// KindOf returns the Kind of the first *Error in err's chain. Unclassified
// errors (nil, or plain errors such as driver failures) are internal.
func KindOf(err error) Kind {
	for err != nil {
		if e, ok := err.(*Error); ok {
			return e.Kind
		}
		err = errors.Unwrap(err)
	}
	return KindInternal
}
