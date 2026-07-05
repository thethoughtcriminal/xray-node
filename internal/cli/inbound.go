package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/thethoughtcriminal/xray-node/internal/config"
	"github.com/thethoughtcriminal/xray-node/internal/inbound"
	"github.com/thethoughtcriminal/xray-node/internal/service"
)

func newInboundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inbound",
		Short: "Manage inbounds",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "apply <config.yaml>",
			Short: "Create or update inbound from YAML config",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				spec, err := inbound.LoadFile(args[0])
				if err != nil {
					return err
				}
				cfg, err := config.Load(loadConfigPath())
				if err != nil {
					return err
				}
				node := service.New(cfg)
				result, err := node.ApplyInbound(spec)
				if err != nil {
					return err
				}
				return printJSON(result)
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List inbounds",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := config.Load(loadConfigPath())
				if err != nil {
					return err
				}
				node := service.New(cfg)
				items, err := node.ListInbounds()
				if err != nil {
					return err
				}
				return printJSON(items)
			},
		},
	)
	return cmd
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}
