package helm

import (
	helm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/rest"
)

func initLiqoRepo(helmClient helm.Client) {
	// Define a public chart repository
	chartRepo := repo.Entry{
		Name: "stable",
		URL:  liqoRepo,
	}

	if err := helmClient.AddOrUpdateChartRepo(chartRepo); err != nil {
		panic(err)
	}
}

func InitializeClient(config *rest.Config) (helm.Client, error) {
	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            true,
			Linting:          true,
		},
		RestConfig: config,
	}

	client, err := helm.NewClientFromRestConf(opt)
	if err != nil {
		return nil, err
	}

	initLiqoRepo(client)

	// Add a chart-repository to the client
	return client, nil
}
