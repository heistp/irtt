// +build prod

package irtt

import (
	"os"
	"time"
)

const minNonRootInterval = 10 * time.Millisecond

// Note that Windows always reports a UID of -1. Therefore, Windows users will
// not be subject to this restriction.

func validateInterval(i time.Duration) error {
	// do not allow non-root users an interval of less than 10ms
	if i < minNonRootInterval && os.Geteuid() > 0 {
		return Errorf(IntervalNotPermitted, "interval < %s not permitted for non-root user",
			minNonRootInterval)
	}
	return nil
}
