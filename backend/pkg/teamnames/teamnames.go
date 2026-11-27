// Package teamnames turns club names into a comparison key, so the same club
// matches across data sources (FR-1.5): "Manchester United FC", "Man Utd" and
// "Man United" all become "man united".
//
// The key is football-data.co.uk's spelling (the primary source), normalised.
// Other spellings map onto it through the alias table below; add a line there
// when a new source spells a club differently.
package teamnames

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Key normalises a club name for matching: accents removed, lower case,
// apostrophes dropped, other punctuation to spaces, club-type words (FC, AFC,
// CF...) and founding years removed, and known alternative names mapped to the
// primary spelling.
func Key(name string) string {
	b := base(name)
	if canonical, ok := aliasKeys[b]; ok {
		return canonical
	}
	return b
}

// base is the normalisation without the alias table.
func base(name string) string {
	s := stripAccents(strings.ToLower(name))
	s = strings.NewReplacer("'", "", "’", "", "&", " and ", ".", " ").Replace(s)

	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}

	var words []string
	for _, w := range strings.Fields(b.String()) {
		if clubWords[w] || isFoundingYear(w) {
			continue
		}
		words = append(words, w)
	}
	return strings.Join(words, " ")
}

// isFoundingYear matches years in names like "TSG 1899 Hoffenheim".
func isFoundingYear(w string) bool {
	return len(w) == 4 && (strings.HasPrefix(w, "18") || strings.HasPrefix(w, "19")) &&
		strings.IndexFunc(w, func(r rune) bool { return !unicode.IsDigit(r) }) == -1
}

// aliasKeys is the alias table with both sides normalised by base, built once.
// Writing aliases in plain form means the table can't drift from the rules.
var aliasKeys = func() map[string]string {
	m := make(map[string]string, len(aliases))
	for other, primary := range aliases {
		m[base(other)] = base(primary)
	}
	return m
}()

// Same reports whether two names refer to the same club.
func Same(a, b string) bool { return Key(a) == Key(b) }

func stripAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return out
}

// clubWords are legal-form and club-type words that sources add or leave out.
var clubWords = map[string]bool{
	"fc": true, "afc": true, "cf": true, "sc": true, "sv": true, "ac": true,
	"cd": true, "ud": true, "sd": true, "rcd": true, "ca": true, "rc": true,
	"tsg": true, "vfb": true, "vfl": true, "fsv": true, "ssv": true, "bsc": true,
	"1": true, "de": true, "the": true,
}

// aliases maps other spellings to football-data.co.uk's spelling. Write both
// sides as ordinary names; they're normalised automatically (see aliasKeys).
var aliases = map[string]string{
	// England
	"manchester united":        "man united",
	"man utd":                  "man united",
	"manchester city":          "man city",
	"nottingham forest":        "nottm forest",
	"tottenham hotspur":        "tottenham",
	"spurs":                    "tottenham",
	"brighton and hove albion": "brighton",
	"brighton hove albion":     "brighton",
	"wolverhampton wanderers":  "wolves",
	"wolverhampton":            "wolves",
	"newcastle united":         "newcastle",
	"west ham united":          "west ham",
	"leeds united":             "leeds",
	"leicester city":           "leicester",
	"ipswich town":             "ipswich",
	"hull city":                "hull",
	"coventry city":            "coventry",
	"luton town":               "luton",
	"sheffield united":         "sheffield united",
	"sheffield utd":            "sheffield united",
	"afc bournemouth":          "bournemouth",
	"norwich city":             "norwich",
	"stoke city":               "stoke",
	"cardiff city":             "cardiff",
	"swansea city":             "swansea",
	"west bromwich albion":     "west brom",
	"queens park rangers":      "qpr",
	"birmingham city":          "birmingham",
	"blackburn rovers":         "blackburn",
	"huddersfield town":        "huddersfield",
	// Spain
	"athletic club":                 "ath bilbao",
	"athletic bilbao":               "ath bilbao",
	"atletico madrid":               "ath madrid",
	"club atletico madrid":          "ath madrid",
	"atletico de madrid":            "ath madrid",
	"real betis":                    "betis",
	"real betis balompie":           "betis",
	"celta vigo":                    "celta",
	"rc celta":                      "celta",
	"celta de vigo":                 "celta",
	"espanyol":                      "espanol",
	"espanyol barcelona":            "espanol",
	"rayo vallecano":                "vallecano",
	"real sociedad":                 "sociedad",
	"real sociedad de futbol":       "sociedad",
	"deportivo la coruna":           "la coruna",
	"deportivo":                     "la coruna",
	"racing santander":              "santander",
	"real racing club de santander": "santander",
	"rayo vallecano de madrid":      "vallecano",
	"racing":                        "santander",
	"deportivo alaves":              "alaves",
	"getafe":                        "getafe",
	"ca osasuna":                    "osasuna",
	"sevilla":                       "sevilla",
	"valencia":                      "valencia",
	"villarreal":                    "villarreal",
	"malaga":                        "malaga",
	"elche":                         "elche",
	"levante ud":                    "levante",
	"real oviedo":                   "oviedo",
	"girona":                        "girona",
	"real mallorca":                 "mallorca",
	"rcd mallorca":                  "mallorca",
	"las palmas":                    "las palmas",
	// Germany
	"bayern":                   "bayern munich",
	"bayern munchen":           "bayern munich",
	"borussia dortmund":        "dortmund",
	"bayer leverkusen":         "leverkusen",
	"bayer 04 leverkusen":      "leverkusen",
	"eintracht frankfurt":      "ein frankfurt",
	"borussia monchengladbach": "mgladbach",
	"monchengladbach":          "mgladbach",
	"koln":                     "fc koln",
	"cologne":                  "fc koln",
	"union berlin":             "union berlin",
	"hoffenheim":               "hoffenheim",
	"stuttgart":                "stuttgart",
	"werder bremen":            "werder bremen",
	"hamburger":                "hamburg",
	"augsburg":                 "augsburg",
	"freiburg":                 "freiburg",
	"mainz 05":                 "mainz",
	"schalke 04":               "schalke 04",
	"schalke":                  "schalke 04",
	"elversberg":               "elversberg",
	"sv 07 elversberg":         "elversberg",
	"paderborn 07":             "paderborn",
	"rb leipzig":               "rb leipzig",
	"rasenballsport leipzig":   "rb leipzig",
	"wolfsburg":                "wolfsburg",
	"heidenheim 1846":          "heidenheim",
	"st pauli":                 "st pauli",
	"holstein kiel":            "holstein kiel",
	"bochum 1848":              "bochum",
	"darmstadt 98":             "darmstadt",
}
