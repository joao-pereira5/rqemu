package main

import (
	"os"
)

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
