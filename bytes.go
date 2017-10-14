package irtt

import (
	"encoding/hex"
	"strings"
)

// bytes helpers

// make zeroes array so we can use copy builtin for fast zero-ing
var zeroes = make([]byte, 64*1024)

func decodeHexOrNot(s string) (b []byte, err error) {
	if strings.HasPrefix(s, "0x") {
		b, err = hex.DecodeString(s[2:])
		return
	}
	b = []byte(s)
	return
}

func bytesEqual(a, b []byte) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func zero(b []byte) {
	if len(b) > len(zeroes) {
		zeroes = make([]byte, len(b)*2)
	}
	copy(b, zeroes)
}
