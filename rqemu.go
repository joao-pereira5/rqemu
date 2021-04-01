package main

import (
	"fmt"
	"os"
	"os/exec"
	"io/ioutil"
	"strconv"
	"strings"
	"encoding/json"
)

//
// utils
func PrintErr(m interface{}) {
	fmt.Fprintln(os.Stderr, m)
}

func IntToString(i int) string {
	return fmt.Sprintf("%d", i)
}

func DoesFileExist(path string) bool {
	_, e := os.Stat(path)
	if e == nil {
		return true
	}

	if !os.IsNotExist(e) {
		PrintErr(e)
	}
	return false
}

func Exec(command string) string {
	com := strings.Split(command, " ")

	// find command in $PATH
	path, errPath := exec.LookPath(com[0])
	if errPath != nil {
		PrintErr(errPath)
		os.Exit(1)
	}

	cmd := &exec.Cmd {
		Path: path,
		Args: com,
		Stderr: os.Stdout,
	}

	out, err := cmd.Output()
	if err != nil {
		PrintErr(err)
		os.Exit(1)
	}
	return string(out)
}

func ListActiveVms() []string {
	cmd := "pgrep 'qemu' -al | grep -ouE '\\-name .*' | sort | awk '{print $2}' | uniq"
	command := exec.Command("sh", "-c", cmd)
	command.Stderr = os.Stdout

	out, err := command.Output()
	if err != nil {
		PrintErr(err)
		os.Exit(1)
	}

	vms := strings.Split(string(out), "\n")
	var res []string
	for i := 0; i < len(vms); i++ {
		if len(vms[i]) < 1 {
			continue
		}

		res = append(res, vms[i])
	}
	return res
}

func GetVmPids(vmName string) []int {
	cmd := "pgrep 'qemu' -al | grep -uE '\\-name " + vmName + "' | cut -d' ' -f1 | uniq"
	command := exec.Command("sh", "-c", cmd)
	command.Stderr = os.Stdout
	out, err := command.Output()
	if err != nil {
		PrintErr(err)
		os.Exit(1)
	}

	pidsStr := strings.Split(string(out), "\n")
	var pidsInt []int
	for i := 0; i < len(pidsStr); i++ {
		var pid int
		if len(pidsStr[i]) < 1 {
			continue
		}

		pid, err = strconv.Atoi(pidsStr[i])
		if err != nil {
			PrintErr(err)
			os.Exit(1)
		}

		pidsInt = append(pidsInt, pid)
	}
	return pidsInt
}

func GetNumberOfCores() int {
	out := Exec("nproc")
	res, atoiErr := strconv.Atoi(strings.Split(out, "\n")[0])
	if atoiErr != nil {
		PrintErr(atoiErr)
		os.Exit(1)
	}
	return res
}


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


//
// "command" command
type PortMap struct {
	Guest int `json:"guest"`
	Host  int `json:"host"`
}

type Net struct {
	Mode    string    `json:"mode"`
	PortMap []PortMap `json:"map"`
	Tap     []string  `json:"tap"`
}

type Mount struct {
	Host string `json:"host"`
	Tag  string `json:"tag"`
}

type Display struct {
	Mode  string `json:"mode"`
	Gl    bool   `json:"gl"`
	Audio bool   `json:"audio"`
}

type Virtio struct {
	Rng     string   `json:"rng"`
	Balloon bool     `json:"balloon"`
}

type Vm struct {
	Memory  string   `json:"memory"`
	Cores   int      `json:"cores"`
	Disks   []string `json:"disks"`
	Display Display  `json:"display"`
	Virtio  Virtio   `json:"virtio"`
	Mount   []Mount  `json:"mount"`
	Net     Net      `json:"net"`
}

// build a QEMU shell command from the VM JSON config file
func Command(vmName string, breakLinesAfterArgs bool) string {
	configFile := home + "/" + vmName + ".json"

	if !DoesFileExist(configFile) {
		PrintErr("Could not find '" + vmName + "' VM")
		os.Exit(1)
	}

	// load JSON
	var vmJson Vm

	// set default core count to 2
	// if the property is omitted from the JSON file
	vmJson.Cores = 2

	configJson, err := os.Open(configFile)
	if err != nil {
		PrintErr(err)
		os.Exit(1)
	}
	bytes, errBytes := ioutil.ReadAll(configJson)
	if errBytes != nil {
		PrintErr(errBytes)
		os.Exit(1)
	}
	jsonBytes := bytes
	defer configJson.Close()

	errUnmarshal := json.Unmarshal(jsonBytes, &vmJson)
	if errUnmarshal != nil {
		PrintErr(errUnmarshal)
		os.Exit(1)
	}

	if len(vmJson.Memory) < 2 {
		PrintErr("JSON: '.memory' field is required.")
		os.Exit(1)
	}

	// build command
	var command string

	var lb string
	var li string
	if breakLinesAfterArgs {
		lb = " \\\n"
		li = "\t"
	} else {
		lb = " "
		li = ""
	}

	command = "qemu-system-x86_64 -cpu host -enable-kvm -daemonize" + lb
	command += li + "-name " + vmName + "" + lb

	// resources
	command += li + "-m " + vmJson.Memory + "" + lb

	var coreCount int
	coreCount = vmJson.Cores
	if vmJson.Cores <= 0 {
		coreCount = GetNumberOfCores()
	}

	command += li + "-smp " + IntToString(coreCount) + lb

	// display
	isGlOn := vmJson.Display.Gl

	switch vmJson.Display.Mode {
	case "sdl":
		if isGlOn {
			command += li + "-device virtio-vga,virgl=on -display sdl,gl=on,show-cursor=off" + lb
		} else {
			command += li + "-display sdl,show-cursor=off" + lb
		}
	case "gtk":
		if isGlOn {
			command += li + "-device virtio-vga,virgl=on -display gtk,gl=on,show-cursor=off" + lb
		} else {
			command += li + "-display gtk,show-cursor=off" + lb
		}
	case "spice":
		glOption := ""
		if isGlOn {
			glOption = ",gl=on"
		}

		command += li + "-vga qxl -spice unix,addr=" +
		tmp +
		"/" +
		vmName +
		spiceSocketSuffix +
		",disable-ticketing" +
		glOption +
		" -device virtio-serial -chardev spicevmc,id=vdagent,name=vdagent -device virtserialport,chardev=vdagent,name=com.redhat.spice.0" +
		lb
	case "vnc":
		// perhaps, multiple VMs might attempt to use the same VNC port?
		command += li + "-vga vnc :0" + lb
	default:
		command += li + "-display none" + lb
	}

	if vmJson.Display.Audio {
		command += li + "-device intel-hda -device hda-duplex" + lb
	}

	// virtio
	if vmJson.Virtio.Rng == "virtio" {
		command += li + "-object rng-random,id=rng0,filename=/dev/urandom -device virtio-rng-pci,rng=rng0" + lb
	}
	if vmJson.Virtio.Balloon {
		command += li + "-device virtio-balloon" + lb
	}

	// shared folders
	for i := 0; i < len(vmJson.Mount); i++ {
		fs := vmJson.Mount[i]
		command += li + "-virtfs local,path=" +
		fs.Host +
		",mount_tag=" +
		fs.Tag +
		",security_model=mapped-xattr" +
		lb
	}

	// disks
	for i := 0; i < len(vmJson.Disks); i++ {
		command += li + "-drive file=" +
			vmJson.Disks[i] +
			",media=disk" +
			lb
	}

	// network
	switch vmJson.Net.Mode {
	case "nat":
		command += li + "-net user"
		// port mapping
		for i := 0; i < len(vmJson.Net.PortMap); i++ {
			mapping := vmJson.Net.PortMap[i]
			command += ",hostfwd=tcp::" +
				IntToString(mapping.Host) +
				"-:" +
				IntToString(mapping.Guest)
		}
		command += " -net nic" + lb
	case "bridged":
		if len(vmJson.Net.Tap) < 1 {
			PrintErr("Bridged network mode selected, but no TAP interfaces specified. Disabling network.")
			command += li + "-nic none" + lb
		}

		// attach virtual interfaces
		for i := 0; i < len(vmJson.Net.Tap); i++ {
			tapInt := vmJson.Net.Tap[i]
			idx := IntToString(i)
			command += li +
				"-device virtio-net,netdev=n" +
				idx +
				" -netdev tap,id=n" +
				idx +
				",ifname=" +
				tapInt +
				",script=no,downscript=no,vhost=on" +
				lb
		}
	default:
		command += li + "-nic none" + lb
	}

	command += li +
		"-monitor unix:" +
		tmp +
		"/" +
		vmName +
		monSocketSuffix +
		",server,nowait"

	return command
}


//
// "spice" command
func OpenSpiceViewer(vmName string) {
	spiceSocket := tmp + "/" + vmName + spiceSocketSuffix

	if !DoesFileExist(spiceSocket) {
		PrintErr("'" + vmName + "' does not have an available SPICE server.")
		os.Exit(1)
	}

	uri := "spice+unix://" + spiceSocket
	Exec("remote-viewer " + uri)
}



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

		if strings.Contains(command, "-spice") {
			OpenSpiceViewer(args[1])
		}

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

	default:
		PrintErr("No such command '" + args[0] + "'")
		Help()
		os.Exit(1)
	}
}

