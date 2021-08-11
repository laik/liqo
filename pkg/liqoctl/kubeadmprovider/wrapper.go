package kubeadmprovider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	helm "github.com/mittwald/go-helm-client"
	flag "github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	liqohelm "github.com/liqotech/liqo/pkg/liqoctl/helm"
	"github.com/liqotech/liqo/pkg/utils"
)

const (
	providerPrefix             = "kubeadm"
	podCIDRParameterFilter     = `--service-cluster-ip-range=.*`
	ServiceCIDRParameterFilter = `--cluster-cidr=.*`
	kubeSystemNamespaceName    = "kube-system"
)

var kubeControllerManagerLabels = map[string]string{"component": "kube-controller-manager", "tier": "control-plane"}

type kubeadmProvider struct {
	config      *rest.Config
	PodCIDR     string
	ServiceCIDR string
	k8sClient   kubernetes.Interface
	helmClient  helm.Client
}

func NewProviderCommandConstructor() *kubeadmProvider {
	return &kubeadmProvider{}
}

func (k *kubeadmProvider) ValidateParameters(flags *flag.FlagSet) error {
	kubeconfigPath, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		kubeconfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	config, err := utils.UserConfig(kubeconfigPath)
	if err != nil {
		fmt.Printf("Unable to create client config: %s", err)
		return err
	}

	k.k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Unable to create client: %s", err)
		return err
	}

	k.config = config

	k.helmClient, err = liqohelm.InitializeClient(config)
	if err != nil {
		fmt.Printf("Unable to create helmClient: %s", err)
		return err
	}

	return nil
}

func (k *kubeadmProvider) GenerateCommand(ctx context.Context) (string, error) {
	kubeControllerSpec, err := k.k8sClient.CoreV1().Pods(kubeSystemNamespaceName).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(kubeControllerManagerLabels).AsSelector().String(),
	})
	if err != nil {
		return "", err
	}
	if len(kubeControllerSpec.Items) < 1 {
		return "", fmt.Errorf("kube-controller-manager not found")
	}
	if len(kubeControllerSpec.Items[0].Spec.Containers) != 1 {
		return "", fmt.Errorf("unexpected amount of containers in kube-controller-manager")
	}
	command := kubeControllerSpec.Items[0].Spec.Containers[0].Command
	k.PodCIDR, err = extractValueFromArgumentList(ServiceCIDRParameterFilter, command)
	if err != nil {
		return "", err
	}
	k.ServiceCIDR, err = extractValueFromArgumentList(ServiceCIDRParameterFilter, command)
	if err != nil {
		return "", err
	}
	return "", nil
}

func GenerateFlags(*flag.FlagSet) {}

func extractValueFromArgumentList(argumentMatch string, argumentList []string) (string, error) {
	for index := range argumentList {
		matched, _ := regexp.Match(argumentMatch, []byte(argumentList[index]))
		if matched {
			return strings.Split(argumentList[index], "=")[1], nil
		}
	}
	return "", fmt.Errorf("argument not found")
}
