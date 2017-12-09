package irtt

import (
	"encoding/binary"
	"time"
)

type paramType int

const paramsMaxLen = 64

const (
	pProtoVersion = iota + 1
	pDuration
	pInterval
	pLength
	pReceivedStats
	pStampAt
	pClock
	pDSCP
)

// Params are the test parameters sent to and received from the server.
type Params struct {
	ProtoVersion  int           `json:"proto_version"`
	Duration      time.Duration `json:"duration"`
	Interval      time.Duration `json:"interval"`
	Length        int           `json:"length"`
	ReceivedStats ReceivedStats `json:"received_stats"`
	StampAt       StampAt       `json:"stamp_at"`
	Clock         Clock         `json:"clock"`
	DSCP          int           `json:"dscp"`
}

func parseParams(b []byte) (*Params, error) {
	p := &Params{}
	for pos := 0; pos < len(b); {
		n, err := p.readParam(b[pos:])
		if err != nil {
			return nil, err
		}
		pos += n
	}
	return p, nil
}

func (p *Params) bytes() []byte {
	b := make([]byte, paramsMaxLen)
	pos := 0
	pos += binary.PutUvarint(b[pos:], pProtoVersion)
	pos += binary.PutVarint(b[pos:], int64(p.ProtoVersion))
	pos += binary.PutUvarint(b[pos:], pDuration)
	pos += binary.PutVarint(b[pos:], int64(p.Duration))
	pos += binary.PutUvarint(b[pos:], pInterval)
	pos += binary.PutVarint(b[pos:], int64(p.Interval))
	pos += binary.PutUvarint(b[pos:], pLength)
	pos += binary.PutVarint(b[pos:], int64(p.Length))
	pos += binary.PutUvarint(b[pos:], pReceivedStats)
	pos += binary.PutVarint(b[pos:], int64(p.ReceivedStats))
	pos += binary.PutUvarint(b[pos:], pStampAt)
	pos += binary.PutVarint(b[pos:], int64(p.StampAt))
	pos += binary.PutUvarint(b[pos:], pClock)
	pos += binary.PutVarint(b[pos:], int64(p.Clock))
	pos += binary.PutUvarint(b[pos:], pDSCP)
	pos += binary.PutVarint(b[pos:], int64(p.DSCP))
	return b[:pos]
}

func (p *Params) readParam(b []byte) (int, error) {
	pos := 0
	t, n, err := readUvarint(b[pos:])
	if err != nil {
		return 0, err
	}
	pos += n
	v, n, err := readVarint(b[pos:])
	if err != nil {
		return 0, err
	}
	pos += n
	switch t {
	case pProtoVersion:
		p.ProtoVersion = int(v)
	case pDuration:
		p.Duration = time.Duration(v)
		if p.Duration <= 0 {
			return 0, Errorf(InvalidParamValue, "duration %d is <= 0", p.Duration)
		}
	case pInterval:
		p.Interval = time.Duration(v)
		if p.Interval <= 0 {
			return 0, Errorf(InvalidParamValue, "interval %d is <= 0", p.Interval)
		}
	case pLength:
		p.Length = int(v)
	case pReceivedStats:
		p.ReceivedStats, err = ReceivedStatsFromInt(int(v))
		if err != nil {
			return 0, err
		}
	case pStampAt:
		p.StampAt, err = StampAtFromInt(int(v))
		if err != nil {
			return 0, err
		}
	case pClock:
		p.Clock, err = ClockFromInt(int(v))
		if err != nil {
			return 0, err
		}
	case pDSCP:
		p.DSCP = int(v)
	default:
		// note: unknown params are silently ignored
	}
	return pos, nil
}

func readUvarint(b []byte) (v uint64, n int, err error) {
	v, n = binary.Uvarint(b)
	if n == 0 {
		err = Errorf(ShortParamBuffer,
			"param buffer too short for uvarint (%d)", len(b))
	}
	if n < 0 {
		err = Errorf(ParamOverflow,
			"param value overflow for uvarint (read %d)", n)
	}
	return
}

func readVarint(b []byte) (v int64, n int, err error) {
	v, n = binary.Varint(b)
	if n == 0 {
		err = Errorf(ShortParamBuffer,
			"param buffer too short for varint (%d)", len(b))
	}
	if n < 0 {
		err = Errorf(ParamOverflow,
			"param value overflow for varint (read %d)", n)
	}
	return v, n, nil
}
