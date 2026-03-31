package egress

import "errors"

var (
	ErrPolicyNotFound       = errors.New("egress policy not found")
	ErrConnectionRejected   = errors.New("connection rejected by rule")
	ErrNoDefaultPolicy      = errors.New("no default egress policy configured")
	ErrNoPolicyAvailable    = errors.New("no egress policy available")
)
