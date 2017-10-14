// +build !profile

package irtt

const profileEnabled = false

func startProfile(path string) interface {
	Stop()
} {
	return nil
}
