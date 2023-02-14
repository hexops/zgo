package main

import (
	"flag"
	"log"
	"os"

	"github.com/hexops/cmder"
)

var commands cmder.Commander

var usageText = `zgo is a tool for making Go and Zig best friends

Usage:

	zgo <command> [arguments]

The commands are:

	build    compile packages and dependencies

Use "zgo <command> -h" for more information about a command.
`

func main() {
	// Configure logging if desired.
	log.SetFlags(0)
	log.SetPrefix("")

	commands.Run(flag.CommandLine, "zgo", usageText, os.Args[1:])
}
