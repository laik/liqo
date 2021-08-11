package liqoctl

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/liqotech/liqo/pkg/liqoctl/eksprovider"
	"github.com/liqotech/liqo/pkg/liqoctl/kubeadmprovider"
)

func HandleGenerateInstall(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	providerName, err := cmd.Flags().GetString("provider")
	if err != nil {
		return
	}
	provider := getProviderInstance(providerName)
	if provider == nil {
		fmt.Printf("Provider of type %s not found", providerName)
		return
	}

	err = provider.ValidateParameters(cmd.Flags())
	if err != nil {
		fmt.Printf("Unable to initialize configuration: %v", err)
		return
	}
	_, err = provider.GenerateCommand(ctx)
	if err != nil {
		fmt.Printf("Unable to initialize configuration: %v", err)
		return
	}

}

func getProviderInstance(providerType string) InstallCommandGenerator {
	switch providerType {
	case "kubeadm":
		return kubeadmprovider.NewProviderCommandConstructor()
	case "eks":
		return eksprovider.NewProviderCommandConstructor()
	default:
		return nil
	}
}
