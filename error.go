package irtt

import "fmt"

// ErrorCode is a code identifying an Error.
type ErrorCode int

//go:generate stringer -type=ErrorCode

// Error codes.
const (
	InvalidWinAvgWindow ErrorCode = iota
	InvalidExpAvgAlpha
	DSCPError
	DFError
	TTLError
	ShortWrite
	ExpectedReplyFlag
	UnexpectedReplyFlag
	ShortReply
	StampAtMismatch
	ClockMismatch
	UnexpectedSequenceNumber
	InvalidDFString
	FieldsLengthTooLarge
	FieldsCapacityTooLarge
	InvalidStampAtString
	InvalidStampAtInt
	InvalidAllowStampString
	InvalidClockString
	InvalidClockInt
	InvalidSleepFactor
	IntervalNotPermitted
	InvalidWaitString
	InvalidWaitFactor
	InvalidWaitDuration
	NoSuchAverager
	NoSuchFiller
	NoSuchTimer
	NoSuchWaiter
	BadMagic
	NoHMAC
	BadHMAC
	UnexpectedHMAC
	NonexclusiveMidpointTStamp
	InconsistentClocks
	ListenerPanic
	DFNotSupported
	IntervalNonPositive
	DurationNonPositive
	OpenCloseBothSet
	ConnTokenZero
	ConnClosed
	InvalidFlagBitsSet
	ServerClosed
	ShortParamBuffer
	ParamOverflow
	UnknownParam
	NoSuitableAddressFound
	OpenTimeout
	InvalidServerRestriction
	InvalidParamValue
	ParamsChanged
	InvalidReceivedStatsInt
	InvalidReceivedStatsString
	OpenTimeoutTooShort
)

// Error is an IRTT error.
type Error struct {
	Code   ErrorCode
	format string
	Detail []interface{}
}

// Errorf returns a new Error.
func Errorf(code ErrorCode, format string, args ...interface{}) *Error {
	return &Error{code, format, args}
}

func (e *Error) Error() string {
	return fmt.Sprintf(e.format, e.Detail...)
}
