package irtt

import (
	"encoding/binary"
	"encoding/json"
	"time"
)

type paramType int

const paramsMaxLen = 128

const (
	pProtocolVersion = iota + 1
	pDuration
	pInterval
	pLength
	pReceivedStats
	pStampAt
	pClock
	pDSCP
	pServerFill
)

// Params are the test parameters sent to and received from the server.
type Params struct {
	ProtocolVersion int           `json:"proto_version"`
	Duration        time.Duration `json:"duration"`
	Interval        time.Duration `json:"interval"`
	Length          int           `json:"length"`
	ReceivedStats   ReceivedStats `json:"received_stats"`
	StampAt         StampAt       `json:"stamp_at"`
	Clock           Clock         `json:"clock"`
	DSCP            int           `json:"dscp"`
	ServerFill      string        `json:"server_fill"`
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
	if p.ProtocolVersion != 0 {
		pos += binary.PutUvarint(b[pos:], pProtocolVersion)
		pos += binary.PutVarint(b[pos:], int64(p.ProtocolVersion))
	}
	if p.Duration != 0 {
		pos += binary.PutUvarint(b[pos:], pDuration)
		pos += binary.PutVarint(b[pos:], int64(p.Duration))
	}
	if p.Interval != 0 {
		pos += binary.PutUvarint(b[pos:], pInterval)
		pos += binary.PutVarint(b[pos:], int64(p.Interval))
	}
	if p.Length != 0 {
		pos += binary.PutUvarint(b[pos:], pLength)
		pos += binary.PutVarint(b[pos:], int64(p.Length))
	}
	if p.ReceivedStats != 0 {
		pos += binary.PutUvarint(b[pos:], pReceivedStats)
		pos += binary.PutVarint(b[pos:], int64(p.ReceivedStats))
	}
	if p.StampAt != 0 {
		pos += binary.PutUvarint(b[pos:], pStampAt)
		pos += binary.PutVarint(b[pos:], int64(p.StampAt))
	}
	if p.Clock != 0 {
		pos += binary.PutUvarint(b[pos:], pClock)
		pos += binary.PutVarint(b[pos:], int64(p.Clock))
	}
	if p.DSCP != 0 {
		pos += binary.PutUvarint(b[pos:], pDSCP)
		pos += binary.PutVarint(b[pos:], int64(p.DSCP))
	}
	if len(p.ServerFill) > 0 {
		pos += binary.PutUvarint(b[pos:], pServerFill)
		pos += putString(b[pos:], p.ServerFill, maxServerFillLen)
	}
	return b[:pos]
}

func (p *Params) readParam(b []byte) (pos int, err error) {
	var t uint64
	var n int
	t, n, err = readUvarint(b[pos:])
	if err != nil {
		return
	}
	pos += n

	if t == pServerFill {
		p.ServerFill, n, err = readString(b[pos:], maxServerFillLen)
		if err != nil {
			return
		}
	} else {
		var v int64
		v, n, err = readVarint(b[pos:])
		if err != nil {
			return
		}
		switch t {
		case pProtocolVersion:
			p.ProtocolVersion = int(v)
		case pDuration:
			p.Duration = time.Duration(v)
			if p.Duration <= 0 {
				err = Errorf(InvalidParamValue, "duration %d is <= 0", p.Duration)
			}
		case pInterval:
			p.Interval = time.Duration(v)
			if p.Interval <= 0 {
				err = Errorf(InvalidParamValue, "interval %d is <= 0", p.Interval)
			}
		case pLength:
			p.Length = int(v)
		case pReceivedStats:
			p.ReceivedStats, err = ReceivedStatsFromInt(int(v))
		case pStampAt:
			p.StampAt, err = StampAtFromInt(int(v))
		case pClock:
			p.Clock, err = ClockFromInt(int(v))
		case pDSCP:
			p.DSCP = int(v)
		default:
			// note: unknown params are silently ignored
		}
	}
	if err != nil {
		return
	}
	pos += n
	return
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
	return
}

func readString(b []byte, maxLen int) (v string, n int, err error) {
	l, n, err := readUvarint(b[n:])
	if err != nil {
		return
	}
	if l > uint64(maxLen) {
		err = Errorf(ParamOverflow, "string param too large (%d>%d)", l, maxLen)
		return
	}
	if len(b[n:]) < int(l) {
		err = Errorf(ShortParamBuffer,
			"param buffer (%d) too short for string (%d)", len(b[n:]), l)
		return
	}
	v = string(b[n : n+int(l)])
	n += int(l)
	return
}

func putString(b []byte, s string, maxLen int) (n int) {
	l := len(s)
	if l > maxLen {
		l = maxLen
	}
	n += binary.PutUvarint(b[n:], uint64(l))
	n += copy(b[n:], s[:l])
	return
}

func (p *Params) UnmarshalJSON(b []byte) error {
	var f map[string] *json.RawMessage
	json.Unmarshal(b, &f)

	var v map[string]interface{}
	json.Unmarshal(*f["params"], &v)

	p.ProtocolVersion = int(v["proto_version"].(float64))
	p.Duration = time.Duration(v["duration"].(float64))
	p.Interval = time.Duration(v["interval"].(float64))
	p.Length = int(v["length"].(float64))
	rs, _:= ParseReceivedStats(v["received_stats"].(string))
	p.ReceivedStats = rs
	stamp, _:= ParseStampAt(v["stamp_at"].(string))
	p.StampAt = stamp
	clock, _ := ParseClock(v["clock"].(string))
	p.Clock = clock
	p.DSCP = int(v["dscp"].(float64))
	p.ServerFill = v["server_fill"].(string)

	return nil
}
