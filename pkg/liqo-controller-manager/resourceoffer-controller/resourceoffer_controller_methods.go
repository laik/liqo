package resourceoffercontroller

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	configv1alpha1 "github.com/liqotech/liqo/apis/config/v1alpha1"
	sharingv1alpha1 "github.com/liqotech/liqo/apis/sharing/v1alpha1"
	"github.com/liqotech/liqo/pkg/consts"
	crdclient "github.com/liqotech/liqo/pkg/crdClient"
	"github.com/liqotech/liqo/pkg/utils"
	foreigncluster "github.com/liqotech/liqo/pkg/utils/foreignCluster"
	"github.com/liqotech/liqo/pkg/virtualKubelet"
	"github.com/liqotech/liqo/pkg/vkMachinery/forge"
)

// WatchConfiguration watches a ClusterConfig for reconciling updates on ClusterConfig.
func (r *ResourceOfferReconciler) WatchConfiguration(kubeconfigPath string, localCrdClient *crdclient.CRDClient, wg *sync.WaitGroup) {
	defer wg.Done()
	utils.WatchConfiguration(func(configuration *configv1alpha1.ClusterConfig) {
		r.setConfig(configuration)
	}, localCrdClient, kubeconfigPath)
}

func (r *ResourceOfferReconciler) getConfig() *configv1alpha1.ClusterConfig {
	r.configurationMutex.RLock()
	defer r.configurationMutex.RUnlock()

	return r.configuration.DeepCopy()
}

func (r *ResourceOfferReconciler) setConfig(config *configv1alpha1.ClusterConfig) {
	r.configurationMutex.Lock()
	defer r.configurationMutex.Unlock()

	if r.configuration == nil {
		r.configuration = config
		return
	}
	if !reflect.DeepEqual(r.configuration, config) {
		r.configuration = config
	}
}

// setControllerReference sets owner reference to the related ForeignCluster.
func (r *ResourceOfferReconciler) setControllerReference(
	ctx context.Context, resourceOffer *sharingv1alpha1.ResourceOffer) error {
	// get the foreign cluster by clusterID label
	foreignCluster, err := foreigncluster.GetForeignClusterByID(ctx, r.Client, resourceOffer.Spec.ClusterId)
	if err != nil {
		klog.Error(err)
		return err
	}

	// add controller reference, if it is not already set
	if err := controllerutil.SetControllerReference(foreignCluster, resourceOffer, r.Scheme); err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

// setResourceOfferPhase checks if the resource request can be accepted and set its phase accordingly.
func (r *ResourceOfferReconciler) setResourceOfferPhase(
	ctx context.Context, resourceOffer *sharingv1alpha1.ResourceOffer) error {
	// we want only to care about resource offers with a pending status
	if resourceOffer.Status.Phase != "" && resourceOffer.Status.Phase != sharingv1alpha1.ResourceOfferPending {
		return nil
	}

	switch r.getConfig().Spec.AdvertisementConfig.IngoingConfig.AcceptPolicy {
	case configv1alpha1.AutoAcceptMax:
		resourceOffer.Status.Phase = sharingv1alpha1.ResourceOfferAccepted
	case configv1alpha1.ManualAccept:
		// require a manual accept/refuse
		resourceOffer.Status.Phase = sharingv1alpha1.ResourceOfferManualActionRequired
	}
	return nil
}

// checkVirtualKubeletDeployment checks the existence of the VirtualKubelet Deployment
// and sets its status in the ResourceOffer accordingly.
func (r *ResourceOfferReconciler) checkVirtualKubeletDeployment(
	ctx context.Context, resourceOffer *sharingv1alpha1.ResourceOffer) error {
	virtualKubeletDeployment, err := r.getVirtualKubeletDeployment(ctx, resourceOffer)
	if err != nil {
		klog.Error(err)
		return err
	}

	if virtualKubeletDeployment == nil {
		resourceOffer.Status.VirtualKubeletStatus = sharingv1alpha1.VirtualKubeletStatusNone
	} else if resourceOffer.Status.VirtualKubeletStatus != sharingv1alpha1.VirtualKubeletStatusDeleting {
		// there is a virtual kubelet deployment and the phase is not deleting
		resourceOffer.Status.VirtualKubeletStatus = sharingv1alpha1.VirtualKubeletStatusCreated
	}
	return nil
}

// createVirtualKubeletDeployment creates the VirtualKubelet Deployment.
func (r *ResourceOfferReconciler) createVirtualKubeletDeployment(
	ctx context.Context, resourceOffer *sharingv1alpha1.ResourceOffer) error {
	name := virtualKubelet.VirtualKubeletPrefix + resourceOffer.Spec.ClusterId
	nodeName := virtualKubelet.VirtualNodePrefix + resourceOffer.Spec.ClusterId

	namespace := resourceOffer.Namespace
	remoteClusterID := resourceOffer.Spec.ClusterId

	// create the base resources
	vkServiceAccount := forge.VirtualKubeletServiceAccount(name, namespace)
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, vkServiceAccount, func() error {
		return nil
	})
	if err != nil {
		klog.Error(err)
		return err
	}
	klog.V(5).Infof("[%v] ServiceAccount %s/%s reconciled: %s", remoteClusterID, vkServiceAccount.Namespace, vkServiceAccount.Name, op)

	vkClusterRoleBinding := forge.VirtualKubeletClusterRoleBinding(name, namespace, remoteClusterID)
	op, err = controllerutil.CreateOrUpdate(ctx, r.Client, vkClusterRoleBinding, func() error {
		return nil
	})
	if err != nil {
		klog.Error(err)
		return err
	}

	klog.V(5).Infof("[%v] ClusterRoleBinding %s reconciled: %s", remoteClusterID, vkClusterRoleBinding.Name, op)

	// forge the virtual Kubelet
	vkDeployment, err := forge.VirtualKubeletDeployment(
		remoteClusterID, name, namespace, r.liqoNamespace, r.virtualKubeletImage,
		r.initVirtualKubeletImage, nodeName, r.clusterID.GetClusterID())
	if err != nil {
		klog.Error(err)
		return err
	}

	op, err = controllerutil.CreateOrUpdate(ctx, r.Client, vkDeployment, func() error {
		// set the "owner" object name in the annotation to be able to reconcile deployment changes
		vkDeployment.Annotations[resourceOfferAnnotation] = resourceOffer.GetName()
		return nil
	})
	if err != nil {
		klog.Error(err)
		return err
	}
	klog.V(5).Infof("[%v] Deployment %s/%s reconciled: %s", remoteClusterID, vkDeployment.Namespace, vkDeployment.Name, op)

	if op == controllerutil.OperationResultCreated {
		msg := fmt.Sprintf("[%v] Launching virtual-kubelet in namespace %v", remoteClusterID, namespace)
		klog.Info(msg)
		r.eventsRecorder.Event(resourceOffer, "Normal", "VkCreated", msg)
	}

	controllerutil.AddFinalizer(resourceOffer, consts.VirtualKubeletFinalizer)
	resourceOffer.Status.VirtualKubeletStatus = sharingv1alpha1.VirtualKubeletStatusCreated
	return nil
}

// deleteVirtualKubeletDeployment deletes the VirtualKubelet Deployment.
func (r *ResourceOfferReconciler) deleteVirtualKubeletDeployment(
	ctx context.Context, resourceOffer *sharingv1alpha1.ResourceOffer) error {
	virtualKubeletDeployment, err := r.getVirtualKubeletDeployment(ctx, resourceOffer)
	if err != nil {
		klog.Error(err)
		return err
	}
	if virtualKubeletDeployment == nil || !virtualKubeletDeployment.DeletionTimestamp.IsZero() {
		return nil
	}

	if err := r.Client.Delete(ctx, virtualKubeletDeployment); err != nil {
		klog.Error(err)
		return err
	}

	controllerutil.RemoveFinalizer(resourceOffer, consts.VirtualKubeletFinalizer)
	msg := fmt.Sprintf("[%v] Deleting virtual-kubelet in namespace %v", resourceOffer.Spec.ClusterId, resourceOffer.Namespace)
	klog.Info(msg)
	r.eventsRecorder.Event(resourceOffer, "Normal", "VkDeleted", msg)
	return nil
}

// deleteClusterRoleBinding deletes the ClusterRoleBinding related to a VirtualKubelet if the deployment does not exist.
func (r *ResourceOfferReconciler) deleteClusterRoleBinding(
	ctx context.Context, resourceOffer *sharingv1alpha1.ResourceOffer) error {
	labels := forge.ClusterRoleLabels(resourceOffer.Spec.ClusterId)

	if err := r.Client.DeleteAllOf(ctx, &rbacv1.ClusterRoleBinding{}, client.MatchingLabels(labels)); err != nil {
		klog.Error(err)
		return err
	}
	return nil
}

// getVirtualKubeletDeployment returns the VirtualKubelet Deployment given a ResourceOffer.
func (r *ResourceOfferReconciler) getVirtualKubeletDeployment(
	ctx context.Context, resourceOffer *sharingv1alpha1.ResourceOffer) (*appsv1.Deployment, error) {
	var deployList appsv1.DeploymentList
	labels := forge.VirtualKubeletLabels(resourceOffer.Spec.ClusterId)
	if err := r.Client.List(ctx, &deployList, client.MatchingLabels(labels)); err != nil {
		klog.Error(err)
		return nil, err
	}

	if len(deployList.Items) == 0 {
		klog.V(4).Infof("[%v] no VirtualKubelet deployment found", resourceOffer.Spec.ClusterId)
		return nil, nil
	} else if len(deployList.Items) > 1 {
		err := fmt.Errorf("[%v] more than one VirtualKubelet deployment found", resourceOffer.Spec.ClusterId)
		klog.Error(err)
		return nil, err
	}

	return &deployList.Items[0], nil
}

type kubeletDeletePhase string

const (
	kubeletDeletePhaseNone         kubeletDeletePhase = "None"
	kubeletDeletePhaseDrainingNode kubeletDeletePhase = "DrainingNode"
	kubeletDeletePhaseNodeDeleted  kubeletDeletePhase = "NodeDeleted"
)

// getDeleteVirtualKubeletPhase returns the delete phase for the VirtualKubelet created basing on the
// given ResourceOffer.
func getDeleteVirtualKubeletPhase(resourceOffer *sharingv1alpha1.ResourceOffer) kubeletDeletePhase {
	notAccepted := !isAccepted(resourceOffer)
	deleting := !resourceOffer.DeletionTimestamp.IsZero()
	desiredDelete := !resourceOffer.Spec.WithdrawalTimestamp.IsZero()
	nodeDrained := !controllerutil.ContainsFinalizer(resourceOffer, consts.NodeFinalizer)

	// if the ResourceRequest has not been accepted by the local cluster,
	// or it has a DeletionTimestamp not equal to zero (the resource has been deleted),
	// or it has a WithdrawalTimestamp not equal to zero (the remote cluster asked for its graceful deletion),
	// the VirtualKubelet is in a terminating phase, otherwise return the None phase.
	if notAccepted || deleting || desiredDelete {
		// if the liqo.io/node finalizer is not set, the remote cluster has been drained and the node has been deleted,
		// we can then proceed with the VirtualKubelet deletion.
		if nodeDrained {
			return kubeletDeletePhaseNodeDeleted
		}

		// if the finalizer is still present, the node draining has not completed yet, we have to wait before to
		// continue the unpeering process.
		return kubeletDeletePhaseDrainingNode
	}
	return kubeletDeletePhaseNone
}

// isAccepted checks if a ResourceOffer is in Accepted phase.
func isAccepted(resourceOffer *sharingv1alpha1.ResourceOffer) bool {
	return resourceOffer.Status.Phase == sharingv1alpha1.ResourceOfferAccepted
}
