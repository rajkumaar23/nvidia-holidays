// Package model defines the domain types shared by the scraper and the
// ICS generator.
package model

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"
)

// Holiday is a single day off published by NVIDIA on its US benefits page.
type Holiday struct {
	// Name is the human-readable label, e.g. "Memorial Day" or "Free Day".
	Name string
	// Date is the day the holiday is observed (date-only, no time component).
	Date time.Time
	// Observed is true when NVIDIA observes the holiday on a different day
	// than the nominal date (e.g. "Christmas — Observed — Dec. 23, 2026").
	Observed bool
}

// IsFreeDay reports whether this entry is one of NVIDIA's discretionary
// "Free Day" closures rather than a named federal/company holiday.
func (h Holiday) IsFreeDay() bool {
	return strings.EqualFold(strings.TrimSpace(h.Name), "Free Day")
}

// Year returns the calendar year of the holiday date.
func (h Holiday) Year() int { return h.Date.Year() }

// Summary is the label used in the generated calendar, marking observed
// shifts explicitly, e.g. "Christmas (Observed)".
func (h Holiday) Summary() string {
	if h.Observed {
		return h.Name + " (Observed)"
	}
	return h.Name
}

// UID returns a stable, collision-resistant identifier for the event. It is
// derived purely from name+date so repeated generations produce byte-identical
// output, which keeps Git diffs and client re-subscriptions clean.
func (h Holiday) UID() string {
	htr := fnv.New64a()
	// io.Writer on hash never returns an error.
	_, _ = fmt.Fprintf(htr, "%s|%s", h.Name, h.Date.Format("2006-01-02"))
	return fmt.Sprintf("%x@nv-holidays.rajkumaar.co.in", htr.Sum64())
}

// ByDate sorts holidays chronologically, with name as tie-breaker so output
// is fully deterministic.
type ByDate []Holiday

func (a ByDate) Len() int { return len(a) }
func (a ByDate) Less(i, j int) bool {
	if a[i].Date.Equal(a[j].Date) {
		return a[i].Name < a[j].Name
	}
	return a[i].Date.Before(a[j].Date)
}
func (a ByDate) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
