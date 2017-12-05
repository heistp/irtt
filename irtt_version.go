package irtt

import "runtime"

func runVersion(args []string) {
	printf("irtt version: %s", Version)
	printf("build date: %s", BuildDate)
	printf("go version: %s on %s/%s", runtime.Version(),
		runtime.GOOS, runtime.GOARCH)
}
