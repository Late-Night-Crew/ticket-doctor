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
	priors         []float64
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
		svc := services.NewCalibrationService(pool, eng)
		run := func(prior *float64) (*services.Calibration, error) {
			return svc.Calibration(ctx, services.CalibrationRequest{
				Source: services.SourceBacktest, League: backtestFlags.league, Market: backtestFlags.market,
				IncludeLowConfidence: backtestFlags.low, Prior: prior,
			})
		}
		if len(backtestFlags.priors) <= 1 {
			var prior *float64 // the rules in use now
			if len(backtestFlags.priors) == 1 {
				prior = &backtestFlags.priors[0]
			}
			res, err := run(prior)
			if err != nil {
				return err
			}
			printCalibration(res)
			return nil
		}
		return comparePriors(backtestFlags.priors, run)
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

// comparePriors runs the backtest once per prior and prints one line each,
// to choose how strongly to pull team rates towards the league average.
func comparePriors(priors []float64, run func(*float64) (*services.Calibration, error)) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "PRIOR\tBRIER\tSAID 90%+\tHAPPENED\tSAID <10%\tHAPPENED\tLOW SAID\tLOW HAPPENED\tWIN SKILL\tGOALS SKILL\tMARKETS ≤0\t")
	for _, p := range priors {
		r, err := run(&p)
		if err != nil {
			return err
		}
		top, bottom := r.Bins[len(r.Bins)-1], r.Bins[0]
		var low services.BandStat
		for _, b := range r.Bands {
			if b.Band == "low" {
				low = b
			}
		}
		var win, goals float64
		var nWin, nGoals, losing int
		for _, m := range r.Markets {
			if engine.Market(m.Market).Group() == engine.GroupWins {
				win += m.Skill
				nWin++
			} else {
				goals += m.Skill
				nGoals++
			}
			if m.Skill <= 0 {
				losing++
			}
		}
		pct := func(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }
		fmt.Fprintf(w, "%g\t%.4f\t%s (%d)\t%s\t%s (%d)\t%s\t%s\t%s\t%+.3f\t%+.3f\t%d/%d\t\n", p, r.Brier,
			pct(top.Predicted), top.N, pct(top.Actual), pct(bottom.Predicted), bottom.N, pct(bottom.Actual),
			pct(low.Predicted), pct(low.Actual), win/float64(max(1, nWin)), goals/float64(max(1, nGoals)), losing, len(r.Markets))
	}
	_ = w.Flush()
	fmt.Println("\nWIN/GOALS SKILL are averages over those markets; above 0 beats guessing the league average.")
	return nil
}

func init() {
	backtestCmd.Flags().StringVar(&backtestFlags.league, "league", "", "only this league (epl, laliga, bundesliga)")
	backtestCmd.Flags().StringVar(&backtestFlags.market, "market", "", "only this market, e.g. OVER_2_5")
	backtestCmd.Flags().BoolVar(&backtestFlags.low, "include-low-confidence", false, "also rate games where a team has too few past games")
	backtestCmd.Flags().Float64SliceVar(&backtestFlags.priors, "prior", nil, "override the prior_games rule (pull team rates towards the league average by this many games); several, e.g. 0,5,10,20, compares them")
	rootCmd.AddCommand(backtestCmd)
}
