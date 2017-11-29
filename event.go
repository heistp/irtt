package irtt

import (
	"fmt"
	"net"
)

// EventCode uniquely identifies events to improve context.
type EventCode uint64

//go:generate stringer -type=EventCode

// Event codes.
const (
	MultipleAddresses EventCode = 1 << iota
	ServerStart
	ListenerStart
	ListenerStop
	ListenerError
	DropBadMagic
	DropNoHMAC
	DropBadHMAC
	DropUnexpectedHMAC
	DropNonexclusiveMidpointStamp
	DropInconsistentClocks
	DropPacketError
	DropExpectedReply
	DropUnexpectedReply
	DropUnexpectedOpenFlag
	DropSmallPacket
	DropInvalidFlagBitsSet
	DropInvalidConnToken
	DropAddressMismatch
	DropUnparseableParams
	DropShortInterval
	SockoptDSCPFail
	SockoptDSCPAbort
	SockoptDFFail
	SockoptDFAbort
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

// AllEvents is a mask matching all events.
const AllEvents = EventCode(^uint64(0))

// Event is an event sent to a Handler.
type Event struct {
	EventCode  EventCode
	LocalAddr  net.Addr
	RemoteAddr net.Addr
	format     string
	Detail     []interface{}
}

// Eventf returns a new event.
func Eventf(code EventCode, laddr net.Addr, raddr net.Addr, format string,
	args ...interface{}) *Event {
	return &Event{code, laddr, raddr, format, args}
}

func (e *Event) String() string {
	msg := fmt.Sprintf(e.format, e.Detail...)
	return fmt.Sprintf("[%s] %s", e.EventCode.String(), msg)
}

// Handler is called with events.
type Handler interface {
	// OnEvent is called when an event occurs.
	OnEvent(e *Event)
}

// dropCode returns an EventCode for an ErrorCode for packet errors. This is not
// very maintainable, because every time I add or change packet errors I
// have to make sure this mapping stays in sync, but I don't want to resort to
// reflection or other tricks to maintain this mapping, and I don't want common
// error and event codes, because: 1) the 64-bit uint isn't large enough to hold
// them all, and 2) there may not be parity between errors and events.
func dropCode(errcode ErrorCode) EventCode {
	switch errcode {
	case BadMagic:
		return DropBadMagic
	case NoHMAC:
		return DropNoHMAC
	case BadHMAC:
		return DropBadHMAC
	case UnexpectedHMAC:
		return DropUnexpectedHMAC
	case NonexclusiveMidpointTStamp:
		return DropNonexclusiveMidpointStamp
	case InconsistentClocks:
		return DropInconsistentClocks
	case FieldsLengthTooLarge:
		return DropSmallPacket
	case FieldsCapacityTooLarge:
		return DropSmallPacket
	case ExpectedReplyFlag:
		return DropExpectedReply
	case UnexpectedReplyFlag:
		return DropUnexpectedReply
	case InvalidFlagBitsSet:
		return DropInvalidFlagBitsSet
	default:
		return DropPacketError
	}
}
