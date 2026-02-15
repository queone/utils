package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const baseURL = "https://www.njlottery.com/api/v1/draw-games/draws/page"

// fetchPageWithSize fetches one page of draws from the API with custom page size
func fetchPageWithSize(page int, size int, dateFrom, dateTo int64) ([]Draw, error) {
	url := fmt.Sprintf(
		"%s?game-names=Cash+5&status=CLOSED&size=%d&page=%d&date-from=%d&date-to=%d",
		baseURL,
		size,
		page,
		dateFrom,
		dateTo,
	)

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://www.njlottery.com/en-us/drawgames/jerseycash.html")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	return apiResp.Draws, nil
}

// fetchDrawsByDateRange pages through the API for a specific date-from/date-to range
// Saves intermediate results via saveCallback after each page
// Stops when it reaches ~365 draws or runs out of data
func fetchDrawsByDateRange(from, to time.Time, existing []Draw, saveCallback func([]Draw) error) ([]Draw, error) {
	const pageSize = 365 // Fetch a full year at once
	const maxDraws = 365 // Stop after approximately one year of daily draws

	all := append([]Draw(nil), existing...) // copy existing
	page := 0
	fromMs := from.UnixMilli()
	toMs := to.UnixMilli()
	newDrawsCount := 0

	for {
		var draws []Draw
		var err error

		for attempt := 1; attempt <= 5; attempt++ {
			draws, err = fetchPageWithSize(page, pageSize, fromMs, toMs)
			if err != nil {
				if strings.Contains(err.Error(), "500") {
					fmt.Printf("Server 500 on page %d, retry %d/5...\n", page, attempt)
					time.Sleep(2 * time.Second)
					continue
				}
				return nil, err
			}
			break
		}

		if err != nil {
			// If we already have some draws, return what we have instead of erroring out
			if newDrawsCount > 0 {
				fmt.Printf("Error fetching page %d after retries. Saved %d draws successfully.\n", page, newDrawsCount)
				return all, nil
			}
			return nil, fmt.Errorf("page %d failed after retries: %w", page, err)
		}

		if len(draws) == 0 {
			break
		}

		all = append(all, draws...)
		newDrawsCount += len(draws)
		sort.Slice(all, func(i, j int) bool { return all[i].DrawTime < all[j].DrawTime })

		if saveCallback != nil {
			if err := saveCallback(all); err != nil {
				fmt.Printf("Warning: failed to save after page %d: %v\n", page, err)
			}
		}

		// For initial fetch with large page size, if we got close to a full year, stop
		// This avoids trying to fetch page 1 which causes 500 errors
		if page == 0 && len(draws) >= 350 {
			break
		}

		// Check if we've already fetched enough
		if newDrawsCount >= maxDraws {
			fmt.Printf("Reached ~365 draws limit, stopping fetch\n")
			break
		}

		page++
	}

	return all, nil
}

// fetchAllDrawsIncremental fetches draws in year-long chunks
func fetchAllDrawsIncremental(existing []Draw, saveCallback func([]Draw) error) ([]Draw, error) {
	var dateFrom, dateTo time.Time

	if len(existing) == 0 {
		// First run: fetch last year (from 1 year ago to now)
		dateTo = time.Now()
		dateFrom = dateTo.AddDate(-1, 0, 0)
	} else {
		// Subsequent runs: fetch the year BEFORE the oldest draw
		sort.Slice(existing, func(i, j int) bool { return existing[i].DrawTime < existing[j].DrawTime })
		oldest := time.UnixMilli(existing[0].DrawTime)

		// Fetch from (oldest - 1 year) to oldest
		dateTo = oldest.Add(-time.Millisecond)
		dateFrom = dateTo.AddDate(-1, 0, 0)
	}

	beforeCount := len(existing)
	allDraws, err := fetchDrawsByDateRange(dateFrom, dateTo, existing, saveCallback)
	newCount := len(allDraws) - beforeCount

	// Print summary line
	periodStr := fmt.Sprintf("%s → %s", dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02"))
	fmt.Printf("%-33s  %7d  %12d\n", periodStr, newCount, len(allDraws))

	return allDraws, err
}

// fetchPage fetches a single page (default size=1) for the latest draw
func fetchPage(page int) ([]Draw, error) {
	const defaultSize = 1
	from := time.Now().AddDate(-10, 0, 0)
	to := time.Now().AddDate(0, 0, 1)

	return fetchPageWithSize(page, defaultSize, from.UnixMilli(), to.UnixMilli())
}

// saveDrawsCallback persists draws to $HOME/.config/cash5/draws.json
func saveDrawsCallback(draws []Draw) error {
	path := fmt.Sprintf("%s/.config/cash5/draws.json", os.Getenv("HOME"))
	if err := os.MkdirAll(fmt.Sprintf("%s/.config/cash5", os.Getenv("HOME")), 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(draws)
}
