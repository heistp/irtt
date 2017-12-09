package irtt

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"
)

// Filler is a Reader used for filling the payload in packets.
type Filler interface {
	io.Reader

	String() string
}

// PatternFiller can be used to fill with a repeating byte pattern.
type PatternFiller struct {
	Bytes []byte
	buf   []byte
	pos   int
}

// NewPatternFiller returns a new PatternFiller.
func NewPatternFiller(bytes []byte) *PatternFiller {
	var blen int
	if len(bytes) > patternMaxInitLen {
		blen = len(bytes)
	} else {
		blen = patternMaxInitLen / len(bytes) * (len(bytes) + 1)
	}
	buf := make([]byte, blen)
	for i := 0; i < len(buf); i += len(bytes) {
		copy(buf[i:], bytes)
	}
	return &PatternFiller{bytes, buf, 0}
}

// NewDefaultPatternFiller returns a new PatternFiller with the default pattern.
func NewDefaultPatternFiller() *PatternFiller {
	return NewPatternFiller(DefaultFillPattern)
}

func (f *PatternFiller) Read(p []byte) (n int, err error) {
	l := 0
	for l < len(p) {
		c := copy(p[l:], f.buf[f.pos:])
		l += c
		f.pos = (f.pos + c) % len(f.Bytes)
	}
	return l, nil
}

func (f *PatternFiller) String() string {
	return fmt.Sprintf("pattern:%x", f.Bytes)
}

// RandFiller is a Filler that fills with data from math.rand.
type RandFiller struct {
	*rand.Rand
}

// NewRandFiller returns a new RandFiller.
func NewRandFiller() *RandFiller {
	return &RandFiller{rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (rf *RandFiller) String() string {
	return "rand"
}

// FillerFactories are the registered Filler factories.
var FillerFactories = make([]FillerFactory, 0)

// FillerFactory can create a Filler from a string.
type FillerFactory struct {
	FactoryFunc func(string) (Filler, error)
	Usage       string
}

// RegisterFiller registers a new Filler.
func RegisterFiller(fn func(string) (Filler, error), usage string) {
	FillerFactories = append(FillerFactories, FillerFactory{fn, usage})
}

// NewFiller returns a Filler from a string.
func NewFiller(s string) (Filler, error) {
	if s == "none" {
		return nil, nil
	}
	for _, fac := range FillerFactories {
		f, err := fac.FactoryFunc(s)
		if err != nil {
			return nil, err
		}
		if f != nil {
			return f, nil
		}
	}
	return nil, Errorf(NoSuchFiller, "no such Filler %s", s)
}

func init() {
	RegisterFiller(
		func(s string) (f Filler, err error) {
			if s == "rand" {
				f = NewRandFiller()
			}
			return
		},
		"rand: use random bytes from Go's math.rand",
	)

	RegisterFiller(
		func(s string) (Filler, error) {
			args := strings.Split(s, ":")
			if args[0] != "pattern" {
				return nil, nil
			}
			var b []byte
			if len(args) == 1 {
				b = DefaultFillPattern
			} else {
				var err error
				b, err = hex.DecodeString(args[1])
				if err != nil {
					return nil, err
				}
			}
			return NewPatternFiller(b), nil
		},
		fmt.Sprintf("pattern:XX: use repeating pattern of hex (default %x)",
			DefaultFillPattern),
	)
}
