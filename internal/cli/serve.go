package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
			if err := cfg.Validate(); err != nil {
				return err
			}
			server := api.New(cfg)
			fmt.Printf("xray-node API listening on http://%s\n", cfg.API.Listen)

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			errCh := make(chan error, 1)
			go func() {
				errCh <- server.ListenAndServe()
			}()

			select {
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_ = server.Shutdown(shutdownCtx)
				return nil
			case err := <-errCh:
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					return err
				}
				return nil
			}
		},
	}
}
