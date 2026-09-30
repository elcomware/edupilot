// Package api holds the wire types shared by the Wails and HTTP transports.
//
// The shapes here are the contract the frontend compiles against
// (frontend/src/gateway/types). Field names and JSON tags are therefore part of
// the public API: renaming one breaks the interface in every language.
package api

import (
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	campusdomain "github.com/elcomware/edupilot/internal/platform/campus/domain"
	organisationdomain "github.com/elcomware/edupilot/internal/platform/organisation/domain"
)

// Result mirrors the frontend Result<T> union: {ok: true, value} or
// {ok: false, error}. A transport never returns a bare value or throws, so the
// interface can render a localised message for any code.
type Result[T any] struct {
	OK    bool      `json:"ok"`
	Value *T        `json:"value,omitempty"`
	Error *AppError `json:"error,omitempty"`
}

// AppError is the serialisable error shape. It carries a code, never a cause.
type AppError struct {
	Code    apperr.Code    `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Error makes a failure envelope a Go error, so the HTTP transport can log it and
// tests can assert on the code without reaching through the envelope by hand.
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Code) + ": " + e.Message
}

// Unwrap returns the value of a successful result, or the failure envelope as an
// error. The frontend reads the JSON fields directly; this is for Go callers.
func (r Result[T]) Unwrap() (T, error) {
	if !r.OK || r.Value == nil {
		var zero T
		if r.Error == nil {
			return zero, apperr.New(apperr.CodeInternal, "the transport returned neither a value nor an error")
		}
		return zero, r.Error
	}
	return *r.Value, nil
}

// OK wraps a successful value.
func OK[T any](value T) Result[T] {
	return Result[T]{OK: true, Value: &value}
}

// Fail converts an error into the transport failure shape. Non-application
// errors become INTERNAL without their text, so a SQL fragment or a file path
// never reaches a user's screen.
func Fail[T any](err error) Result[T] {
	wire := apperr.FromError(err)
	return Result[T]{OK: false, Error: &AppError{Code: wire.Code, Message: wire.Message, Details: wire.Details}}
}

// PageRequest is the shared paging request.
type PageRequest struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Search string `json:"search,omitempty"`
	Sort   string `json:"sort,omitempty"`
}

// Page is the shared paged envelope.
type Page[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

// DefaultLimit and MaxLimit bound a page so one request cannot ask for an
// unbounded result set from a school laptop.
const (
	DefaultLimit = 50
	MaxLimit     = 200
)

// Normalize clamps a page request into a safe range.
func (r PageRequest) Normalize() PageRequest {
	normalised := r
	if normalised.Limit <= 0 {
		normalised.Limit = DefaultLimit
	}
	if normalised.Limit > MaxLimit {
		normalised.Limit = MaxLimit
	}
	if normalised.Offset < 0 {
		normalised.Offset = 0
	}
	return normalised
}

// NewPage cuts a page out of a full result set. The total is reported before
// the window is applied, so a table can show "1-50 of 812".
func NewPage[T any](all []T, request PageRequest) Page[T] {
	normalised := request.Normalize()

	start := normalised.Offset
	if start > len(all) {
		start = len(all)
	}
	end := start + normalised.Limit
	if end > len(all) {
		end = len(all)
	}

	window := all[start:end]
	if window == nil {
		window = []T{}
	}
	return Page[T]{Items: window, Total: int64(len(all))}
}

// OrganisationView is the organisation projection sent to the frontend.
type OrganisationView struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	LegalName     string `json:"legalName"`
	Currency      string `json:"currency"`
	Country       string `json:"country"`
	DefaultLocale string `json:"defaultLocale"`
}

// NewOrganisationView maps the domain aggregate to the wire shape.
func NewOrganisationView(organisation *organisationdomain.Organisation) OrganisationView {
	return OrganisationView{
		ID:            organisation.ID.String(),
		Name:          organisation.Name,
		LegalName:     organisation.LegalName,
		Currency:      organisation.Currency.Code(),
		Country:       organisation.Country,
		DefaultLocale: organisation.DefaultLocale,
	}
}

// CampusView is the campus projection sent to the frontend.
type CampusView struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisationId"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	Address        string `json:"address"`
}

// NewCampusView maps the domain aggregate to the wire shape.
func NewCampusView(campus *campusdomain.Campus) CampusView {
	return CampusView{
		ID:             campus.ID.String(),
		OrganisationID: campus.OrganisationID.String(),
		Name:           campus.Name,
		Code:           campus.Code,
		Address:        campus.Address.Line1,
	}
}

// NewCampusViews maps a slice of campuses to the wire shape.
func NewCampusViews(campuses []*campusdomain.Campus) []CampusView {
	views := make([]CampusView, 0, len(campuses))
	for _, campus := range campuses {
		views = append(views, NewCampusView(campus))
	}
	return views
}

// ParseID parses an identifier received from a client. A malformed identifier
// is a client bug, not a missing record.
func ParseID(value string) (ids.UUID, bool) {
	identifier, err := ids.Parse(value)
	if err != nil {
		return ids.UUID{}, false
	}
	return identifier, true
}

// InvalidID is the failure for an identifier a client could not have meant. It is
// a validation failure rather than a lookup miss, because a malformed identifier
// is a bug in the caller and a "not found" would send them looking for a record
// that was never there.
func InvalidID(field, value string) error {
	return apperr.ValidationFailed("that identifier is not valid").
		WithDetail("field", field).
		WithDetail("value", value)
}

// NotFoundEntity is the failure for a record the caller may not see. A record in
// another organisation is reported the same way as one that does not exist, so a
// caller cannot probe for identifiers belonging to another school.
func NotFoundEntity(entity, id string) error {
	return apperr.NotFound(entity, id)
}

// ParseOptionalID parses an identifier that the caller may legitimately omit. An
// empty string yields a nil identifier, which the application layer reads as
// "the current academic year" rather than as an error.
func ParseOptionalID(field, value string) (ids.UUID, error) {
	if value == "" {
		return ids.UUID{}, nil
	}
	identifier, ok := ParseID(value)
	if !ok {
		return ids.UUID{}, InvalidID(field, value)
	}
	return identifier, nil
}
