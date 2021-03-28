package main

import (
	"fmt"
	"os"
)

//
// utils
func PrintErr(m interface{}) {
	fmt.Fprintln(os.Stderr, m)
}


//
// config variables
var home string

// environment variables
const evarHome = "RQEMU_HOME"
const evarXdgDataHome = "XDG_DATA_HOME"
const evarXdgRuntimeDir = "XDG_RUNTIME_DIR"
const evarXdgConfigHome = "XDG_CONFIG_HOME"


//
// help command
func Help() {
	msg := `NAME
	rqemu - interactive command line QEMU user interface

SYNOPSIS
	rqemu <command> [<sub command>...]
	rqemu help <command>

COMMANDS
	create
	edit
	start
	help
	stop
	ls      list active or inactive VMs
	locate  print configuration directory
	monitor connect to a VM's QEMU monitor
	spice   connect to a VM's SPICE server`

	fmt.Println(msg)
}

// set the directory to place/read all RQEMU files
func SetHomeDirectory() {
	home = os.Getenv(evarHome)
	if len(home) > 0 {
		return
	}
	const dirName = "rqemu"

	home = os.Getenv(evarXdgDataHome)
	if len(home) > 0 {
		home = home + "/" + dirName
		return
	}

	home = os.Getenv(evarXdgRuntimeDir)
	if len(home) > 0 {
		home = home + "/" + dirName
		return
	}

	home = os.Getenv(evarXdgConfigHome)
	if len(home) > 0 {
		home = home + "/" + dirName
		return
	}

	home = os.Getenv("HOME")
	if len(home) > 0 {
		PrintErr("Couldn't find " +
			evarHome + ", " +
			evarXdgDataHome + ", " +
			evarXdgRuntimeDir + ", " +
			evarXdgConfigHome + "env variables, using HOME.")
		home = home + "/." + dirName
		return
	}

	PrintErr("Couldn't find " +
		evarHome + ", " +
		evarXdgDataHome + ", " +
		evarXdgRuntimeDir + ", " +
		evarXdgConfigHome + " or even HOME env variables, using PWD.")
	home = dirName
}

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		Help()
		os.Exit(1)
	}

	SetHomeDirectory()

	switch args[0] {
	case "help":
		Help()

	case "locate":
		fmt.Println(home)

	default:
		PrintErr("No such command '" + args[0] + "'")
		Help()
		os.Exit(1)
	}
}

