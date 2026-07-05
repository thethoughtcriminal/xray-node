package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var (
		assumeYes bool
		keep3xUI  bool
	)

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove xray-node and 3x-ui installed by install.sh",
		Long: `Removes the xray-node systemd service, binary, config, and install directory.
By default also uninstalls 3x-ui and Xray. Requires root.

Equivalent to: sudo ./scripts/uninstall.sh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("uninstall must be run as root (sudo xray-node uninstall)")
			}

			script, err := findUninstallScript()
			if err != nil {
				return err
			}

			runArgs := []string{script}
			if assumeYes {
				runArgs = append(runArgs, "--yes")
			}
			if keep3xUI {
				runArgs = append(runArgs, "--keep-3xui")
			}

			c := exec.Command("bash", runArgs...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}

	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "Skip confirmation")
	cmd.Flags().BoolVar(&keep3xUI, "keep-3xui", false, "Keep 3x-ui installed")
	return cmd
}

func findUninstallScript() (string, error) {
	candidates := []string{
		"/opt/xray-node/scripts/uninstall.sh",
		"scripts/uninstall.sh",
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "scripts", "uninstall.sh"),
			filepath.Join(exeDir, "..", "scripts", "uninstall.sh"),
		)
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "scripts", "uninstall.sh"))
	}

	for _, path := range candidates {
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		return abs, nil
	}

	return "", fmt.Errorf("uninstall script not found; run: curl -fsSL https://raw.githubusercontent.com/thethoughtcriminal/xray-node/main/scripts/uninstall.sh | sudo bash")
}
