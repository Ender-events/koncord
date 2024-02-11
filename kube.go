package main

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (bot *Bot) getPodsFromDeployement(ctx context.Context, namespace string, name string) []v1.Pod {
	dep, err := bot.kube.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	pods, err := bot.kube.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(dep.GetLabels()),
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster found\n", len(pods.Items))
	return pods.Items
}
