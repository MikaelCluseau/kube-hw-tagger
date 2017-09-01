package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MikaelCluseau/kube-hw-tagger/pkg/udev"
)

func main() {
	flag.Parse()

	k8sSetup()

	for _, watcher := range []*deviceWatcher{
		{
			Prefix:    "node-devices.alpha.kubernetes.io",
			Subsystem: "block",
			Filter:    filterBlock,
			// Notes:
			// - ID_SERIAL and DM_UUID are too long
			// - DM_NAME catches LVM devices, including the docker pool that we propbably don't want
			IdProperties: map[string]string{
				"wwn": "ID_WWN",
				"sn":  "ID_SERIAL_SHORT",
			},
		},
	} {
		go watcher.Run()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill, syscall.SIGTERM)
	s := <-sig
	log.Print("Got signal ", s, ", exiting.")
	os.Exit(0)
}

func filterBlock(dev *udev.Device) bool {
	if dev.DevType != "disk" {
		return false
	}
	return true
}
