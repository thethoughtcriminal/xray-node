package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thethoughtcriminal/xray-node/internal/api"
	"github.com/thethoughtcriminal/xray-node/internal/config"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP management API",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(loadConfigPath())
			if err != nil {
				return err
			}
			server := api.New(cfg)
			fmt.Printf("xray-node API listening on http://%s\n", cfg.API.Listen)
			return server.ListenAndServe()
		},
	}
}
