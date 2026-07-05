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
	var (
		portFlag         int
		sniFlag          string
		nonInteractive   bool
	)

	applyCmd := &cobra.Command{
		Use:   "apply <config.yaml>",
		Short: "Create or update inbound from YAML config",
		Long: `Create or update an inbound from YAML.

When run in a terminal, prompts for port (and SNI for VLESS Reality).
Use --port / --sni to pass values without prompts, or --non-interactive to use YAML as-is.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := inbound.LoadFile(args[0])
			if err != nil {
				return err
			}
			overrides, err := resolveInboundOverrides(spec, portFlag, sniFlag, nonInteractive)
			if err != nil {
				return err
			}
			if err := spec.ApplyOverrides(overrides); err != nil {
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
	}
	applyCmd.Flags().IntVar(&portFlag, "port", 0, "listen port (skips prompt)")
	applyCmd.Flags().StringVar(&sniFlag, "sni", "", "reality SNI / server name (skips prompt)")
	applyCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "use YAML values without prompts")

	cmd := &cobra.Command{
		Use:   "inbound",
		Short: "Manage inbounds",
	}
	cmd.AddCommand(
		applyCmd,
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

func resolveInboundOverrides(spec *inbound.Spec, portFlag int, sniFlag string, nonInteractive bool) (inbound.Overrides, error) {
	overrides := inbound.Overrides{
		Port: portFlag,
		SNI:  sniFlag,
	}
	if nonInteractive || !isTerminal() {
		return overrides, nil
	}

	if overrides.Port == 0 {
		port, err := promptInt("Port", spec.Port)
		if err != nil {
			return overrides, err
		}
		overrides.Port = port
	}

	if spec.IsRealityVLESS() && overrides.SNI == "" {
		defaultSNI := spec.DefaultSNI()
		if defaultSNI == "" {
			defaultSNI = "www.microsoft.com"
		}
		sni, err := promptString("SNI", defaultSNI)
		if err != nil {
			return overrides, err
		}
		overrides.SNI = sni
	}

	return overrides, nil
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
