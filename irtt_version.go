package irtt

import "runtime"

func runVersion(args []string) {
	printf("irtt version: %s", Version)
	printf("protocol version: %d", ProtocolVersion)
	printf("json format version: %d", JSONFormatVersion)
	printf("build date: %s", BuildDate)
	printf("go version: %s on %s/%s", runtime.Version(),
		runtime.GOOS, runtime.GOARCH)
}
