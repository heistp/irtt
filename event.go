package irtt

import (
	"fmt"
	"net"
)

// Code uniquely identifies events and errors to improve context.
type Code int

//go:generate stringer -type=Code

// Event codes.
const (
	MultipleAddresses Code = iota + 1
	ServerStart
	ListenerStart
	ListenerStop
	ListenerError
	Drop
	DropUnparseableParams
	DropInvalidConnToken
	DropAddressMismatch
	DropShortInterval
	Connecting
	Connected
	WaitForPackets
	NewConn
	OpenClose
	CloseConn
	NoDSCPSupport
	ServerRestriction
	DurationLimitExceeded
	NoTest
	NoReceiveDstAddrSupport
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
