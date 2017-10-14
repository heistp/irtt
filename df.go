package irtt

import (
	"fmt"
)

// DF is the value for the do not fragment bit.
type DF int

// DF constants.
const (
	DFDefault DF = iota
	DFFalse
	DFTrue
)

var dfs = [...]string{"default", "false", "true"}

func (d DF) String() string {
	if int(d) < 0 || int(d) > len(dfs) {
		return fmt.Sprintf("DF:%d", d)
	}
	return dfs[int(d)]
}

// DFFromString returns a DF value from its string.
func DFFromString(s string) (DF, error) {
	for i, x := range dfs {
		if x == s {
			return DF(i), nil
		}
	}
	return DFDefault, Errorf(InvalidDFString, "invalid DF string: %s", s)
}
