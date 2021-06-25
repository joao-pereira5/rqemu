package main

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)

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

type Display struct {
	Mode  string `json:"mode"`
	Gl    bool   `json:"gl"`
	Audio bool   `json:"audio"`
}

type Virtio struct {
	Rng     string `json:"rng"`
	Balloon bool   `json:"balloon"`
}

type Vm struct {
	Memory  string   `json:"memory"`
	Cores   int      `json:"cores"`
	Disks   []string `json:"disks"`
	Display Display  `json:"display"`
	Virtio  Virtio   `json:"virtio"`
	Mount   []string `json:"mount"`
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
			command += li + "-device virtio-vga,virgl=on -display sdl,gl=on,show-cursor=on" + lb
		} else {
			command += li + "-display sdl,show-cursor=off" + lb
		}
	case "gtk":
		if isGlOn {
			command += li + "-device virtio-vga,virgl=on -display gtk,gl=on,show-cursor=on" + lb
		} else {
			command += li + "-display gtk,show-cursor=off" + lb
		}
	case "spice":
		glOption := ""
		if isGlOn {
			glOption = ",gl=on"
		}

		command += li + "-vga qxl -spice addr=" +
			tmp +
			"/" +
			vmName +
			spiceSocketSuffix +
			",disable-ticketing=on,unix=on" +
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
	mountTagPrefix := "virtfs"
	for i := 0; i < len(vmJson.Mount); i++ {
		fs := vmJson.Mount[i]
		command += li + "-virtfs local,path=" +
			fs +
			",mount_tag=" +
			mountTagPrefix +
			IntToString(i) +
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

		rand.Seed(time.Now().UnixNano())
		macPrefix := "22:d0:46:" +
			IntToHex(rand.Int()) +
			":" +
			IntToHex(rand.Int()) +
			":"

		// attach virtual interfaces
		for i := 0; i < len(vmJson.Net.Tap); i++ {
			tapInt := vmJson.Net.Tap[i]
			idx := IntToString(i)
			command += li +
				"-device virtio-net,netdev=n" +
				idx +
				",mac=" +
				macPrefix +
				IntToHex(i) +
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
