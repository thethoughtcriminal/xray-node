package cli

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/thethoughtcriminal/xray-node/internal/config"
	"github.com/thethoughtcriminal/xray-node/internal/masterclient"
)

func newJoinCmd() *cobra.Command {
	var (
		masterURL  string
		token      string
		name       string
		publicHost string
		masterIP   string
		skipOpen   bool
	)
	cmd := &cobra.Command{
		Use:   "join",
		Short: "Register this node with xray-master (self-enrollment)",
		Long: `Connects an already installed xray-node to a master server.
Obtain a token on the master: xray-master node token create --name NODE_NAME`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if masterURL == "" || token == "" || name == "" {
				return fmt.Errorf("--master-url, --token, and --name are required")
			}
			cfgPath := loadConfigPath()
			if cfgPath == "" {
				cfgPath = config.DefaultPath
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			if cfg.API.Key == "" {
				return fmt.Errorf("api.key is empty in %s", cfgPath)
			}

			if publicHost == "" {
				publicHost, err = detectPublicHost()
				if err != nil {
					return fmt.Errorf("set --public-host: %w", err)
				}
			}

			port := portFromListen(cfg.API.Listen)
			if !skipOpen {
				if err := openAPIForMaster(cfgPath, cfg, port, masterIP); err != nil {
					return err
				}
				if err := waitLocalAPI(port); err != nil {
					return err
				}
				cfg, _ = config.Load(cfgPath)
			}

			apiURL := fmt.Sprintf("http://%s", net.JoinHostPort(publicHost, port))
			ip := publicHost
			if host, _, err := net.SplitHostPort(cfg.API.Listen); err == nil && host != "" && host != "0.0.0.0" && host != "127.0.0.1" {
				ip = host
			}

			client := masterclient.New(masterURL)
			resp, err := client.Enroll(masterclient.EnrollRequest{
				Token:      token,
				Name:       name,
				APIURL:     apiURL,
				APIKey:     cfg.API.Key,
				PublicHost: publicHost,
				IP:         ip,
			})
			if err != nil {
				return err
			}
			fmt.Printf("enrolled with master as %q (%s) status=%s\n", resp.Name, resp.ID, resp.Status)
			fmt.Println("On master: xray-master sync users")
			return nil
		},
	}
	cmd.Flags().StringVar(&masterURL, "master-url", "", "xray-master public URL (server.public_url)")
	cmd.Flags().StringVar(&token, "token", "", "one-time enroll token from master")
	cmd.Flags().StringVar(&name, "name", "", "node name (must match token)")
	cmd.Flags().StringVar(&publicHost, "public-host", "", "hostname/IP in client VPN links (default: auto-detect)")
	cmd.Flags().StringVar(&masterIP, "master-ip", "", "master public IP for ufw (optional)")
	cmd.Flags().BoolVar(&skipOpen, "skip-open-api", false, "do not change api.listen or firewall")
	return cmd
}

func openAPIForMaster(cfgPath string, cfg *config.Config, port, masterIP string) error {
	listen := fmt.Sprintf("0.0.0.0:%s", port)
	if err := config.SetAPIListen(cfgPath, listen); err != nil {
		return err
	}
	if masterIP != "" && commandExists("ufw") {
		_ = exec.Command("ufw", "allow", "from", masterIP, "to", "any", "port", port).Run()
	}
	if commandExists("systemctl") {
		if out, err := exec.Command("systemctl", "restart", "xray-node").CombinedOutput(); err != nil {
			return fmt.Errorf("restart xray-node: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func waitLocalAPI(port string) error {
	url := fmt.Sprintf("http://127.0.0.1:%s/healthz", port)
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("xray-node API not ready at %s", url)
		}
		time.Sleep(time.Second)
	}
}

func portFromListen(listen string) string {
	if listen == "" {
		return "9472"
	}
	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "9472"
	}
	return port
}

func detectPublicHost() (string, error) {
	if ip := strings.TrimSpace(os.Getenv("NODE_PUBLIC_HOST")); ip != "" {
		return ip, nil
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(raw))
	if ip == "" {
		return "", fmt.Errorf("empty response from ipify")
	}
	return ip, nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
