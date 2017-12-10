package irtt

import (
	"strings"
)

const globChar = "*"

func globAny(patterns []string, subj string) bool {
	for _, p := range patterns {
		if glob(p, subj) {
			return true
		}
	}
	return false
}

func glob(pattern, subj string) bool {
	if pattern == "" {
		return subj == pattern
	}

	if pattern == globChar {
		return true
	}

	parts := strings.Split(pattern, globChar)

	if len(parts) == 1 {
		return subj == pattern
	}

	leadingGlob := strings.HasPrefix(pattern, globChar)
	trailingGlob := strings.HasSuffix(pattern, globChar)
	end := len(parts) - 1

	for i := 0; i < end; i++ {
		idx := strings.Index(subj, parts[i])

		switch i {
		case 0:
			if !leadingGlob && idx != 0 {
				return false
			}
		default:
			if idx < 0 {
				return false
			}
		}

		subj = subj[idx+len(parts[i]):]
	}

	return trailingGlob || strings.HasSuffix(subj, parts[end])
}
