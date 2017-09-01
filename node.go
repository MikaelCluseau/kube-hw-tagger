package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	nodeName string
	dryRun   bool
)

func init() {
	nodeName = os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatal("Must have NODE_NAME environment variable")
	}
	flag.BoolVar(&dryRun, "dry-run", false, "dry run (don't update nodes)")
}

func nodeUpdateLabels(prefix string, keys map[string]bool, value string) {
	knodes := kube.Core().Nodes()
	node, err := knodes.Get(nodeName, metav1.GetOptions{})
	if err != nil {
		log.Fatal("Failed to fetch node ", nodeName, ": ", err)
	}

	update := false

	for key, _ := range node.Labels {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if keys[key] {
			continue
		}
		log.Print("Removing label ", key)
		delete(node.Labels, key)
		update = true
	}

	for key, _ := range keys {
		if currentValue, ok := node.Labels[key]; ok && currentValue == value {
			continue
		} else if !ok {
			log.Print("Adding label ", key, "=", value)
		} else {
			log.Print("Setting label ", key, "=", value)
		}
		node.Labels[key] = value
		update = true
	}

	if !update || dryRun {
		return
	}

	log.Print("Updating node ", nodeName)
	if _, err := knodes.Update(node); err != nil {
		log.Fatal("Unable to update node: ", err)
	}
}

func jsonLabels(value map[string]string) string {
	ba, _ := json.MarshalIndent(value, "", "  ")
	return string(ba)
}
