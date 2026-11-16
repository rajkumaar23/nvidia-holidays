// Command gen scrapes the NVIDIA US time-off page and writes an ICS feed.
//
// Usage:
//
//	gen -out docs/nvidia-holidays.ics [-url <page>] [-json <file>]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rajkumaar23/nvidia-holidays/internal/ics"
	"github.com/rajkumaar23/nvidia-holidays/internal/model"
	"github.com/rajkumaar23/nvidia-holidays/internal/scraper"
)

func main() {
	var (
		url      = flag.String("url", scraper.DefaultURL, "NVIDIA time-off page URL")
		out      = flag.String("out", "docs/nvidia-holidays.ics", "output path for the .ics feed")
		jsonOut  = flag.String("json", "", "optional: also write holidays as JSON to this path")
		timeout  = flag.Duration("timeout", 30*time.Second, "HTTP timeout")
		inPlaceD = flag.Bool("dry-run", false, "print the feed to stdout instead of writing files")
	)
	flag.Parse()

	if err := run(*url, *out, *jsonOut, *timeout, *inPlaceD); err != nil {
		log.Fatalf("gen: %v", err)
	}
}

func run(url, out, jsonOut string, timeout time.Duration, dryRun bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout+5*time.Second)
	defer cancel()

	client := scraper.NewClient(url, &http.Client{Timeout: timeout})
	holidays, err := client.Fetch(ctx)
	if err != nil {
		return err
	}
	feed := ics.Generate(holidays)

	years := map[int]int{}
	for _, h := range holidays {
		years[h.Year()]++
	}
	fmt.Printf("scraped %d events across years:", len(holidays))
	for y, n := range years {
		fmt.Printf(" %d=%d", y, n)
	}
	fmt.Println()

	if dryRun {
		fmt.Print(feed)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	// 0o644: the feed is world-readable static content by design.
	if err := os.WriteFile(out, []byte(feed), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Printf("wrote %s\n", out)

	if jsonOut != "" {
		if err := writeJSON(jsonOut, holidays); err != nil {
			return err
		}
		fmt.Printf("wrote %s\n", jsonOut)
	}
	return nil
}

type jsonHoliday struct {
	Name     string `json:"name"`
	Date     string `json:"date"`
	Observed bool   `json:"observed"`
	FreeDay  bool   `json:"free_day"`
}

func writeJSON(path string, holidays []model.Holiday) error {
	items := make([]jsonHoliday, 0, len(holidays))
	for _, h := range holidays {
		items = append(items, jsonHoliday{
			Name:     h.Name,
			Date:     h.Date.Format("2006-01-02"),
			Observed: h.Observed,
			FreeDay:  h.IsFreeDay(),
		})
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
