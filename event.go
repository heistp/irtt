package irtt

import (
	"fmt"
	"net"
)

// Code uniquely identifies events and errors to improve context.
type Code int

//go:generate stringer -type=Code

// Server event codes.
const (
	MultipleAddresses Code = iota + 1*1024
	ServerStart
	ServerStop
	ListenerStart
	ListenerStop
	ListenerError
	Drop
	NewConn
	OpenClose
	CloseConn
	NoDSCPSupport
	ExceededDuration
	NoReceiveDstAddrSupport
	RemoveNoConn
	InvalidServerFill
)

// Client event codes.
const (
	Connecting Code = iota + 2*1024
	Connected
	WaitForPackets
	ServerRestriction
	NoTest
	ConnectedClosed
)

// Event is an event sent to a Handler.
type Event struct {
	Code       Code
	LocalAddr  *net.UDPAddr
	RemoteAddr *net.UDPAddr
	format     string
	Detail     []interface{}
}

// Eventf returns a new event.
func Eventf(code Code, laddr *net.UDPAddr, raddr *net.UDPAddr, format string,
	detail ...interface{}) *Event {
	return &Event{code, laddr, raddr, format, detail}
}

func (e *Event) String() string {
	msg := fmt.Sprintf(e.format, e.Detail...)
	if e.RemoteAddr != nil {
		return fmt.Sprintf("[%s] [%s] %s", e.RemoteAddr, e.Code.String(), msg)
	}
	return fmt.Sprintf("[%s] %s", e.Code.String(), msg)
}

// Handler is called with events.
type Handler interface {
	// OnEvent is called when an event occurs.
	OnEvent(e *Event)
}
