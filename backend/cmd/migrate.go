package cmd

import (
	"context"
	"errors"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"

	"ticketdoctor/db"
	"ticketdoctor/logging/loggedmodule"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply database migrations (embedded in the binary)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Database.URL == "" {
			return errors.New("no database URL: set TD_DATABASE_URL or database.url in config.yaml")
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
		defer cancel()

		conn, err := db.OpenSQL(ctx, cfg.Database.URL)
		if err != nil {
			return err
		}
		defer conn.Close()

		// goose's Provider works on the embedded files directly and keeps no global state.
		provider, err := goose.NewProvider(goose.DialectPostgres, conn, db.Migrations())
		if errors.Is(err, goose.ErrNoMigrations) {
			logger.Info("no migrations yet; nothing to do", loggedmodule.Attr(loggedmodule.Migrate))
			return nil
		}
		if err != nil {
			return err
		}

		results, err := provider.Up(ctx)
		if err != nil {
			return err
		}
		for _, r := range results {
			logger.Info("applied migration", loggedmodule.Attr(loggedmodule.Migrate),
				"file", r.Source.Path, "duration_ms", r.Duration.Milliseconds())
		}
		current, err := provider.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		logger.Info("database up to date", loggedmodule.Attr(loggedmodule.Migrate),
			"version", current, "applied", len(results))
		return nil
	},
}

func init() { rootCmd.AddCommand(migrateCmd) }
