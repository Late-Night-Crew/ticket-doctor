package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"ticketdoctor/db"
	"ticketdoctor/engine"
	"ticketdoctor/logging/loggedmodule"
	"ticketdoctor/pkg/footballdata"
	"ticketdoctor/services"
)

var importFlags struct {
	leagues []string
	seasons []string
	fromDir string
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import results from football-data.co.uk and rebuild team profiles",
	Long: `Downloads each league's results for the current and previous season,
saves new or changed matches, and rebuilds every team's home and away profile.
Safe to run again: unchanged matches are left as they are.

This is the daily job (run by GitHub Actions, not by the server).`,
	Example: `  ticketdoctor import
  ticketdoctor import --league epl --season 2627
  ticketdoctor import --from-dir ./downloads     # reads ./downloads/<season>/<division>.csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
		defer cancel()

		seasons := importFlags.seasons
		if len(seasons) == 0 {
			// Current and previous season, so early-season profiles still have 10 matches.
			current := footballdata.SeasonOf(time.Now())
			previous, _ := footballdata.PreviousSeason(current)
			seasons = []string{previous, current}
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
		var source services.ResultsSource = footballdata.NewClient("")
		if importFlags.fromDir != "" {
			source = services.DirSource{Dir: importFlags.fromDir}
		}

		logger.Info("import starting", loggedmodule.Attr(loggedmodule.Import),
			"database", cfg.Database.Describe(), "seasons", seasons, "leagues", importFlags.leagues)
		summary, err := services.NewImportService(pool, source, eng, logger).Run(ctx, services.ImportOptions{
			Leagues: importFlags.leagues,
			Seasons: seasons,
		})
		if summary != nil {
			logger.Info("import finished", loggedmodule.Attr(loggedmodule.Import), "run_id", summary.RunID, "ok", err == nil)
		}
		return err
	},
}

func init() {
	importCmd.Flags().StringSliceVar(&importFlags.leagues, "league", nil, "league id(s) to import, e.g. epl,laliga (default: all)")
	importCmd.Flags().StringSliceVar(&importFlags.seasons, "season", nil, "season code(s), e.g. 2526,2627 (default: previous and current)")
	importCmd.Flags().StringVar(&importFlags.fromDir, "from-dir", "", "read <dir>/<season>/<division>.csv instead of downloading")
	rootCmd.AddCommand(importCmd)
}
