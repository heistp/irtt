// +build profile

package irtt

import (
	"github.com/pkg/profile"
)

const profileEnabled = true

func startProfile(path string) interface {
	Stop()
} {
	//debug.SetGCPercent(-1)
	return profile.Start(profile.CPUProfile, profile.ProfilePath(path),
		profile.NoShutdownHook)
}
