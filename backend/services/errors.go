package services

import "errors"

// UserError is a problem with the request, with a message safe to show the
// user as is. Handlers turn it into a problem-details response with Status;
// any other error is a 500 with a generic message.
type UserError struct {
	Status int
	Msg    string
}

func (e *UserError) Error() string { return e.Msg }

// NotFound is a 404 user error.
func NotFound(msg string) error { return &UserError{Status: 404, Msg: msg} }

// Invalid is a 422 user error.
func Invalid(msg string) error { return &UserError{Status: 422, Msg: msg} }

// AsUserError unwraps a UserError.
func AsUserError(err error) (*UserError, bool) {
	var ue *UserError
	ok := errors.As(err, &ue)
	return ue, ok
}
