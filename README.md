# NVIDIA Holidays & Free Days

> **⚠️ Unofficial community project — not affiliated with, endorsed by, or sponsored by
> NVIDIA Corporation.** NVIDIA® is a registered trademark of NVIDIA Corporation.
> The data source is NVIDIA's public benefits page; if the two ever disagree, NVIDIA wins.

> A subscribable, always-fresh **ICS calendar feed** of NVIDIA's US company holidays
> and Free Days — scraped daily from NVIDIA's official
> [time-off page](https://www.nvidia.com/en-us/benefits/time-off/) and published
> via GitHub Pages. No API keys, no maintenance, one-time setup.

**📅 Landing page:** https://rajkumaar23.github.io/nvidia-holidays/
**🔗 Feed URL:** `https://rajkumaar23.github.io/nvidia-holidays/nvidia-holidays.ics`

## Subscribe

### iPhone / Mac (Apple Calendar)

1. **Settings → Calendar → Accounts → Add Account → Other → Add Subscribed Calendar**
   (on Mac: Calendar app → File → New Calendar Subscription)
2. Paste:
   ```
   https://rajkumaar23.github.io/nvidia-holidays/nvidia-holidays.ics
   ```
3. Name it *NVIDIA Holidays*, tap Done.

Or just open this one-tap link on your iPhone:
[`webcal://rajkumaar23.github.io/nvidia-holidays/nvidia-holidays.ics`](webcal://rajkumaar23.github.io/nvidia-holidays/nvidia-holidays.ics)

### Google Calendar

1. Open [calendar.google.com](https://calendar.google.com) → **Other calendars → + → From URL**
2. Paste the feed URL above → **Add calendar**.

(On the landing page there's also a one-click *Add to Google Calendar* button.)

### Android / anything else

Any calendar client that supports subscribing to an ICS URL works — paste the feed URL.

## What's in the feed

| | |
|---|---|
| **Coverage** | Every year currently published by NVIDIA (e.g. 2026 + 2027) |
| **Events** | Company/federal holidays (🟣 `#7E57C2`) and Free Days (🔵 `#1E88E5`) |
| **Type** | All-day, `TRANSP:TRANSPARENT` — never blocks your availability |
| **Refresh** | Feed advertises a 12-hour TTL (`X-PUBLISHED-TTL`); cron re-scrapes daily |
| **Stability** | Stable, content-derived UIDs → clients update in place, never duplicate |

## How it works

```
nvidia.com time-off page
        │  (daily @ 12:30 UTC, GitHub Actions cron)
        ▼
 cmd/gen (Go) ──► internal/scraper ──► internal/ics ──► docs/nvidia-holidays.ics
                                                    └──► docs/holidays.json
        │
        ▼
 actions/deploy-pages  ──►  GitHub Pages (landing page + feed)
```

- **Deterministic output** — same input always produces byte-identical bytes,
  so redeploys are no-ops and diffs are reviewable.
- **Failure-safe** — if the scrape or a sanity check fails, the workflow fails
  *before* deploying and the last good feed stays live.
- **Structure-change alarm** — if NVIDIA redesigns the page, the scraper fails
  loudly (`ErrNoHolidays`) instead of publishing an empty calendar; you get an
  email from Actions and a green-feed stays serving.

## Development

Requires Go ≥ 1.27. No runtime dependencies beyond `golang.org/x/net/html`.

```sh
make test        # gofmt check + go vet + go test
make gen         # scrape live and write docs/nvidia-holidays.ics + holidays.json
make dry-run     # print the generated feed to stdout
```

Project layout:

```
cmd/gen/            CLI entrypoint (flags: -url -out -json -timeout -dry-run)
internal/scraper/   HTTP fetch + HTML parsing of the time-off page
internal/ics/       RFC 5545 renderer (escaping, 75-octet folding, CRLF)
internal/model/     Holiday domain type, stable UIDs, sorting
docs/               Landing page (data fetched client-side at runtime)
.github/workflows/  refresh.yml — daily cron: lint → test → scrape → deploy
```

## Design decisions

- **HTML over the `.md` mirror** — NVIDIA's `.md` endpoint lags behind the
  rendered page (it can be months stale); the HTML is the source of truth.
- **Year-heading anchoring** — entries are only accepted under a
  `YYYY Holidays and Free Days` heading and the explicit date must agree with
  the heading year, so unrelated lists on the page can't leak in.
- **`TRANSP:TRANSPARENT`** — holidays show as all-day banners without marking
  you busy in scheduling.
- **Observed shifts kept explicit** — `Christmas (Observed)` preserves
  NVIDIA's exact observance date rather than guessing federal dates.

## Disclaimer

Unofficial project. Data comes from NVIDIA's public benefits page; this repo
is not affiliated with or endorsed by NVIDIA. If NVIDIA and this feed ever
disagree, **NVIDIA wins** — check
[the official page](https://www.nvidia.com/en-us/benefits/time-off/).

## License

MIT — see [LICENSE](LICENSE).
