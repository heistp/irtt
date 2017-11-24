package irtt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

var printTo io.Writer = os.Stdout

type flusher interface {
	Flush() error
}

func printf(format string, args ...interface{}) {
	fmt.Fprintf(printTo, fmt.Sprintf("%s\n", format), args...)
}

func println(s string) {
	fmt.Fprintln(printTo, s)
}

func setTabWriter(flags uint) {
	printTo = tabwriter.NewWriter(printTo, 0, 0, 2, ' ', flags)
}

func setBufio() {
	printTo = bufio.NewWriter(printTo)
}

func flush() {
	if f, ok := printTo.(flusher); ok {
		f.Flush()
	}
}

func exitOnError(err error, code int) {
	if err != nil {
		printTo = os.Stderr
		printf("Error: %s", err)
		os.Exit(code)
	}
}
