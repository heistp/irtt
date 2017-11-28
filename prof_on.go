// +build profile

package irtt

import (
	"runtime/debug"

	"github.com/pkg/profile"
)

const profileEnabled = true

func startProfile(path string) interface {
	Stop()
} {
	debug.SetGCPercent(-1)
	return profile.Start(profile.MemProfile, profile.ProfilePath(path),
		profile.NoShutdownHook)
}
