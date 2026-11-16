package scraper

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "time-off.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 38 {
		t.Fatalf("got %d holidays, want 38", len(got))
	}

	want := []struct {
		name     string
		date     time.Time
		observed bool
	}{
		{"New Year's Day", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"Juneteenth", time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC), true},
		{"Free Day", time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC), false},
		{"Free Day", time.Date(2027, 3, 25, 0, 0, 0, 0, time.UTC), false},
		{"Independence Day", time.Date(2027, 7, 5, 0, 0, 0, 0, time.UTC), true},
		{"Dr. Martin Luther King, Jr. Day", time.Date(2027, 1, 18, 0, 0, 0, 0, time.UTC), false},
	}
	for _, w := range want {
		found := false
		for _, h := range got {
			if h.Name == w.name && h.Date.Equal(w.date) && h.Observed == w.observed {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing holiday %s %s observed=%v", w.name, w.date.Format("2006-01-02"), w.observed)
		}
	}
}

func TestParseEntryRejectsGarbage(t *testing.T) {
	bad := []string{
		"Start with Veer: Use Veer to plan your leave",
		"Memorial Day—Somewhere 45, 2026",
		"Memorial Day—May 32, 2026",
		"Memorial Day—Feb 30, 2026",
		"",
	}
	for _, line := range bad {
		if _, ok := parseEntry(line, 2026); ok {
			t.Errorf("parseEntry(%q) accepted garbage", line)
		}
	}
}

func TestParseNoHolidays(t *testing.T) {
	if _, err := Parse([]byte("<html><body><p>nothing here</p></body></html>")); err == nil {
		t.Fatal("Parse() returned nil error for empty page, want ErrNoHolidays")
	}
}
