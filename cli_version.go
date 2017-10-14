package irtt

import "runtime"

func runVersion(args []string) {
	printf("irtt version %s, %s", Version, runtime.Version())
}
