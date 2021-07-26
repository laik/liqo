package net

import (
	"context"
	"strings"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo/test/e2e/testutils/util"
)

// EnsureNetTesterPods creates the NetTest pods and waits for them to be ready.
func EnsureNetTesterPods(ctx context.Context, homeClient kubernetes.Interface, homeID, remoteID string) error {
	ns, err := util.EnforceNamespace(ctx, homeClient, homeID, TestNamespaceName)
	if err != nil && !kerrors.IsAlreadyExists(err) {
		klog.Error(err)
		return err
	}
	podRemote := forgeTesterPod(image, podTesterRemoteCl, ns.Name, true, remoteID)
	_, err = homeClient.CoreV1().Pods(ns.Name).Create(ctx, podRemote, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		klog.Error(err)
		return err
	}
	podLocal := forgeTesterPod(image, podTesterLocalCl, ns.Name, false, "")
	_, err = homeClient.CoreV1().Pods(ns.Name).Create(ctx, podLocal, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		klog.Error(err)
		return err
	}
	return nil
}

// CheckTesterPods retrieves the netTest pods and returns true if all the pods are up and ready.
func CheckTesterPods(ctx context.Context, homeClient, foreignClient kubernetes.Interface, homeClusterID string) bool {
	reflectedNamespace := TestNamespaceName + "-" + homeClusterID
	return util.IsPodUp(ctx, homeClient, TestNamespaceName, podTesterLocalCl, true) &&
		util.IsPodUp(ctx, homeClient, TestNamespaceName, podTesterRemoteCl, true) &&
		util.IsPodUp(ctx, foreignClient, reflectedNamespace, podTesterRemoteCl, false)
}

// forgeTesterPod deploys the Remote pod of the test.
func forgeTesterPod(image, podName, namespace string, isRemote bool, remoteID string) *v1.Pod {
	nodeName := ""
	if isRemote {
		nodeName = strings.Join([]string{"liqo", remoteID}, "-")
	}

	pod1 := v1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels:    map[string]string{"app": podName},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "tester",
					Image:           image,
					Resources:       v1.ResourceRequirements{},
					ImagePullPolicy: "IfNotPresent",
					Ports: []v1.ContainerPort{{
						ContainerPort: 80,
					}},
				},
			},
			NodeName: nodeName,
		},
		Status: v1.PodStatus{},
	}
	return &pod1
}
