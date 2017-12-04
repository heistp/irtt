package irtt

// Error codes.
const (
	InvalidWinAvgWindow Code = (iota + 1) * -1
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
	DFNotSupported
	IntervalNonPositive
	DurationNonPositive
	ConnTokenZero
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
	AllocateResultsPanic
	NoMatchingInterfaces
	NoMatchingInterfacesUp
	UnspecifiedWithSpecifiedAddresses
	InvalidGCModeString
	UnexpectedOpenFlag
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
