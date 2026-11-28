package cmd

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"ticketdoctor/db"
	"ticketdoctor/engine"
	"ticketdoctor/logging/loggedmodule"
	"ticketdoctor/pkg/footballdata"
	"ticketdoctor/pkg/footballdataorg"
	"ticketdoctor/services"
)

var scoresCmd = &cobra.Command{
	Use:   "scores",
	Short: "Save full-time scores for games just played, and settle saved tickets",
	Long: `A quick check for match days (addendum A10). Finds games that kicked off
between 36 hours and 100 minutes ago with no result yet, asks football-data.org
for their full-time scores (one request per league), saves them, rebuilds those
leagues' team profiles and settles saved tickets.

With no game awaiting a result it makes no request and exits at once, so it is
cheap to run often: GitHub Actions runs it every 15 minutes on match days.
The full import (ticketdoctor import) still runs twice a day, and the results
file replaces these scores when it arrives.

Needs a football-data.org API key (TD_FOOTBALL_DATA_ORG_TOKEN).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
		defer cancel()

		token := cfg.FootballDataOrg.Token
		if token == "" {
			return errors.New("no football-data.org API key: set TD_FOOTBALL_DATA_ORG_TOKEN")
		}
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

		svc := services.NewImportService(pool, footballdata.NewClient(""), eng, logger).
			WithFixtures(footballdataorg.NewClient(token))
		summary, err := svc.QuickScores(ctx, time.Now())
		if summary == nil {
			return err
		}
		if summary.RunID == 0 {
			logger.Info("no games awaiting a result", loggedmodule.Attr(loggedmodule.Import))
			return err
		}
		logger.Info("score check finished", loggedmodule.Attr(loggedmodule.Import),
			"run_id", summary.RunID, "results", summary.Results(), "ok", err == nil)

		var cerr error
		if summary.Results() > 0 {
			settled, _, e := services.NewSavedService(pool, time.Now).Cleanup(ctx)
			if cerr = e; cerr != nil {
				logger.Error("saved ticket settling failed", loggedmodule.Attr(loggedmodule.Import), "error", cerr)
			} else {
				logger.Info("saved tickets settled", loggedmodule.Attr(loggedmodule.Import), "settled", settled)
			}
		}
		return errors.Join(err, cerr)
	},
}

func init() {
	rootCmd.AddCommand(scoresCmd)
}
