package switch_driver

import "errors"

var (
	ErrInvalidSwitchDriver  = errors.New("invalid switch driver")
	ErrNotTurnable          = errors.New("not turnable")
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
)
