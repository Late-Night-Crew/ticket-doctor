package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"ticketdoctor/api"
	"ticketdoctor/db"
	"ticketdoctor/engine"
	"ticketdoctor/logging/loggedmodule"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the HTTP API",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Stop cleanly on Ctrl+C or SIGTERM (what Render sends on redeploy).
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		conn := cfg.Database.ConnString()
		if conn == "" {
			return errNoDatabase
		}
		pool, err := db.OpenPool(ctx, conn, cfg.Database.MaxConns)
		if err != nil {
			return err
		}
		defer pool.Close()
		eng, err := engine.New(cfg.Rules.Thresholds())
		if err != nil {
			return err
		}

		logger.Info("starting API",
			loggedmodule.Attr(loggedmodule.Server),
			"address", cfg.ListenAddress(),
			"environment", cfg.Environment,
			"database", cfg.Database.Describe(),
			"version", version,
		)
		if err := api.New(cfg, logger, version, api.Deps{Pool: pool, Engine: eng}).Start(ctx); err != nil {
			return err
		}
		logger.Info("API stopped", loggedmodule.Attr(loggedmodule.Server))
		return nil
	},
}

func init() { rootCmd.AddCommand(serverCmd) }
