package footballdata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func open(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestParseCurrentFormat(t *testing.T) {
	matches, skipped, err := Parse(open(t, "E0_2627_sample.csv"))
	if err != nil {
		t.Fatal(err)
	}
	// 5 rows: one has an unreadable score; two blank rows are ignored.
	if len(matches) != 4 {
		t.Fatalf("got %d matches, want 4", len(matches))
	}
	if len(skipped) != 1 || skipped[0].Line != 4 || !strings.Contains(skipped[0].Reason, "unreadable score") {
		t.Fatalf("skipped = %+v, want line 4 unreadable score", skipped)
	}

	first := matches[0]
	if first.Division != "E0" || first.HomeTeam != "Arsenal" || first.AwayTeam != "Coventry" ||
		first.HomeGoals != 3 || first.AwayGoals != 0 || first.Line != 2 {
		t.Fatalf("first = %+v", first)
	}
	// Average odds are preferred over Bet365.
	if *first.OddsHome != 1.19 || *first.OddsDraw != 6.77 || *first.OddsAway != 14.19 {
		t.Fatalf("odds = %v %v %v, want the Avg columns", *first.OddsHome, *first.OddsDraw, *first.OddsAway)
	}
	// 20:00 London in August is BST (UTC+1).
	if want := time.Date(2026, 8, 21, 19, 0, 0, 0, time.UTC); !first.Kickoff.Equal(want) {
		t.Fatalf("kickoff = %v, want %v", first.Kickoff.UTC(), want)
	}
	// Falls back to Bet365 odds when the averages are empty; keeps apostrophes in names.
	if m := matches[1]; m.HomeTeam != "Nott'm Forest" || m.OddsHome == nil || *m.OddsHome != 3.1 {
		t.Fatalf("second = %+v", m)
	}
	// UK clocks go forward on 28 March 2027, so 15:00 on 29 March is BST (UTC+1).
	if want := time.Date(2027, 3, 29, 14, 0, 0, 0, time.UTC); !matches[3].Kickoff.Equal(want) {
		t.Fatalf("spring kickoff = %v, want %v", matches[3].Kickoff.UTC(), want)
	}
}

func TestParseOldFormat(t *testing.T) {
	matches, skipped, err := Parse(open(t, "E0_0304_old.csv"))
	if err != nil || len(skipped) != 0 || len(matches) != 2 {
		t.Fatalf("matches=%d skipped=%v err=%v", len(matches), skipped, err)
	}
	m := matches[0]
	// Two-digit year, no Time column (15:00 assumed, BST in August), BbAv odds.
	if want := time.Date(2003, 8, 16, 14, 0, 0, 0, time.UTC); !m.Kickoff.Equal(want) {
		t.Fatalf("kickoff = %v, want %v", m.Kickoff.UTC(), want)
	}
	if m.OddsHome == nil || *m.OddsHome != 1.36 {
		t.Fatalf("odds = %v", m.OddsHome)
	}
}

func TestParseRejectsOtherFiles(t *testing.T) {
	if _, _, err := Parse(strings.NewReader("Name,Score\nfoo,1\n")); err == nil || !strings.Contains(err.Error(), "missing column") {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := Parse(strings.NewReader("")); err == nil {
		t.Fatal("empty file should be an error")
	}
}

func TestParseRowProblems(t *testing.T) {
	csv := "Div,Date,Time,HomeTeam,AwayTeam,FTHG,FTAG\n" +
		"E0,31/02/2026,15:00,A,B,1,0\n" + // impossible date
		"E0,01/09/2026,15:00,A,A,1,0\n" + // same team twice
		"E0,01/09/2026,15:00,A,,1,0\n" + // missing away team
		"E0,01/09/2026,15:00,A,B,,\n" + // no score
		"E0,01/09/2026,15:00,A,B,-1,0\n" // negative score
	matches, skipped, err := Parse(strings.NewReader(csv))
	if err != nil || len(matches) != 0 || len(skipped) != 5 {
		t.Fatalf("matches=%d skipped=%+v err=%v", len(matches), skipped, err)
	}
}

func TestSeasons(t *testing.T) {
	for _, tt := range []struct {
		date time.Time
		want string
	}{
		{time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC), "2627"},
		{time.Date(2027, 5, 20, 0, 0, 0, 0, time.UTC), "2627"},
		{time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), "2526"},
		{time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), "2627"},
		{time.Date(2099, 9, 1, 0, 0, 0, 0, time.UTC), "9900"},
		{time.Date(2100, 3, 1, 0, 0, 0, 0, time.UTC), "9900"},
	} {
		if got := SeasonOf(tt.date); got != tt.want {
			t.Errorf("SeasonOf(%s) = %s, want %s", tt.date.Format("2006-01-02"), got, tt.want)
		}
	}
	if prev, err := PreviousSeason("2627"); err != nil || prev != "2526" {
		t.Errorf("PreviousSeason(2627) = %s, %v", prev, err)
	}
	if prev, _ := PreviousSeason("0001"); prev != "9900" {
		t.Errorf("PreviousSeason(0001) = %s, want 9900", prev)
	}
	for _, bad := range []string{"26", "2628", "abcd", "26-27"} {
		if ValidSeason(bad) == nil {
			t.Errorf("ValidSeason(%q) should fail", bad)
		}
	}
}

func TestClientDownloadsAndRetries(t *testing.T) {
	sample, _ := os.ReadFile("testdata/E0_2627_sample.csv")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		switch {
		case r.URL.Path == "/mmz4281/9900/E0.csv":
			http.NotFound(w, r)
		case n == 1:
			http.Error(w, "busy", http.StatusServiceUnavailable) // first attempt fails, retry succeeds
		case r.Header.Get("User-Agent") == "":
			http.Error(w, "no user agent", http.StatusForbidden)
		default:
			_, _ = w.Write(sample)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	c.RetryWait = time.Millisecond
	matches, _, err := c.Results(context.Background(), "E0", "2627")
	if err != nil || len(matches) != 4 || calls.Load() != 2 {
		t.Fatalf("matches=%d calls=%d err=%v", len(matches), calls.Load(), err)
	}

	calls.Store(1) // skip the "busy" answer
	if _, _, err := c.Results(context.Background(), "E0", "9900"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing season: err = %v", err)
	}
	if _, _, err := c.Results(context.Background(), "E0", "26"); err == nil {
		t.Fatal("bad season code should fail before any request")
	}
}
