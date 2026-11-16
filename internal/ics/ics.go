// Package ics renders holidays as an RFC 5545 iCalendar feed.
package ics

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rajkumaar23/nvidia-holidays/internal/model"
)

const (
	prodID   = "-//rajkumaar23//NVIDIA Holidays & Free Days//EN"
	calendar = "NVIDIA Holidays & Free Days"
	// NVIDIA US HQ timezone; all-day events are timezone-free but clients
	// use this hint for display and reminder defaults.
	wrTimezone = "America/Los_Angeles"
	refreshTTL = "PT12H"

	colorHoliday = "#76B900" // NVIDIA green
	colorFreeDay = "#1E88E5"
)

// Generate renders a complete VCALENDAR document for the given holidays.
// The output is deterministic: the same input always produces byte-identical
// bytes, which makes CI commits clean and idempotent.
func Generate(holidays []model.Holiday) string {
	sorted := make([]model.Holiday, len(holidays))
	copy(sorted, holidays)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Date.Equal(sorted[j].Date) {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].Date.Before(sorted[j].Date)
	})

	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	line(&b, "VERSION", "2.0")
	line(&b, "PRODID", prodID)
	line(&b, "CALSCALE", "GREGORIAN")
	line(&b, "METHOD", "PUBLISH")
	line(&b, "X-WR-CALNAME", calendar)
	line(&b, "X-WR-CALDESC", "US holidays and Free Days published by NVIDIA, auto-refreshed daily.")
	line(&b, "X-WR-TIMEZONE", wrTimezone)
	fmt.Fprintf(&b, "REFRESH-INTERVAL;VALUE=DURATION:%s\r\n", refreshTTL)
	fmt.Fprintf(&b, "X-PUBLISHED-TTL:%s\r\n", refreshTTL)

	for _, h := range sorted {
		writeEvent(&b, h)
	}
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}

func writeEvent(b *strings.Builder, h model.Holiday) {
	color := colorHoliday
	tag := "NVIDIA company holiday"
	if h.IsFreeDay() {
		color = colorFreeDay
		tag = "NVIDIA Free Day"
	}
	b.WriteString("BEGIN:VEVENT\r\n")
	line(b, "UID", h.UID())
	// Deterministic DTSTAMP derived from the event date.
	line(b, "DTSTAMP", h.Date.Format("20060102")+"T000000Z")
	fmt.Fprintf(b, "DTSTART;VALUE=DATE:%s\r\n", h.Date.Format("20060102"))
	fmt.Fprintf(b, "DTEND;VALUE=DATE:%s\r\n", h.Date.AddDate(0, 0, 1).Format("20060102"))
	line(b, "SUMMARY", h.Summary())
	line(b, "DESCRIPTION", tag)
	line(b, "CATEGORIES", tag)
	line(b, "TRANSP", "TRANSPARENT")
	line(b, "STATUS", "CONFIRMED")
	line(b, "X-APPLE-CALENDAR-COLOR", color)
	b.WriteString("END:VEVENT\r\n")
}

// line writes one RFC 5545 property: "NAME:escaped-value", folded so every
// physical line fits within 75 octets.
func line(b *strings.Builder, name, value string) {
	full := name + ":" + escapeText(value)
	first := true
	for {
		limit := 75
		if !first {
			limit = 74 // leave one octet for the leading continuation space
		}
		if len(full) <= limit {
			break
		}
		cut := limit
		for cut > 0 && !utf8Start(full[cut]) {
			cut--
		}
		b.WriteString(full[:cut])
		b.WriteString("\r\n ")
		full = full[cut:]
		first = false
	}
	b.WriteString(full)
	b.WriteString("\r\n")
}

// escapeText applies RFC 5545 TEXT escaping.
func escapeText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `;`, `\;`)
	s = strings.ReplaceAll(s, `,`, `\,`)
	return s
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }
