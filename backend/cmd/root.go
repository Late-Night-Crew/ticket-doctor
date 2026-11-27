// Package cmd holds the ticketdoctor CLI (cobra), mirroring nsms and parrot:
//
//	ticketdoctor server    start the HTTP API
//	ticketdoctor migrate   apply the embedded database migrations
//	ticketdoctor import    load results and rebuild team profiles (the daily
//	                       job, run from GitHub Actions rather than inside the
//	                       server; SRS 2.3)
//	ticketdoctor admin     create and manage admin accounts (B6)
package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"ticketdoctor/config"
	"ticketdoctor/logging"
)

// version is set at build time: go build -ldflags "-X ticketdoctor/cmd.version=v0.1.0"
var version = "dev"

var (
	cfgFile string
	cfg     *config.Config
	logger  *slog.Logger
)

var rootCmd = &cobra.Command{
	Use:     "ticketdoctor",
	Short:   "Ticket Doctor backend: API server and data jobs",
	Version: version,
	// Print usage only for wrong flags or arguments, not when a command fails.
	SilenceUsage: true,
	// Runs before every subcommand: load config and set up logging once.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load(cfgFile)
		if err != nil {
			return err
		}
		l, err := logging.New(c.Logging, os.Stderr)
		if err != nil {
			return err
		}
		cfg, logger = c, l
		return nil
	},
}

// Execute runs the CLI.
func Execute() error { return rootCmd.Execute() }

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ./config.yaml if present)")
}
