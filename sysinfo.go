package irtt

import (
	"os"
	"runtime"
)

// SystemInfo stores based system information.
type SystemInfo struct {
	OS        string `json:"os"`
	NumCPU    int    `json:"cpus"`
	GoVersion string `json:"go_version"`
	Hostname  string `json:"hostname"`
}

// NewSystemInfo returns a new SystemInfo.
func NewSystemInfo() *SystemInfo {
	s := &SystemInfo{
		OS:        runtime.GOOS,
		NumCPU:    runtime.NumCPU(),
		GoVersion: runtime.Version(),
	}
	s.Hostname, _ = os.Hostname()
	return s
}
