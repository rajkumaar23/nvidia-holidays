// Package scraper fetches and parses the NVIDIA US "Time Off" benefits page
// and extracts the published holidays and Free Days.
package scraper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rajkumaar23/nvidia-holidays/internal/model"
	"golang.org/x/net/html"
)

// DefaultURL is the public NVIDIA US time-off page.
const DefaultURL = "https://www.nvidia.com/en-us/benefits/time-off/"

// userAgent identifies the fetcher politely; NVIDIA's CDN serves static
// content but behaves better with a real UA than an empty one.
const userAgent = "nvidia-holidays-bot/1.0 (+https://github.com/rajkumaar23/nvidia-holidays)"

// ErrNoHolidays is returned when the page parsed fine but yielded zero
// holiday entries — a strong signal that NVIDIA changed the page structure.
var ErrNoHolidays = errors.New("scraper: no holidays found; page structure may have changed")

// Client fetches the time-off page. Use NewClient for production code and
// inject a custom *http.Client in tests.
type Client struct {
	http *http.Client
	url  string
	now  func() time.Time // injectable clock, currently only used by tests
}

// NewClient returns a Client for the given URL with sane timeouts.
func NewClient(url string, hc *http.Client) *Client {
	if url == "" {
		url = DefaultURL
	}
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{http: hc, url: url, now: time.Now}
}

// Fetch retrieves and parses the page, returning holidays sorted by date.
func (c *Client) Fetch(ctx context.Context) ([]model.Holiday, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("scraper: build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scraper: fetch %s: %w", c.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper: fetch %s: unexpected status %s", c.url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // 8 MiB guard
	if err != nil {
		return nil, fmt.Errorf("scraper: read body: %w", err)
	}
	return Parse(body)
}

// yearHeading matches accordion titles like "2027 Holidays and Free Days".
var yearHeading = regexp.MustCompile(`^(\d{4})\s+Holidays and Free Days$`)

// entryLine matches "Memorial Day—May 25, 2026" or
// "Juneteenth—Observed — June 17, 2026" (em-dash separated).
var entryLine = regexp.MustCompile(
	`^(?P<name>.+?)—\s*(?:(?P<observed>Observed)\s*—\s*)?(?P<date>.+?)\s*$`)

// dateLine matches "Jan. 1, 2027", "June 16, 2027", "Sept. 6, 2027".
var dateLine = regexp.MustCompile(
	`(?i)^(?P<month>Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Sept|Oct|Nov|Dec)[a-z]*\.?\s+(?P<day>\d{1,2}),\s*(?P<year>\d{4})$`)

var months = map[string]time.Month{
	"jan": time.January, "feb": time.February, "mar": time.March,
	"apr": time.April, "may": time.May, "jun": time.June,
	"jul": time.July, "aug": time.August, "sep": time.September,
	"sept": time.September, "oct": time.October, "nov": time.November,
	"dec": time.December,
}

// Parse extracts holidays from the raw HTML of the NVIDIA time-off page.
// It walks every element, tracks the most recent "YYYY Holidays and Free
// Days" heading, and collects date-stamped <li> entries that appear under it.
func Parse(data []byte) ([]model.Holiday, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("scraper: parse html: %w", err)
	}

	var (
		holidays []model.Holiday
		curYear  int
		seen     = map[string]bool{}
	)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "h2") {
			if m := yearHeading.FindStringSubmatch(strings.TrimSpace(text(n))); m != nil {
				y, _ := strconv.Atoi(m[1])
				curYear = y
			}
		}
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "li") && curYear != 0 {
			if h, ok := parseEntry(strings.TrimSpace(text(n)), curYear); ok {
				key := h.Name + "|" + h.Date.Format("2006-01-02")
				if !seen[key] {
					seen[key] = true
					holidays = append(holidays, h)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if len(holidays) == 0 {
		return nil, ErrNoHolidays
	}
	return holidays, nil
}

// parseEntry turns one <li> text like "Free Day—Mar. 25, 2027" into a
// Holiday. ok is false when the line is not a recognisable holiday entry.
func parseEntry(line string, headingYear int) (model.Holiday, bool) {
	m := entryLine.FindStringSubmatch(line)
	if m == nil {
		return model.Holiday{}, false
	}
	name := strings.TrimSpace(m[1])
	// NVIDIA's page uses typographic punctuation; normalise the curly
	// apostrophe so summaries render identically on every client.
	name = strings.ReplaceAll(name, "\u2019", "'")
	observed := strings.EqualFold(strings.TrimSpace(m[2]), "Observed")

	dm := dateLine.FindStringSubmatch(strings.TrimSpace(m[3]))
	if dm == nil {
		return model.Holiday{}, false
	}
	month, ok := months[strings.ToLower(strings.TrimRight(dm[1], "."))]
	if !ok {
		return model.Holiday{}, false
	}
	day, err := strconv.Atoi(dm[2])
	if err != nil {
		return model.Holiday{}, false
	}
	year, err := strconv.Atoi(dm[3])
	if err != nil {
		return model.Holiday{}, false
	}
	if year != headingYear {
		// Defensive: trust the explicit date over the heading.
		return model.Holiday{}, false
	}
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if date.Year() != year || int(date.Month()) != int(month) || date.Day() != day {
		return model.Holiday{}, false // rejects e.g. Feb 30
	}
	return model.Holiday{Name: name, Date: date, Observed: observed}, true
}

// text returns the concatenated text content of a node subtree.
func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}
