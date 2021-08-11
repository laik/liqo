package gkeprovider

import (
	"context"
	"fmt"

	flag "github.com/spf13/pflag"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
	"k8s.io/klog/v2"
)

const (
	providerPrefix = "gke"
)

type gkeProvider struct {
	credentialsPath string

	projectID string
	zone      string
	clusterID string

	endpoint    string
	serviceCIDR string
	podCIDR     string
}

func NewProviderCommandConstructor() *gkeProvider {
	return &gkeProvider{}
}

func (k *gkeProvider) ValidateParameters(flags *flag.FlagSet) (err error) {
	k.credentialsPath, err = flags.GetString(prefixedName("credentials-path"))
	if err != nil {
		return err
	}
	if k.credentialsPath == "" {
		err := fmt.Errorf("--gke.credentials-path not provided")
		return err
	}
	klog.V(3).Infof("GKE Credentials Path: %v", k.credentialsPath)

	k.projectID, err = flags.GetString(prefixedName("project-id"))
	if err != nil {
		return err
	}
	if k.projectID == "" {
		err := fmt.Errorf("--gke.project-id not provided")
		return err
	}
	klog.V(3).Infof("GKE ProjectID: %v", k.projectID)

	k.zone, err = flags.GetString(prefixedName("region"))
	if err != nil {
		return err
	}
	if k.zone == "" {
		err := fmt.Errorf("--gke.region not provided")
		return err
	}
	klog.V(3).Infof("GKE Zone: %v", k.zone)

	k.clusterID, err = flags.GetString(prefixedName("cluster-id"))
	if err != nil {
		return err
	}
	if k.clusterID == "" {
		err := fmt.Errorf("--gke.cluster-id not provided")
		return err
	}
	klog.V(3).Infof("GKE ClusterID: %v", k.clusterID)

	return nil
}

func (k *gkeProvider) GenerateCommand(ctx context.Context) (string, error) {
	svc, err := container.NewService(ctx, option.WithCredentialsFile(k.credentialsPath))
	if err != nil {
		return "", err
	}

	cluster, err := svc.Projects.Zones.Clusters.Get(k.projectID, k.zone, k.clusterID).Do()
	if err != nil {
		return "", err
	}

	k.endpoint = cluster.Endpoint
	k.serviceCIDR = cluster.ServicesIpv4Cidr
	k.podCIDR = cluster.ClusterIpv4Cidr

	// TODO: delete it
	klog.Info(k)

	return "", nil
}

func GenerateFlags(flags *flag.FlagSet) {
	subFlag := flag.NewFlagSet(providerPrefix, flag.ExitOnError)
	subFlag.SetNormalizeFunc(func(f *flag.FlagSet, name string) flag.NormalizedName {
		return flag.NormalizedName(prefixedName(name))
	})

	subFlag.String("credentials-path", "", "Path to the GCP credentials JSON file")
	subFlag.String("project-id", "", "The GCP project where your cluster is deployed in")
	subFlag.String("zone", "", "The GCP zone where your cluster is running")
	subFlag.String("cluster-id", "", "The GKE clusterID of your cluster")

	flags.AddFlagSet(subFlag)
}

func prefixedName(name string) string {
	return fmt.Sprintf("%v.%v", providerPrefix, name)
}
