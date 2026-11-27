package services

import (
	"testing"
	"time"
)

func cat(day, hour int) time.Time { return time.Date(2026, 11, day, hour, 0, 0, 0, CAT) }

func TestWindowRangeInCAT(t *testing.T) {
	// 23-29 Nov 2026 is Monday to Sunday.
	cases := []struct {
		name     string
		w        Window
		now      time.Time
		from, to time.Time
	}{
		{"today, midday", Today, cat(27, 12), cat(27, 12), cat(28, 0)},
		{"weekend from Monday", Weekend, cat(23, 9), cat(28, 0), cat(30, 0)},
		{"weekend from Friday", Weekend, cat(27, 23), cat(28, 0), cat(30, 0)},
		{"weekend on Saturday is the rest of it", Weekend, cat(28, 15), cat(28, 15), cat(30, 0)},
		{"weekend on Sunday is the rest of today", Weekend, cat(29, 18), cat(29, 18), cat(30, 0)},
	}
	for _, c := range cases {
		from, to := WindowRange(c.w, c.now)
		if !from.Equal(c.from) || to == nil || !to.Equal(c.to) {
			t.Errorf("%s: got %v to %v, want %v to %v", c.name, from, to, c.from, c.to)
		}
	}
	if from, to := WindowRange(Upcoming, cat(27, 12)); !from.Equal(cat(27, 12)) || to != nil {
		t.Errorf("upcoming: got %v to %v, want from now with no end", from, to)
	}
}

func TestWindowUsesCATNotUTC(t *testing.T) {
	// 23:30 UTC on Friday is already 01:30 on Saturday in CAT: "today" is Saturday.
	now := time.Date(2026, 11, 27, 23, 30, 0, 0, time.UTC)
	_, to := WindowRange(Today, now)
	if want := cat(29, 0); !to.Equal(want) {
		t.Fatalf("today ends %v, want %v (end of Saturday in CAT)", to, want)
	}
	if _, offset := now.In(CAT).Zone(); offset != 2*60*60 {
		t.Fatalf("CAT offset = %d, want +2h", offset)
	}
}

func TestParseWindow(t *testing.T) {
	for in, want := range map[string]Window{"": Upcoming, "today": Today, "weekend": Weekend, "upcoming": Upcoming} {
		if got, err := ParseWindow(in); err != nil || got != want {
			t.Errorf("%q: %v %v", in, got, err)
		}
	}
	if _, err := ParseWindow("tomorrow"); err == nil {
		t.Error("tomorrow should be rejected")
	}
}
