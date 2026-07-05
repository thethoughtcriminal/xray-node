package cli

import (
	"github.com/spf13/cobra"
)

var configPath string

func Execute() error {
	root := &cobra.Command{
		Use:   "xray-node",
		Short: "Manage a 3x-ui VPN node",
	}
	root.PersistentFlags().StringVar(&configPath, "config", "", "path to config file (default /etc/xray-node/config.yaml)")

	root.AddCommand(
		newServeCmd(),
		newInboundCmd(),
		newClientCmd(),
	)
	return root.Execute()
}

func loadConfigPath() string {
	return configPath
}
