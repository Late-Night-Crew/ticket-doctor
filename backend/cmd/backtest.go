package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"ticketdoctor/db"
	"ticketdoctor/engine"
	"ticketdoctor/services"
)

var backtestFlags struct {
	league, market string
	low            bool
}

// backtest replays every finished match in the database, rating each from the
// games before it, and prints how the estimates compare with what happened.
var backtestCmd = &cobra.Command{
	Use:   "backtest",
	Short: "Check the estimates against past results (calibration)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		conn := cfg.Database.ConnString()
		if conn == "" {
			return errNoDatabase
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
		defer cancel()
		pool, err := db.OpenPool(ctx, conn, 2)
		if err != nil {
			return err
		}
		defer pool.Close()
		eng, err := engine.New(cfg.Rules.Thresholds())
		if err != nil {
			return err
		}
		res, err := services.NewCalibrationService(pool, eng).Calibration(ctx, services.CalibrationRequest{
			Source: services.SourceBacktest, League: backtestFlags.league, Market: backtestFlags.market,
			IncludeLowConfidence: backtestFlags.low,
		})
		if err != nil {
			return err
		}
		printCalibration(res)
		return nil
	},
}

func printCalibration(r *services.Calibration) {
	pct := func(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }
	fmt.Printf("%d games rated (%d left out for thin profiles), %d estimates, Brier %.4f\n\n",
		r.Matches, r.SkippedLowConfidence, r.N, r.Brier)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "ESTIMATE\tN\tPREDICTED\tHAPPENED\tGAP\t")
	for _, b := range r.Bins {
		if b.N == 0 {
			continue
		}
		fmt.Fprintf(w, "%.0f–%.0f%%\t%d\t%s\t%s\t%+.1f\t\n", b.From*100, b.To*100, b.N, pct(b.Predicted), pct(b.Actual), (b.Actual-b.Predicted)*100)
	}
	_ = w.Flush()

	fmt.Println()
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "BAND\tN\tPREDICTED\tHAPPENED\t")
	for _, b := range r.Bands {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t\n", b.Band, b.N, pct(b.Predicted), pct(b.Actual))
	}
	_ = w.Flush()

	fmt.Println()
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "MARKET\tN\tPREDICTED\tHAPPENED\tBRIER\tBASELINE\tSKILL\t")
	for _, m := range r.Markets {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%.4f\t%.4f\t%+.3f\t\n", m.Market, m.N, pct(m.Predicted), pct(m.Actual), m.Brier, m.BaseBrier, m.Skill)
	}
	_ = w.Flush()
	fmt.Println("\nSKILL above 0 beats always predicting the market's average rate; 0 or below means the stats add nothing.")
}

func init() {
	backtestCmd.Flags().StringVar(&backtestFlags.league, "league", "", "only this league (epl, laliga, bundesliga)")
	backtestCmd.Flags().StringVar(&backtestFlags.market, "market", "", "only this market, e.g. OVER_2_5")
	backtestCmd.Flags().BoolVar(&backtestFlags.low, "include-low-confidence", false, "also rate games where a team has too few past games")
	rootCmd.AddCommand(backtestCmd)
}
