package ics

import (
	"strings"
	"testing"
	"time"

	"github.com/rajkumaar23/nvidia-holidays/internal/model"
)

func TestGenerateDeterministic(t *testing.T) {
	holidays := []model.Holiday{
		{Name: "Free Day", Date: time.Date(2027, 3, 25, 0, 0, 0, 0, time.UTC)},
		{Name: "New Year's Day", Date: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Name: "Juneteenth", Date: time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC), Observed: true},
	}
	first := Generate(holidays)
	second := Generate(holidays)
	if first != second {
		t.Fatal("Generate is not deterministic")
	}
	if !strings.HasPrefix(first, "BEGIN:VCALENDAR\r\n") || !strings.HasSuffix(first, "END:VCALENDAR\r\n") {
		t.Fatal("missing calendar envelope")
	}
	// Events sorted chronologically.
	if strings.Index(first, "Juneteenth") > strings.Index(first, "New Year") {
		t.Error("events not sorted by date")
	}
}

func TestEventFields(t *testing.T) {
	h := model.Holiday{Name: "Christmas", Date: time.Date(2026, 12, 23, 0, 0, 0, 0, time.UTC), Observed: true}
	feed := Generate([]model.Holiday{h})
	for _, want := range []string{
		"DTSTART;VALUE=DATE:20261223\r\n",
		"DTEND;VALUE=DATE:20261224\r\n", // exclusive end = next day
		"SUMMARY:Christmas (Observed)",
		"TRANSP:TRANSPARENT",
		"X-APPLE-CALENDAR-COLOR:#76B900",
	} {
		if !strings.Contains(feed, want) {
			t.Errorf("feed missing %q", want)
		}
	}
}

func TestEscapeAndFolding(t *testing.T) {
	got := escapeText("a,b;c\\d")
	if got != `a\,b\;c\\d` {
		t.Errorf("escapeText = %q", got)
	}
	// Long property lines must fold at <=75 octets per physical line and
	// must round-trip when a client unfolds (strip CRLF+space).
	var b strings.Builder
	line(&b, "X-WR-CALDESC", strings.Repeat("description text. ", 20))
	out := b.String()
	for _, phys := range strings.Split(strings.TrimRight(out, "\r\n"), "\r\n") {
		if len(phys) > 75 {
			t.Errorf("physical line of %d octets: %q", len(phys), phys)
		}
	}
	unfolded := strings.ReplaceAll(strings.TrimRight(out, "\r\n"), "\r\n ", "")
	if !strings.HasPrefix(unfolded, "X-WR-CALDESC:description text. ") {
		t.Errorf("unfold corrupted value: %q", unfolded)
	}
	// Folding must not corrupt multibyte characters.
	lineBuf := &strings.Builder{}
	line(lineBuf, "SUMMARY", strings.Repeat("é", 60))
	if strings.Contains(lineBuf.String(), "\ufffd") {
		t.Error("folding corrupted UTF-8")
	}
}

func TestUIDStable(t *testing.T) {
	h := model.Holiday{Name: "Free Day", Date: time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)}
	if h.UID() != h.UID() {
		t.Error("UID unstable")
	}
	other := model.Holiday{Name: "Free Day", Date: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)}
	if h.UID() == other.UID() {
		t.Error("UID collision across dates")
	}
}
