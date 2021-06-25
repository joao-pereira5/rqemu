package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

//
// utils
func PrintErr(m interface{}) {
	fmt.Fprintln(os.Stderr, m)
}

func IntToString(i int) string {
	return fmt.Sprintf("%d", i)
}

func IntToHex(i int) string {
	return fmt.Sprintf("%02x", i%255)
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

func Run(command string) {
	com := strings.Split(command, " ")

	// find command in $PATH
	path, errPath := exec.LookPath(com[0])
	if errPath != nil {
		PrintErr(errPath)
		os.Exit(1)
	}

	cmd := &exec.Cmd{
		Path:   path,
		Args:   com,
		Stdout: os.Stdout,
		Stderr: os.Stdout,
		Stdin:  os.Stdin,
	}

	if err := cmd.Start(); err != nil {
		PrintErr(err)
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		PrintErr(err)
		os.Exit(1)
	}
}

func Exec(command string) string {
	com := strings.Split(command, " ")

	// find command in $PATH
	path, errPath := exec.LookPath(com[0])
	if errPath != nil {
		PrintErr(errPath)
		os.Exit(1)
	}

	cmd := &exec.Cmd{
		Path:   path,
		Args:   com,
		Stderr: os.Stdout,
		Stdin:  os.Stdin,
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
