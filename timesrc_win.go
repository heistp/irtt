// +build windows

package irtt

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WindowsTimeSource uses GetSystemTimePreciseAsFileTime for the wall clock and
// QueryPerformanceFrequency/Counter for the monotonic clock.
type WindowsTimeSource struct {
	period int64

	qpc *windows.LazyProc
}

// Now returns a Time containing the current time.
func (w *WindowsTimeSource) Now(clock Clock) Time {
	switch clock {
	case Wall:
		return Time{w.systemTimePreciseNs(), time.Duration(0)}
	case Monotonic:
		return Time{0, w.queryPerformanceCounterNs()}
	case BothClocks:
		return Time{w.systemTimePreciseNs(), w.queryPerformanceCounterNs()}
	default:
		panic(fmt.Sprintf("unknown clock %s", clock))
	}
}

func (w *WindowsTimeSource) String() string {
	return "windows"
}

func (w *WindowsTimeSource) systemTimePreciseNs() int64 {
	var t windows.Filetime
	windows.GetSystemTimePreciseAsFileTime(&t)
	return t.Nanoseconds()
}

func (w *WindowsTimeSource) queryPerformanceCounterNs() time.Duration {
	var ctr int64
	ret, _, err := w.qpc.Call(uintptr(unsafe.Pointer(&ctr)))
	if ret == 0 {
		panic(err)
	}
	return time.Duration(ctr * w.period)
}

// NewWindowsTimeSource returns a new WindowsTimeSource.
func NewWindowsTimeSource() (ts *WindowsTimeSource, err error) {
	k := windows.NewLazySystemDLL("kernel32.dll")
	if err = k.Load(); err != nil {
		return
	}

	qpf := k.NewProc("QueryPerformanceFrequency")
	if err = qpf.Find(); err != nil {
		return
	}

	qpc := k.NewProc("QueryPerformanceCounter")
	if err = qpc.Find(); err != nil {
		return
	}

	var freq int64
	var ret uintptr
	if ret, _, err = qpf.Call(uintptr(unsafe.Pointer(&freq))); ret == 0 {
		return
	}

	err = nil
	ts = &WindowsTimeSource{1000000000 / freq, qpc}

	return
}

// NewDefaultTimeSource returns a WindowsTimeSource for Windows and GoTimeSource
// for everything else.
func NewDefaultTimeSource() TimeSource {
	wts, err := NewWindowsTimeSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "falling back to Go time source: %s\n", err)
		return NewGoTimeSource()
	}
	return wts
}

func init() {
	RegisterTimeSource(
		func(s string) (t TimeSource, err error) {
			if s == "windows" {
				t, err = NewWindowsTimeSource()
			}
			return
		},
		"windows: GetSystemTimePreciseAsFileTime/QueryPerformanceCounter",
	)
}
