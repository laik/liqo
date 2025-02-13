package provider

import (
	"context"
	"io"

	module "github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	stats "github.com/virtual-kubelet/virtual-kubelet/node/api/statsv1alpha1"
	v1 "k8s.io/api/core/v1"
)

// Provider contains the methods required to implement a virtual-kubelet provider.
//
// Errors produced by these methods should implement an interface from
// github.com/liqotech/liqo/cmdInternal/errdefs package in order for the
// core logic to be able to understand the type of failure.
type Provider interface {
	module.PodLifecycleHandler

	PodMetricsProvider

	// GetContainerLogs retrieves the logs of a container by name from the provider.
	GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error)

	// RunInContainer executes a command in a container in the pod, copying data
	// between in/out/err and the container's stdin/stdout/stderr.
	RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error

	// ConfigureNode enables a provider to configure the node object that
	// will be used for Kubernetes.
	ConfigureNode(context.Context, *v1.Node)
}

// PodMetricsProvider is an optional interface that providers can implement to expose pod stats.
type PodMetricsProvider interface {
	GetStatsSummary(context.Context) (*stats.Summary, error)
}
