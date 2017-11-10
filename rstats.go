package irtt

import (
	"encoding/json"
	"fmt"
)

// ReceivedStats selects what information to gather about received packets.
type ReceivedStats int

// ReceivedStats constants.
const (
	ReceivedStatsNone   ReceivedStats = 0x00
	ReceivedStatsCount  ReceivedStats = 0x01
	ReceivedStatsWindow ReceivedStats = 0x02
	ReceivedStatsBoth   ReceivedStats = ReceivedStatsCount | ReceivedStatsWindow
)

var rss = [...]string{"none", "count", "window", "both"}

func (rs ReceivedStats) String() string {
	if int(rs) < 0 || int(rs) >= len(rss) {
		return fmt.Sprintf("ReceivedStats:%d", rs)
	}
	return rss[rs]
}

// ReceivedStatsFromInt returns a ReceivedStats value from its int constant.
func ReceivedStatsFromInt(v int) (ReceivedStats, error) {
	if v < int(ReceivedStatsNone) || v > int(ReceivedStatsBoth) {
		return ReceivedStatsNone, Errorf(InvalidReceivedStatsInt,
			"invalid ReceivedStats int: %d", v)
	}
	return ReceivedStats(v), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (rs ReceivedStats) MarshalJSON() ([]byte, error) {
	return json.Marshal(rs.String())
}

// ReceivedStatsFromString returns a ReceivedStats value from its string.
func ReceivedStatsFromString(s string) (ReceivedStats, error) {
	for i, v := range rss {
		if v == s {
			return ReceivedStats(i), nil
		}
	}
	return ReceivedStatsNone, Errorf(InvalidReceivedStatsString,
		"invalid ReceivedStats string: %s", s)
}

// Lost indicates the lost status of a packet.
type Lost int

// Lost constants.
const (
	LostTrue Lost = iota
	LostDown
	LostUp
	LostFalse
)

var lsts = [...]string{"true", "true_down", "true_up", "false"}

func (l Lost) String() string {
	if int(l) < 0 || int(l) >= len(lsts) {
		return fmt.Sprintf("Lost:%d", l)
	}
	return lsts[l]
}

// MarshalJSON implements the json.Marshaler interface.
func (l Lost) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}
