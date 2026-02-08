package main

import (
	"context"
	"os"

	"github.com/gigiozzz/kubedial/common/provider"
	"github.com/spf13/cobra"
)

var (
	commanderURL string
	agentToken   string
	agentName    string
	clusterName  string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "kubedialer",
	Short: "Kubedialer agent for applying manifests from kubecommander",
	Long: `Kubedialer is a Kubernetes agent that pulls commands from kubecommander
and applies/deletes manifests on the local cluster.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx := provider.InitLoggingWithContext(cmd.Context())
		cmd.SetContext(ctx)
	},
}

// Execute executes the root command
func Execute() {
	ctx := context.Background()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&commanderURL, "commander-url", "", "URL of the kubecommander server")
	rootCmd.PersistentFlags().StringVar(&agentToken, "agent-token", "", "Bearer token for authentication")
	rootCmd.PersistentFlags().StringVar(&agentName, "agent-name", "", "Name of this agent")
	rootCmd.PersistentFlags().StringVar(&clusterName, "cluster-name", "", "Name of the Kubernetes cluster")
}
