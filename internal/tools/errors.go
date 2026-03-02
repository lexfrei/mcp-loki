package tools

import "github.com/cockroachdb/errors"

// ErrValidation indicates invalid parameters provided by the caller.
var ErrValidation = errors.New("validation error")

// ErrLokiRequest indicates a failure communicating with the Loki API.
var ErrLokiRequest = errors.New("loki request error")

// validationErr marks an error as a validation error.
func validationErr(err error) error {
	//nolint:wrapcheck // Mark adds a sentinel category, the caller already provides context.
	return errors.Mark(err, ErrValidation)
}

// lokiErr wraps a message and underlying error as a Loki request error.
func lokiErr(msg string, err error) error {
	//nolint:wrapcheck // Mark adds a sentinel category on top of Wrap which provides context.
	return errors.Mark(errors.Wrap(err, msg), ErrLokiRequest)
}
