package irtt

import "os"

const exitCodeSuccess = 0

const exitCodeRuntimeError = 1

const exitCodeBadCommandLine = 2

const exitCodeDoubleSignal = 3

const defaultQuiet = false

const defaultReallyQuiet = false

const defaultHMACKey = ""

type command struct {
	name  string
	desc  string
	run   func([]string)
	usage func()
}

var commands []command

func registerCommand(name, desc string, run func([]string), usage func()) {
	commands = append(commands, command{name, desc, run, usage})
}

func getCommand(name string) *command {
	for _, c := range commands {
		if c.name == name {
			return &c
		}
	}
	return nil
}

func init() {
	registerCommand("client", "runs the client", runClientCLI, clientUsage)
	registerCommand("server", "runs the server", runServerCLI, serverUsage)
	registerCommand("bench", "runs HMAC and fill benchmarks", runBench, nil)
	registerCommand("timer", "runs timer resolution test", runTimer, nil)
	registerCommand("clock", "runs wall vs monotonic clock test", runClock, nil)
	registerCommand("sleep", "runs sleep accuracy test", runSleep, nil)
	registerCommand("version", "shows the version", runVersion, nil)
}

// RunCLI runs the command line interface with the given arguments (typically
// os.Args).
func RunCLI(args []string) {
	if len(args) < 2 {
		usageAndExit(cliUsage, exitCodeBadCommandLine)
	}

	unknownCommandAndExit := func(cmd string) {
		printf("Error: unknown command %s\n", cmd)
		usageAndExit(cliUsage, exitCodeBadCommandLine)
	}

	if args[1] == "help" {
		if len(args) < 3 {
			usageAndExit(cliUsage, exitCodeBadCommandLine)
		}
		c := getCommand(args[2])
		if c == nil {
			unknownCommandAndExit(args[2])
		}
		if c.usage == nil {
			printf("%s: %s", c.name, c.desc)
			os.Exit(exitCodeSuccess)
		}
		usageAndExit(c.usage, exitCodeSuccess)
	}

	c := getCommand(args[1])
	if c == nil {
		unknownCommandAndExit(args[1])
	}
	c.run(args[2:])
}

func cliUsage() {
	setTabWriter(0)
	printf("irtt: measures round-trip time with isochronous UDP packets")
	printf("")
	printf("Usage:")
	printf("")
	printf("\t\t\t\tirtt command [arguments]")
	printf("\t\t\t\tirtt help command")
	printf("")
	printf("Commands:")
	printf("")
	for _, c := range commands {
		printf("\t\t\t\t%s\t%s\t", c.name, c.desc)
	}
}

func usageAndExit(usage func(), exitCode int) {
	if exitCode != exitCodeSuccess {
		printTo = os.Stderr
	}
	usage()
	flush()
	os.Exit(exitCode)
}
