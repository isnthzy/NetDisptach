package nic

import "errors"

var (
	ErrNICNotFound = errors.New("NIC not found")
	ErrNICDown     = errors.New("NIC is down")
	ErrNoIP        = errors.New("NIC has no IP address")
)
