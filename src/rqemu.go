package main

import (
	"fmt"
	"os"
	"strings"
)

//
// config variables
var home string
var tmp string

const spiceSocketSuffix = ".spice.sock"
const monSocketSuffix = ".mon.sock"

// environment variables
const evarHome = "RQEMU_HOME"
const evarXdgDataHome = "XDG_DATA_HOME"
const evarXdgRuntimeDir = "XDG_RUNTIME_DIR"
const evarXdgConfigHome = "XDG_CONFIG_HOME"

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
	tmp = home + "/tmp"

	os.MkdirAll(home, os.ModePerm)
	os.MkdirAll(tmp, os.ModePerm)
	os.Chdir(home)

	switch args[0] {
	case "help":
		if len(args) < 2 || len(args[1]) <= 0 {
			Help()
			return
		}
		HelpCommand(args[1])

	case "locate":
		fmt.Println(home)

	case "command":
		if len(args) < 2 || len(args[1]) <= 0 {
			HelpCommand(args[0])
			os.Exit(1)
			return
		}
		command := Command(args[1], true)
		fmt.Println(command)

	case "spice":
		if len(args) < 2 || len(args[1]) <= 0 {
			HelpCommand(args[0])
			os.Exit(1)
			return
		}
		OpenSpiceViewer(args[1])

	case "start":
		if len(args) < 2 || len(args[1]) <= 0 {
			HelpCommand(args[0])
			os.Exit(1)
			return
		}
		command := Command(args[1], false)
		Exec(command)
		fmt.Println("'" + args[1] + "' VM started.")

	case "stop":
		if len(args) < 2 || len(args[1]) <= 0 {
			HelpCommand(args[0])
			os.Exit(1)
			return
		}

		pids := GetVmPids(args[1])
		if len(pids) < 1 {
			PrintErr("'" + args[1] + "' VM, not active.")
			os.Exit(1)
		}

		for i := 0; i < len(pids); i++ {
			pid := pids[i]
			proc, err := os.FindProcess(pid)

			if err != nil {
				PrintErr(err)
			}
			proc.Kill()
		}

		// remove monitor and spice files
		os.Remove(tmp + "/" + args[1] + monSocketSuffix)
		os.Remove(tmp + "/" + args[1] + spiceSocketSuffix)
		fmt.Println("Destroyed '" + args[1] + "' VM.")

	case "cdrom":
		if len(args) < 3 || len(args[1]) <= 0 {
			HelpCommand(args[0])
			os.Exit(1)
			return
		}
		command := Command(args[1], false)
		command += " -cdrom " + args[2]
		command += " -boot d"

		// provide a graphical window, for setup
		command = strings.Replace(command, "-display none", "-display gtk,show-cursor=off", -1)

		Exec(command)
		fmt.Println("'" + args[1] + "' VM started with CDROM + '" + args[2] + "'.")

		if strings.Contains(command, "-spice") {
			OpenSpiceViewer(args[1])
		}

	case "ls":
		vms := ListActiveVms()
		for i := 0; i < len(vms); i++ {
			fmt.Println(vms[i])
		}

	case "monitor":
		if len(args) < 2 || len(args[1]) <= 0 {
			HelpCommand(args[0])
			os.Exit(1)
			return
		}

		monFile := args[1] + monSocketSuffix

		// find a way to spawn nc without a GUI crutch
		Run("nc -U " + tmp + "/" + monFile)

	default:
		PrintErr("No such command '" + args[0] + "'")
		Help()
		os.Exit(1)
	}
}
