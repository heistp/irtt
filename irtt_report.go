package irtt

import (
	"encoding/json"
	"io"
	"os"
)

// runReport emits a report from a JSON file.
func runReport(args []string) {
	var r io.Reader
	if len(args) == 0 || args[0] == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(args[0])
		if err != nil {
			exitOnError(err, exitCodeRuntimeError)
		}
		defer f.Close()
		r = f
	}
	var p PrintableResult
	d := json.NewDecoder(r)
	if err := d.Decode(&p); err != nil {
		exitOnError(err, exitCodeRuntimeError)
	}
	printResult(&p)
}

func reportUsage() {
	setBufio()
	printf("Usage: report [file|-]")
	printf("")
	printf("Emits the standard end of test report from JSON output.")
	printf("")
	printf("If a filename is given, JSON is read from the given file.")
	printf("")
	printf("If no argument or a - character is given, JSON is read from stdin.")
}
