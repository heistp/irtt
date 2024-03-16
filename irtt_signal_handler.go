package irtt

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

const (
	signalChannelSize = 10
)

// initSignalHandler sets up signal handling for the process, and
// will call cancel() when recieved
func initSignalHandler(cancel context.CancelFunc, reallyQuiet bool) {

	sigs := make(chan os.Signal, signalChannelSize)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	sig := <-sigs
	if !reallyQuiet {
		printf("%s", sig)
	}
	cancel()

	<-sigs
	if !reallyQuiet {
		printf("second interrupt, exiting")
	}
	os.Exit(exitCodeDoubleSignal)

}
