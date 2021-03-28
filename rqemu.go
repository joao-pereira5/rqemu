package main

import (
	"fmt"
	"os"
	"os/exec"
	"io/ioutil"
	"strconv"
	"strings"
	"regexp"
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

func GetNumberOfCores() int {
	out, err := exec.Command("nproc").Output()
	if err != nil {
		PrintErr(err)
		os.Exit(1)
	}

	res, atoiErr := strconv.Atoi(strings.Split(string(out), "\n")[0])
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
	Host int  `json:"host"`
}

type Net struct {
	Mode string       `json:"mode"`
	PortMap []PortMap `json:"map"`
	Tap []string      `json:"tap"`
}

type Mount struct {
	Host string `json:"host"`
	Tag string  `json:"tag"`
}

type Display struct {
	Mode string `json:"mode"`
	Gl bool     `json:"gl"`
}

type Vm struct {
	Memory  string   `json:"memory"`
	Cores   int      `json:"cores"`
	Disks   []string `json:"disks"`
	Display Display  `json:"display"`
	Rng     string   `json:"rng"`
	Balloon bool     `json:"balloon"`
	Audio   bool     `json:"audio"`
	Mount   []Mount  `json:"mount"`
	Net     Net      `json:"net"`
}

// build a QEMU shell command from the VM JSON config file
func Command(vmName string) string {
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
	jsonString := string(bytes)
	defer configJson.Close()

	re := regexp.MustCompile(`.*//.*\n`)
	jsonString = re.ReplaceAllString(jsonString, "")
	//fmt.Println(jsonString)

	jsonBytes := []byte(jsonString)
	errUnmarshal := json.Unmarshal(jsonBytes, &vmJson)
	if errUnmarshal != nil {
		PrintErr(errUnmarshal)
		os.Exit(1)
	}

	if len(vmJson.Memory) < 2 {
		PrintErr("JSON: '.memory' field is required.")
		os.Exit(1)
	}
	if len(vmJson.Disks) < 1 {
		PrintErr("JSON: '.disks' field is required.")
		os.Exit(1)
	}

	// build command
	var command string
	lb := " \\\n"

	command = "qemu-system-x86_64 -cpu host -enable-kvm -daemonize" + lb
	command += "\t-name \"" + vmName + "\"" + lb

	// resources
	command += "\t-m \"" + vmJson.Memory + "\"" + lb

	var coreCount int
	coreCount = vmJson.Cores
	if vmJson.Cores <= 0 {
		coreCount = GetNumberOfCores()
	}
	command += "\t-smp " + IntToString(coreCount) + lb

	// display
	isGlOn := vmJson.Display.Gl

	switch vmJson.Display.Mode {
	case "sdl":
		if isGlOn {
			command += "\t-device virtio-vga,virgl=on -display sdl,gl=on,show-cursor=off" + lb
		} else {
			command += "\t-display sdl,show-cursor=off" + lb
		}
	case "gtk":
		if isGlOn {
			command += "\t-device virtio-vga,virgl=on -display gtk,gl=on,show-cursor=off" + lb
		} else {
			command += "\t-display gtk,show-cursor=off" + lb
		}
	case "spice":
		command += "\t-vga qxl -spice unix,addr=\"" +
		tmp +
		"/" +
		vmName +
		spiceSocketSuffix +
		"\",disable-ticketing -device virtio-serial -chardev spicevmc,id=vdagent,name=vdagent -device virtserialport,chardev=vdagent,name=com.redhat.spice.0" +
		lb
	case "vnc":
		// perhaps, multiple VMs might attempt to use the same VNC port?
		command += "\t-vga vnc :0" + lb
	default:
		command += "\t-display none" + lb
	}

	// virtio
	if vmJson.Rng == "virtio" {
		command += "\t-object rng-random,id=rng0,filename=\"/dev/urandom\" -device virtio-rng-pci,rng=rng0" + lb
	}
	if vmJson.Balloon {
		command += "\t-device virtio-balloon" + lb
	}

	// audio
	if vmJson.Audio {
		command += "\t-device intel-hda -device hda-duplex" + lb
	}

	// shared folders
	for i := 0; i < len(vmJson.Mount); i++ {
		fs := vmJson.Mount[i]
		command += "\t-virtfs local,path=\"" +
		fs.Host +
		"\",mount_tag=\"" +
		fs.Tag +
		"\",security_model=mapped-xattr" +
		lb
	}

	// disks
	for i := 0; i < len(vmJson.Disks); i++ {
		command += "\t-drive file=\"" +
			vmJson.Disks[i] +
			"\",media=disk" +
			lb
	}

	// network
	switch vmJson.Net.Mode {
	case "nat":
		command += "\t-net user"
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
		// attach virtual interfaces
		for i := 0; i < len(vmJson.Net.Tap); i++ {
			tapInt := vmJson.Net.Tap[i]
			idx := IntToString(i)
			command += "\t-device virtio-net,netdev=n" +
				idx +
				" -netdev tap,id=n" +
				idx +
				",ifname=" +
				tapInt +
				",script=no,downscript=no,vhost=on" +
				lb
		}
	default:
		command += "\t-nic none" + lb
	}

	command += "\t-monitor unix:\"" +
		tmp +
		"/" +
		vmName +
		monSocketSuffix +
		"\",server,nowait"

	return command
}


//
// "help" command
func Help() {
	msg := `NAME
	rqemu - interactive command line QEMU user interface

USAGE
	rqemu <command> [<sub command>...]
	rqemu help <command>

COMMANDS
	create
	edit
	start
	help
	stop
	command print QEMU command from JSON config file
	ls      list active or inactive VMs
	locate  print configuration directory
	monitor connect to a VM's QEMU monitor
	spice   connect to a VM's SPICE server`

	fmt.Println(msg)
}

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
	qemu-system-x86_64 -cpu host -enable-kvm -name "win10" -m 6G -smp 2 -nic none -display none -drive file="win10.qcow2",media=disk -monitor unix:"./tmp/win_mon.sock",server,nowait`

	default:
		Help()
		return
	}
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
	tmp = home + "/tmp"

	os.MkdirAll(home, os.ModePerm)
	os.MkdirAll(tmp, os.ModePerm)

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
		fmt.Println(Command(args[1]))

	default:
		PrintErr("No such command '" + args[0] + "'")
		Help()
		os.Exit(1)
	}
}

