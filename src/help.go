package main

import (
	"fmt"
)

//
// "help" command
func HelpCommand(comName string) {
	var msg string

	switch comName {

	case "locate":
		msg = `NAME
	locate - print configuration directory

USAGE
	rqemu locate

ENVIRONMENT
	RQEMU_HOME - directory to place RQEMU's files
	XDG and HOME evars will be used as alternatives.

EXAMPLE
	$ rqemu locate
	/home/bingo/.rqemu

	$ export XDG_DATA_HOME="$HOME/.local/share"; rqemu locate
	/home/bingo/.local/share/rqemu

	$ export RQEMU_HOME="/var/local/rqemu"; rqemu locate
	/var/local/rqemu`

	case "command":
		msg = `NAME
	command - print QEMU command from JSON config file

USAGE
	rqemu command <vm name>

EXAMPLE
	$ rqemu command debian10-example
	qemu-system-x86_64 -cpu host -enable-kvm \
		-name "debian10-example" \
		-m 6G \
		-smp 2 \
		-nic none \
		-display none \
		-drive file="debian10-example.qcow2",media=disk \
		-monitor unix:"./tmp/debian10-example.mon.sock",server,nowait`

	case "spice":
		msg = `NAME
	spice - connect to a VM's SPICE server

USAGE
	rqemu spice <vm name>

EXAMPLE
	$ rqemu spice debian10-example`

	case "monitor":
		msg = `NAME
	monitor - connect to a VM's monitor server

USAGE
	rqemu monitor <vm name>

EXAMPLE
	$ rqemu spice debian10-example`

	case "stop":
		msg = `NAME
	start - stop an active VM

USAGE
	rqemu stop <vm name>

EXAMPLE
	$ rqemu stop debian10-example`

	case "start":
		msg = `NAME
	start - start VM

USAGE
	rqemu start <vm name>

EXAMPLE
	$ rqemu start debian10-example`

	case "cdrom":
		msg = `NAME
	cdrom - start VM booting from an ISO file

USAGE
	rqemu cdrom <vm name> <iso location>

EXAMPLE
	$ rqemu cdrom debian10-example /mnt/isos/debian_install.iso`

	default:
		msg = `NAME
	rqemu - interactive command line QEMU user interface

USAGE
	rqemu <command> [<sub command>...]
	rqemu help <command>

COMMANDS
	create  create a new VM
	edit    edit the configuration of a VM
	start   start VM
	cdrom   start VM booting from an ISO file
	help    print command details
	stop    kill an active VM
	command print QEMU command from JSON config file
	ls      list active or inactive VMs
	locate  print configuration directory
	monitor connect to a VM's QEMU monitor
	spice   connect to a VM's SPICE server`

	}
	fmt.Println(msg)
}

func Help() {
	HelpCommand("")
}
