package cmd

import (
	flag "github.com/spf13/pflag"

	"github.com/liqotech/liqo/pkg/liqoctl/aksprovider"
	"github.com/liqotech/liqo/pkg/liqoctl/eksprovider"
	"github.com/liqotech/liqo/pkg/liqoctl/gkeprovider"
	"github.com/liqotech/liqo/pkg/liqoctl/kubeadmprovider"
)

var providers = []string{"kubeadm", "eks", "gke", "aks"}

var providerInitFunc = map[string]func(*flag.FlagSet){
	"kubeadm": kubeadmprovider.GenerateFlags,
	"eks":     eksprovider.GenerateFlags,
	"gke":     gkeprovider.GenerateFlags,
	"aks":     aksprovider.GenerateFlags,
}
