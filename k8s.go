package main

import (
	"log"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kube *kubernetes.Clientset
)

func k8sSetup() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Print(err)
		log.Print("Falling back to local Kubernetes client configuration")
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			log.Fatal("No kubernetes config: ", err)
		}
	}

	if client, err := kubernetes.NewForConfig(config); err == nil {
		kube = client
	} else {
		log.Fatal("Unable to load Kubernetes config: ", err)
	}
}
