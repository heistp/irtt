// +build profile

package irtt

import "github.com/pkg/profile"

const profileEnabled = true

func startProfile(path string) interface {
	Stop()
} {
	return profile.Start(profile.CPUProfile, profile.ProfilePath(path),
		profile.NoShutdownHook)
}
