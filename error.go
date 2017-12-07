package irtt

// Common error codes.
const (
	ShortWrite Code = -1 * (iota + 1)
	InvalidDFString
	FieldsLengthTooLarge
	FieldsCapacityTooLarge
	InvalidStampAtString
	InvalidStampAtInt
	InvalidAllowStampString
	InvalidClockString
	InvalidClockInt
	BadMagic
	NoHMAC
	BadHMAC
	UnexpectedHMAC
	NonexclusiveMidpointTStamp
	InconsistentClocks
	DFNotSupported
	InvalidFlagBitsSet
	ShortParamBuffer
	ParamOverflow
	UnknownParam
	InvalidParamValue
)

// Server error codes.
const (
	NoMatchingInterfaces Code = -1 * (iota + 1*1024)
	NoMatchingInterfacesUp
	UnspecifiedWithSpecifiedAddresses
	InvalidGCModeString
	UnexpectedReplyFlag
	NoSuitableAddressFound
	InvalidConnToken
	ShortInterval
	LargeRequest
	AddressMismatch
)

// Client error codes.
const (
	InvalidWinAvgWindow Code = -1 * (iota + 2*1024)
	InvalidExpAvgAlpha
	AllocateResultsPanic
	UnexpectedOpenFlag
	DFError
	TTLError
	ExpectedReplyFlag
	ShortReply
	StampAtMismatch
	ClockMismatch
	UnexpectedSequenceNumber
	InvalidSleepFactor
	InvalidWaitString
	InvalidWaitFactor
	InvalidWaitDuration
	NoSuchAverager
	NoSuchFiller
	NoSuchTimer
	NoSuchWaiter
	IntervalNonPositive
	DurationNonPositive
	ConnTokenZero
	ServerClosed
	OpenTimeout
	InvalidServerRestriction
	ParamsChanged
	InvalidReceivedStatsInt
	InvalidReceivedStatsString
	OpenTimeoutTooShort
)

// Error is an IRTT error.
type Error struct {
	*Event
}

// Errorf returns a new Error.
func Errorf(code Code, format string, detail ...interface{}) *Error {
	return &Error{Eventf(code, nil, nil, format, detail...)}
}

func (e *Error) Error() string {
	return e.Event.String()
}
