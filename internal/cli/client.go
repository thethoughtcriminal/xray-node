package cli

import (
	"github.com/spf13/cobra"

	"github.com/thethoughtcriminal/xray-node/internal/config"
	"github.com/thethoughtcriminal/xray-node/internal/service"
)

func newClientCmd() *cobra.Command {
	var (
		inboundRemark string
		email         string
		clientUUID    string
		subID         string
		flow          string
		auth          string
		totalGB       int64
		expiryDays    int
		limitIP       int
	)

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Manage clients",
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add client to inbound",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(loadConfigPath())
			if err != nil {
				return err
			}
			node := service.New(cfg)
			client, err := node.AddClient(service.AddClientInput{
				InboundRemark: inboundRemark,
				Email:         email,
				UUID:          clientUUID,
				SubID:         subID,
				Flow:          flow,
				Auth:          auth,
				TotalGB:       totalGB,
				ExpiryDays:    expiryDays,
				LimitIP:       limitIP,
				Enable:        true,
			})
			if err != nil {
				return err
			}
			return printJSON(client)
		},
	}
	addCmd.Flags().StringVar(&inboundRemark, "inbound", "", "inbound remark (required)")
	addCmd.Flags().StringVar(&email, "email", "", "client email (required)")
	addCmd.Flags().StringVar(&clientUUID, "uuid", "", "client UUID (auto for vless/vmess)")
	addCmd.Flags().StringVar(&subID, "sub-id", "", "subscription id (auto)")
	addCmd.Flags().StringVar(&flow, "flow", "", "vless flow (default xtls-rprx-vision)")
	addCmd.Flags().StringVar(&auth, "auth", "", "hysteria auth password (auto)")
	addCmd.Flags().Int64Var(&totalGB, "total-gb", 0, "traffic limit in GB (0 = unlimited)")
	addCmd.Flags().IntVar(&expiryDays, "expiry-days", 0, "expiry in days (0 = unlimited)")
	addCmd.Flags().IntVar(&limitIP, "limit-ip", 0, "IP limit (0 = unlimited)")
	_ = addCmd.MarkFlagRequired("inbound")
	_ = addCmd.MarkFlagRequired("email")

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable client",
		RunE: func(cmd *cobra.Command, args []string) error {
			return setClientEnabled(true, inboundRemark, email)
		},
	}
	enableCmd.Flags().StringVar(&inboundRemark, "inbound", "", "inbound remark (required)")
	enableCmd.Flags().StringVar(&email, "email", "", "client email (required)")
	_ = enableCmd.MarkFlagRequired("inbound")
	_ = enableCmd.MarkFlagRequired("email")

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable client",
		RunE: func(cmd *cobra.Command, args []string) error {
			return setClientEnabled(false, inboundRemark, email)
		},
	}
	disableCmd.Flags().StringVar(&inboundRemark, "inbound", "", "inbound remark (required)")
	disableCmd.Flags().StringVar(&email, "email", "", "client email (required)")
	_ = disableCmd.MarkFlagRequired("inbound")
	_ = disableCmd.MarkFlagRequired("email")

	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Get client traffic stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(loadConfigPath())
			if err != nil {
				return err
			}
			node := service.New(cfg)
			stats, err := node.ClientStats(inboundRemark, email)
			if err != nil {
				return err
			}
			return printJSON(stats)
		},
	}
	statsCmd.Flags().StringVar(&inboundRemark, "inbound", "", "inbound remark (required)")
	statsCmd.Flags().StringVar(&email, "email", "", "client email (required)")
	_ = statsCmd.MarkFlagRequired("inbound")
	_ = statsCmd.MarkFlagRequired("email")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List clients on inbound",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(loadConfigPath())
			if err != nil {
				return err
			}
			node := service.New(cfg)
			clients, err := node.ListClients(inboundRemark)
			if err != nil {
				return err
			}
			return printJSON(clients)
		},
	}
	listCmd.Flags().StringVar(&inboundRemark, "inbound", "", "inbound remark (required)")
	_ = listCmd.MarkFlagRequired("inbound")

	cmd.AddCommand(addCmd, enableCmd, disableCmd, statsCmd, listCmd)
	return cmd
}

func setClientEnabled(enabled bool, inboundRemark, email string) error {
	cfg, err := config.Load(loadConfigPath())
	if err != nil {
		return err
	}
	node := service.New(cfg)
	if err := node.SetClientEnabled(inboundRemark, email, enabled); err != nil {
		return err
	}
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	printErr("client " + email + " " + state)
	return nil
}
