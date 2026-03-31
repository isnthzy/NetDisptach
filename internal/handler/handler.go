package handler

import (
	"netdispatch/internal/connmgr"
	"netdispatch/internal/egress"
	"netdispatch/internal/nic"
)

// Context provides context for handling requests
type Context struct {
	ConnMgr    *connmgr.Manager
	NICManager *nic.Manager
	EgressMgr  *egress.Manager
	Dialer     *nic.Dialer
}

// Handler interface for protocol handlers
type Handler interface {
	Handle(conn interface{})
}
