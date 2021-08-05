/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/liqotech/liqo/pkg/liqoctl"
)

// generateInstallCmd represents the generateInstall command
var generateInstallCmd = &cobra.Command{
	Use:   "generate-install-command",
	Short: "generate the helm install command for the selected cluster",
	Long: `

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: liqoctl.HandleGenerateInstall,
}

func init() {
	rootCmd.AddCommand(generateInstallCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	generateInstallCmd.Flags().StringP("provider", "p", "kubeadm", "The provider for the cluster")
	generateInstallCmd.Flags().StringP("cluster-name", "c", "", "The name of target-cluster to retrieve it from the provider")
	generateInstallCmd.Flags().StringP("output", "o", "values", "the output format. It can be: 'values','command' ")


	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// generateInstallCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
