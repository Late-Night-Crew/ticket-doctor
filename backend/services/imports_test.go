package services

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"ticketdoctor/db/dbtest"
	"ticketdoctor/engine"
)

var both = ImportOptions{Leagues: []string{"epl"}, Seasons: []string{"2526", "2627"}}

func newImporter(pool *pgxpool.Pool, dir string) *ImportService {
	return NewImportService(pool, DirSource{Dir: dir}, engine.Default(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestImportBuildsMatchesTeamsAndProfiles(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()

	sum, err := newImporter(pool, "testdata/import").Run(ctx, both)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	li := sum.Leagues[0]
	if li.Rows != 18 || li.Inserted != 18 || li.Updated != 0 || li.Unchanged != 0 {
		t.Fatalf("rows/inserted/updated/unchanged = %d/%d/%d/%d, want 18/18/0/0", li.Rows, li.Inserted, li.Updated, li.Unchanged)
	}
	if li.SkippedRows != 1 || !strings.Contains(li.SkippedSample[0], "2627 line 6") {
		t.Fatalf("skipped = %d %v, want the bad score on 2627 line 6", li.SkippedRows, li.SkippedSample)
	}
	wantTeams := []string{"Arsenal", "Chelsea", "Everton", "Leeds", "Nott'm Forest"}
	got := slices.Clone(li.NewTeams)
	slices.Sort(got)
	if !slices.Equal(got, wantTeams) {
		t.Fatalf("new teams = %v, want %v", got, wantTeams)
	}
	if li.Profiles != 10 {
		t.Fatalf("profiles = %d, want 10 (5 teams × home and away)", li.Profiles)
	}
	// "Limited data" means fewer than 5 matches at home OR away. This small league
	// gives every team at most 4 on one side: Arsenal 5 home / 4 away, Chelsea 4 / 5,
	// Everton 5 / 4, Leeds (relegated) 3 / 3, Nott'm Forest (promoted) 1 / 2.
	if !slices.Equal(li.LowConfidence, wantTeams) {
		t.Fatalf("low confidence = %v, want all five", li.LowConfidence)
	}
	var forestHome, forestAway int
	if err := pool.QueryRow(ctx, `
		SELECT max(sample_size) FILTER (WHERE venue = 'home'), max(sample_size) FILTER (WHERE venue = 'away')
		FROM team_profiles p JOIN teams t ON t.id = p.team_id WHERE t.name = 'Nott''m Forest'`).Scan(&forestHome, &forestAway); err != nil {
		t.Fatal(err)
	}
	if forestHome != 1 || forestAway != 2 {
		t.Fatalf("Nott'm Forest samples = %d home / %d away, want 1 / 2", forestHome, forestAway)
	}

	// Arsenal at home: 2-0, 1-1, 4-0 (2025/26) and 3-1, 1-2 (2026/27).
	var p struct {
		sample                                       int
		win, draw, loss, gf, ga, scored, clean, btts float64
		o15, o25, u35                                float64
		form                                         string
	}
	err = pool.QueryRow(ctx, `
		SELECT p.sample_size, p.win_rate, p.draw_rate, p.loss_rate, p.gf_avg, p.ga_avg, p.scored_rate,
		       p.clean_sheet_rate, p.btts_rate, p.over15_rate, p.over25_rate, p.under35_rate, t.form
		FROM team_profiles p JOIN teams t ON t.id = p.team_id
		WHERE t.name = 'Arsenal' AND p.venue = 'home' AND p.window_n = 10`).Scan(
		&p.sample, &p.win, &p.draw, &p.loss, &p.gf, &p.ga, &p.scored, &p.clean, &p.btts, &p.o15, &p.o25, &p.u35, &p.form)
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name      string
		got, want float64
	}{
		{"win", p.win, 0.6}, {"draw", p.draw, 0.2}, {"loss", p.loss, 0.2},
		{"goals for", p.gf, 2.2}, {"goals against", p.ga, 0.8},
		{"scored", p.scored, 1}, {"clean sheets", p.clean, 0.4}, {"btts", p.btts, 0.6},
		{"over 1.5", p.o15, 1}, {"over 2.5", p.o25, 0.6}, {"under 3.5", p.u35, 0.6},
	}
	if p.sample != 5 {
		t.Errorf("sample = %d, want 5", p.sample)
	}
	for _, c := range checks {
		if math.Abs(c.got-c.want) > 1e-9 {
			t.Errorf("Arsenal home %s = %v, want %v", c.name, c.got, c.want)
		}
	}
	// All venues, last five, oldest first: 13/12 W, 20/12 L, 15/08 W, 22/08 W, 29/08 L.
	if p.form != "W-L-W-W-L" {
		t.Errorf("Arsenal form = %q, want W-L-W-W-L", p.form)
	}

	// Kick-off stored in UTC: 15:00 London in August = 14:00 UTC; odds kept to 2 dp.
	var kickoff string
	var oddsHome float64
	if err := pool.QueryRow(ctx, `
		SELECT to_char(m.kickoff AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), m.odds_home
		FROM matches m JOIN teams h ON h.id = m.home_team_id
		WHERE h.name = 'Arsenal' AND m.season = '2526' AND m.kickoff < '2025-08-17'`).Scan(&kickoff, &oddsHome); err != nil {
		t.Fatal(err)
	}
	if kickoff != "2025-08-16 14:00" || oddsHome != 1.8 {
		t.Errorf("kickoff %s odds %v, want 2025-08-16 14:00 and 1.8", kickoff, oddsHome)
	}

	// The run is recorded with its summary.
	var status string
	var summary []byte
	if err := pool.QueryRow(ctx, `SELECT status, summary FROM import_runs WHERE id = $1`, sum.RunID).Scan(&status, &summary); err != nil {
		t.Fatal(err)
	}
	var stored ImportSummary
	if status != "succeeded" || json.Unmarshal(summary, &stored) != nil || stored.Leagues[0].Inserted != 18 {
		t.Fatalf("import_runs row: status %s summary %s", status, summary)
	}
}

func TestImportIsIdempotentAndPicksUpCorrections(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	if _, err := newImporter(pool, "testdata/import").Run(ctx, both); err != nil {
		t.Fatal(err)
	}

	// Running again changes nothing.
	sum, err := newImporter(pool, "testdata/import").Run(ctx, both)
	if err != nil {
		t.Fatal(err)
	}
	if li := sum.Leagues[0]; li.Inserted != 0 || li.Updated != 0 || li.Unchanged != 18 || len(li.NewTeams) != 0 {
		t.Fatalf("second run: %+v, want everything unchanged", li)
	}

	// A corrected score in the source updates that one match and the profiles.
	dir := t.TempDir()
	copyTree(t, "testdata/import", dir)
	path := filepath.Join(dir, "2627", "E0.csv")
	data, _ := os.ReadFile(path)
	fixed := strings.Replace(string(data), "Arsenal,Chelsea,1,2,A", "Arsenal,Chelsea,3,2,H", 1)
	if err := os.WriteFile(path, []byte(fixed), 0o600); err != nil {
		t.Fatal(err)
	}
	sum, err = newImporter(pool, dir).Run(ctx, both)
	if err != nil {
		t.Fatal(err)
	}
	if li := sum.Leagues[0]; li.Updated != 1 || li.Unchanged != 17 || li.Inserted != 0 {
		t.Fatalf("after correction: %+v, want 1 updated", li)
	}
	var win float64
	if err := pool.QueryRow(ctx, `SELECT p.win_rate FROM team_profiles p JOIN teams t ON t.id = p.team_id
		WHERE t.name = 'Arsenal' AND p.venue = 'home'`).Scan(&win); err != nil {
		t.Fatal(err)
	}
	if math.Abs(win-0.8) > 1e-9 {
		t.Fatalf("Arsenal home win rate = %v after the correction, want 0.8", win)
	}
}

func TestImportFailureIsRecordedAndOtherLeaguesStillImport(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()

	// testdata has no La Liga files, so laliga fails while epl imports.
	sum, err := newImporter(pool, "testdata/import").Run(ctx, ImportOptions{Leagues: []string{"epl", "laliga"}, Seasons: both.Seasons})
	if err == nil || !strings.Contains(err.Error(), "laliga") {
		t.Fatalf("err = %v, want a laliga failure", err)
	}
	byLeague := map[string]LeagueImport{}
	for _, li := range sum.Leagues {
		byLeague[li.League] = li
	}
	if byLeague["epl"].Inserted != 18 || byLeague["epl"].Error != "" {
		t.Fatalf("epl should still import: %+v", byLeague["epl"])
	}
	if byLeague["laliga"].Error == "" {
		t.Fatal("laliga should report its error")
	}
	var status string
	var errText *string
	if err := pool.QueryRow(ctx, `SELECT status, error FROM import_runs WHERE id = $1`, sum.RunID).Scan(&status, &errText); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || errText == nil || !strings.Contains(*errText, "laliga") {
		t.Fatalf("run recorded as %s / %v", status, errText)
	}
}

func TestImportRejectsBadOptions(t *testing.T) {
	pool := dbtest.New(t)
	imp := newImporter(pool, "testdata/import")
	for name, opts := range map[string]ImportOptions{
		"no seasons":     {Leagues: []string{"epl"}},
		"bad season":     {Seasons: []string{"2026"}},
		"unknown league": {Leagues: []string{"seriea"}, Seasons: []string{"2627"}},
	} {
		if _, err := imp.Run(context.Background(), opts); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}
