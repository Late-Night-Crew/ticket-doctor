package teamnames

import "testing"

// Every club in the 2026/27 football-data.co.uk files, keyed as stored in teams.name_key.
var primary = []string{
	// Premier League (E0)
	"Arsenal", "Aston Villa", "Bournemouth", "Brentford", "Brighton", "Chelsea", "Coventry",
	"Crystal Palace", "Everton", "Fulham", "Hull", "Ipswich", "Leeds", "Liverpool", "Man City",
	"Man United", "Newcastle", "Nott'm Forest", "Sunderland", "Tottenham",
	// La Liga (SP1)
	"Alaves", "Ath Bilbao", "Ath Madrid", "Barcelona", "Betis", "Celta", "Elche", "Espanol",
	"Getafe", "La Coruna", "Levante", "Malaga", "Osasuna", "Real Madrid", "Santander", "Sevilla",
	"Sociedad", "Valencia", "Vallecano", "Villarreal",
	// Bundesliga (D1)
	"Augsburg", "Bayern Munich", "Dortmund", "Ein Frankfurt", "Elversberg", "FC Koln", "Freiburg",
	"Hamburg", "Hoffenheim", "Leverkusen", "M'gladbach", "Mainz", "Paderborn", "RB Leipzig",
	"Schalke 04", "Stuttgart", "Union Berlin", "Werder Bremen",
}

// Official long names, as other sources (e.g. football-data.org) spell them.
var official = map[string]string{
	"Manchester United FC":          "Man United",
	"Man Utd":                       "Man United",
	"Manchester City FC":            "Man City",
	"Nottingham Forest FC":          "Nott'm Forest",
	"Tottenham Hotspur FC":          "Tottenham",
	"Brighton & Hove Albion FC":     "Brighton",
	"AFC Bournemouth":               "Bournemouth",
	"Newcastle United FC":           "Newcastle",
	"Leeds United FC":               "Leeds",
	"Ipswich Town FC":               "Ipswich",
	"Hull City AFC":                 "Hull",
	"Coventry City FC":              "Coventry",
	"Athletic Club":                 "Ath Bilbao",
	"Club Atlético de Madrid":       "Ath Madrid",
	"Real Betis Balompié":           "Betis",
	"RC Celta de Vigo":              "Celta",
	"RCD Espanyol de Barcelona":     "Espanol",
	"Rayo Vallecano de Madrid":      "Vallecano",
	"Real Sociedad de Fútbol":       "Sociedad",
	"RC Deportivo La Coruña":        "La Coruna",
	"Real Racing Club de Santander": "Santander",
	"Deportivo Alavés":              "Alaves",
	"Levante UD":                    "Levante",
	"FC Bayern München":             "Bayern Munich",
	"Borussia Dortmund":             "Dortmund",
	"Bayer 04 Leverkusen":           "Leverkusen",
	"Eintracht Frankfurt":           "Ein Frankfurt",
	"Borussia Mönchengladbach":      "M'gladbach",
	"1. FC Köln":                    "FC Koln",
	"1. FC Union Berlin":            "Union Berlin",
	"TSG 1899 Hoffenheim":           "Hoffenheim",
	"VfB Stuttgart":                 "Stuttgart",
	"SV Werder Bremen":              "Werder Bremen",
	"Hamburger SV":                  "Hamburg",
	"FC Augsburg":                   "Augsburg",
	"SC Freiburg":                   "Freiburg",
	"1. FSV Mainz 05":               "Mainz",
	"FC Schalke 04":                 "Schalke 04",
	"SV 07 Elversberg":              "Elversberg",
	"SC Paderborn 07":               "Paderborn",
	"RB Leipzig":                    "RB Leipzig",
}

func TestPrimaryNamesAreStableAndDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, name := range primary {
		k := Key(name)
		if k == "" {
			t.Errorf("Key(%q) is empty", name)
		}
		if Key(k) != k {
			t.Errorf("Key is not stable for %q: %q then %q", name, k, Key(k))
		}
		if other, dup := seen[k]; dup {
			t.Errorf("%q and %q share the key %q", name, other, k)
		}
		seen[k] = name
	}
}

func TestOfficialNamesMatchPrimary(t *testing.T) {
	for other, want := range official {
		if !Same(other, want) {
			t.Errorf("%q → %q, but %q → %q", other, Key(other), want, Key(want))
		}
	}
}

func TestNormalisation(t *testing.T) {
	for in, want := range map[string]string{
		"Nott'm Forest":   "nottm forest",
		"M'gladbach":      "mgladbach",
		"  Man   City ":   "man city",
		"Atlético Madrid": "ath madrid",
		"1. FC Köln":      "koln", // same key as football-data.co.uk's "FC Koln"
		"Some New Club":   "some new club",
	} {
		if got := Key(in); got != want {
			t.Errorf("Key(%q) = %q, want %q", in, got, want)
		}
	}
	if Same("Arsenal", "Chelsea") {
		t.Error("different clubs must not match")
	}
}
