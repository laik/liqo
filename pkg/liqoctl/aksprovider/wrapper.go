package aksprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-07-01/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	flag "github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

const (
	providerPrefix = "aks"
)

type aksProvider struct {
	subscriptionID    string
	resourceGroupName string
	resourceName      string

	authorizer *autorest.Authorizer

	endpoint    string
	serviceCIDR string
	podCIDR     string
}

func NewProviderCommandConstructor() *aksProvider {
	return &aksProvider{}
}

func (k *aksProvider) ValidateParameters(flags *flag.FlagSet) (err error) {
	k.subscriptionID, err = flags.GetString(prefixedName("subscription-id"))
	if err != nil {
		return err
	}
	if k.subscriptionID == "" {
		err := fmt.Errorf("--aks.subscription-id not provided")
		return err
	}
	klog.V(3).Infof("AKS SubscriptionID: %v", k.subscriptionID)

	k.resourceGroupName, err = flags.GetString(prefixedName("resource-group-name"))
	if err != nil {
		return err
	}
	if k.resourceGroupName == "" {
		err := fmt.Errorf("--aks.resource-group-name not provided")
		return err
	}
	klog.V(3).Infof("AKS ResourceGroupName: %v", k.resourceGroupName)

	k.resourceName, err = flags.GetString(prefixedName("resource-name"))
	if err != nil {
		return err
	}
	if k.resourceName == "" {
		err := fmt.Errorf("--aks.resource-name not provided")
		return err
	}
	klog.V(3).Infof("AKS ResourceName: %v", k.resourceName)

	authorizer, err := auth.NewAuthorizerFromCLI()
	if err != nil {
		return err
	}

	k.authorizer = &authorizer

	return nil
}

func (k *aksProvider) GenerateCommand(ctx context.Context) (string, error) {
	aksClient := containerservice.NewManagedClustersClient(k.subscriptionID)
	aksClient.Authorizer = *k.authorizer

	cluster, err := aksClient.Get(ctx, k.resourceGroupName, k.resourceName)
	if err != nil {
		return "", err
	}

	switch cluster.NetworkProfile.NetworkPlugin {
	case containerservice.NetworkPluginKubenet:
		k.setupKubenet(&cluster)
	case containerservice.NetworkPluginAzure:
		if err = k.setupAzureCNI(ctx, &cluster); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unknown AKS network plugin %v", cluster.NetworkProfile.NetworkPlugin)
	}

	k.endpoint = *cluster.Fqdn

	// TODO: delete it
	klog.Info(k)

	return "", nil
}

func GenerateFlags(flags *flag.FlagSet) {
	subFlag := flag.NewFlagSet(providerPrefix, flag.ExitOnError)
	subFlag.SetNormalizeFunc(func(f *flag.FlagSet, name string) flag.NormalizedName {
		return flag.NormalizedName(prefixedName(name))
	})

	subFlag.String("subscription-id", "", "The ID of the Azure Subscription of your cluster")
	subFlag.String("resource-group-name", "", "The Azure ResourceGroup name of your cluster")
	subFlag.String("resource-name", "", "The Azure Name of your cluster")

	flags.AddFlagSet(subFlag)
}

func (k *aksProvider) setupKubenet(cluster *containerservice.ManagedCluster) {
	k.podCIDR = *cluster.ManagedClusterProperties.NetworkProfile.PodCidr
	k.serviceCIDR = *cluster.ManagedClusterProperties.NetworkProfile.ServiceCidr
}

func (k *aksProvider) setupAzureCNI(ctx context.Context, cluster *containerservice.ManagedCluster) error {
	vnetSubjectID := (*cluster.AgentPoolProfiles)[0].VnetSubnetID

	networkClient := network.NewSubnetsClient(k.subscriptionID)
	networkClient.Authorizer = *k.authorizer

	vnetName, subnetName, err := parseSubnetID(*vnetSubjectID)
	if err != nil {
		return err
	}

	vnet, err := networkClient.Get(ctx, k.resourceGroupName, vnetName, subnetName, "")
	if err != nil {
		return err
	}

	k.podCIDR = *vnet.AddressPrefix
	k.serviceCIDR = *cluster.ManagedClusterProperties.NetworkProfile.ServiceCidr

	return nil
}

func parseSubnetID(subnetID string) (vnetName string, subnetName string, err error) {
	strs := strings.Split(subnetID, "/")
	l := len(strs)

	if l < 3 {
		err = fmt.Errorf("cannot parse SubnetID %v", subnetID)
		return "", "", err
	}

	return strs[l-3], strs[l-1], nil
}

func prefixedName(name string) string {
	return fmt.Sprintf("%v.%v", providerPrefix, name)
}
