package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type kubeclient struct {
	client *kubernetes.Clientset
}

func NewKubeclient() *kubeclient {
	var kube kubeclient
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	kube.client, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return &kube
}

func (kube *kubeclient) deletePodsFromDeployement(ctx context.Context, namespace string, name string) {
	dep, err := kube.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	pods, err := kube.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(dep.GetLabels()),
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster found\n", len(pods.Items))
	err = kube.client.CoreV1().Pods(namespace).Delete(ctx, pods.Items[0].Name, metav1.DeleteOptions{})
	if err != nil {
		panic(err.Error())
	}
}
