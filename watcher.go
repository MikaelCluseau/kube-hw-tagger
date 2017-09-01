package main

import (
	"crypto/sha1"
	"encoding/base32"
	"log"
	"regexp"
	"strings"

	"github.com/MikaelCluseau/kube-hw-tagger/pkg/udev"
)

var (
	invalidChars = regexp.MustCompile("[^-A-Za-z0-9_.]")
)

type syncEvent struct {
	// true for the initial phase (list) finished marker
	InitFinished bool
	// The event if any
	Event *udev.DeviceEvent
}

type deviceWatcher struct {
	// Label prefix
	Prefix string
	// Subsystem to watch ("block", "net", ...)
	Subsystem string
	// Regexp filter on the device type (".*" for wildcard)
	Filter func(*udev.Device) bool
	// Properties allow for the ID, in priority order
	IdProperties map[string]string

	knownKeys map[string]bool
	canSync   bool
}

func (w *deviceWatcher) Run() {
	w.knownKeys = map[string]bool{}

	syncEvents := make(chan syncEvent, 10)
	go w.watch(syncEvents)

	for syncEvent := range syncEvents {
		if syncEvent.InitFinished {
			w.canSync = true
			w.sync()
		}

		// filter to have only the disks
		event := syncEvent.Event
		if event == nil || event.Device == nil {
			continue
		}
		if !w.Filter(event.Device) {
			continue
		}

		for prefix, property := range w.IdProperties {
			id := event.Device.Properties[property]
			if id == "" {
				continue
			}

			key := w.keyPrefix() + event.Device.DevType + "-" + prefix + "-" + id

			// make a valid name
			key = validKey(key)

			switch event.Action {
			case "add":
				log.Print("Add key: ", key)
				w.knownKeys[key] = true
				w.sync()
			case "remove":
				log.Print("Remove key: ", key)
				delete(w.knownKeys, key)
				w.sync()
			default:
				log.Print("WARN: unknown action: ", event.Action)
			}
		}
	}
}

func validKey(key string) string {
	parts := strings.SplitN(key, "/", 2)

	if len(parts[1]) > 63 {
		// need to truncate the name part
		h := sha1.Sum([]byte(parts[1]))
		suffix := base32.StdEncoding.EncodeToString(h[:])[0:5]
		newName := parts[1][0:63-5-1] + "-" + strings.ToLower(suffix)
		log.Print("Truncating name ", parts[1], " to ", newName)
		parts[1] = newName
	}

	parts[1] = invalidChars.ReplaceAllString(parts[1], "-")

	return parts[0] + "/" + parts[1]
}

func (w *deviceWatcher) sync() {
	if !w.canSync {
		return
	}
	nodeUpdateLabels(w.keyPrefix(), w.knownKeys, "present")
}

func (w *deviceWatcher) keyPrefix() string {
	return w.Prefix + "/" + w.Subsystem + "-"
}

func (w *deviceWatcher) watch(events chan syncEvent) {
	for _, device := range udev.SubsystemDevices(w.Subsystem) {
		events <- syncEvent{
			InitFinished: false,
			Event: &udev.DeviceEvent{
				Action: "add",
				Device: device,
			},
		}
	}

	events <- syncEvent{true, nil}

	deviceEvents := make(chan udev.DeviceEvent, 1)
	go func() {
		err := udev.MonitorDeviceEvents(w.Subsystem, deviceEvents)
		log.Fatal("watch on subsystem ", w.Subsystem, " failed: ", err)
	}()

	for deviceEvent := range deviceEvents {
		events <- syncEvent{false, &deviceEvent}
	}
	log.Fatal("watch on subsystem ", w.Subsystem, " finished")
}
