package main

import (
	"context"
	"time"
)

func main() {
	for {
		kube := NewKubeclient()
		kube.deletePodsFromDeployement(context.TODO(), "default", "kubernetes-bootcamp")
		time.Sleep(10 * time.Second)
	}
}
