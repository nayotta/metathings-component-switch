package switch_driver

import "errors"

var (
	ErrInvalidSwitchDriver = errors.New("invalid switch driver")
	ErrNotStartable        = errors.New("not startable")
	ErrNotStopable         = errors.New("not stopable")
)
