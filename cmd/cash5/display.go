package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"utils/internal/color"
)

func printTimestamp() {
	now := time.Now()
	ampm := "a"
	if now.Hour() >= 12 {
		ampm = "p"
	}
	fmt.Printf("%d-%s-%02d %s %s%s\n",
		now.Year(),
		strings.ToLower(now.Format("Jan")),
		now.Day(),
		now.Format("Mon"),
		now.Format("03:04"),
		ampm,
	)
}

func displayAllDraws(draws []Draw) error {
	if len(draws) == 0 {
		fmt.Println("No draws found")
		return nil
	}

	// Sort by date (oldest first)
	sort.Slice(draws, func(i, j int) bool {
		return draws[i].DrawTime < draws[j].DrawTime
	})

	// Deduplicate by draw ID
	seen := make(map[string]bool)
	uniqueDraws := []Draw{}
	for _, d := range draws {
		if !seen[d.ID] {
			seen[d.ID] = true
			uniqueDraws = append(uniqueDraws, d)
		}
	}

	// Print header
	printTimestamp()
	fmt.Printf("%-12s  %-20s  %15s\n", "DATE", "WINNING NUMBERS", "5/5 PAYOUT")

	for _, d := range uniqueDraws {
		nums, err := extractPrimaryFive(&d)
		if err != nil {
			continue
		}

		drawDate := time.UnixMilli(d.DrawTime).Format("2006-01-02")
		numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])

		// Pass full uniqueDraws array so we can find the next draw
		payout := formatWinner(uniqueDraws, &d)

		fmt.Printf("  %-12s  %-20s  %15s\n", drawDate, numStr, payout)
	}

	return nil
}

func displayLastNDraws(draws []Draw, n int) error {
	if len(draws) == 0 {
		fmt.Println("No draws found")
		return nil
	}

	// Sort by date (oldest first for proper context)
	sort.Slice(draws, func(i, j int) bool {
		return draws[i].DrawTime < draws[j].DrawTime
	})

	// Deduplicate by draw ID
	seen := make(map[string]bool)
	uniqueDraws := []Draw{}
	for _, d := range draws {
		if !seen[d.ID] {
			seen[d.ID] = true
			uniqueDraws = append(uniqueDraws, d)
		}
	}

	// Get last n draws
	startIdx := 0
	if len(uniqueDraws) > n {
		startIdx = len(uniqueDraws) - n
	}
	lastNDraws := uniqueDraws[startIdx:]

	// Print header
	printTimestamp()
	fmt.Printf("  %-12s  %-20s  %15s\n", "DATE", "WINNING NUMBERS", "5/5 PAYOUT")

	for _, d := range lastNDraws {
		nums, err := extractPrimaryFive(&d)
		if err != nil {
			continue
		}

		drawDate := time.UnixMilli(d.DrawTime).Format("2006-01-02")
		numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])

		// Pass full uniqueDraws array so we can find the next draw
		payout := formatWinner(uniqueDraws, &d)

		fmt.Printf("  %-12s  %-20s  %15s\n", drawDate, numStr, payout)
	}

	return nil
}

func debugDrawByDate(draws []Draw, dateStr string) error {
	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}

	// Deduplicate by draw ID first
	seen := make(map[string]bool)
	uniqueDraws := []Draw{}
	for _, d := range draws {
		if !seen[d.ID] {
			seen[d.ID] = true
			uniqueDraws = append(uniqueDraws, d)
		}
	}

	found := false
	for _, d := range uniqueDraws {
		drawDate := time.UnixMilli(d.DrawTime).Format("2006-01-02")
		if drawDate == targetDate.Format("2006-01-02") {
			found = true
			fmt.Printf("Draw for %s:\n", drawDate)
			fmt.Printf("ID: %s\n", d.ID)
			fmt.Printf("GameName: %s\n", d.GameName)
			fmt.Printf("Status: %s\n", d.Status)
			fmt.Printf("EstimatedJackpot: %d (= $%d)\n", d.EstimatedJackpot, d.EstimatedJackpot/100)
			fmt.Printf("Jackpot: %d\n", d.Jackpot)

			fmt.Printf("\nResults (count: %d):\n", len(d.Results))
			for i, r := range d.Results {
				fmt.Printf("  [%d] DrawType: %s\n", i, r.DrawType)
				fmt.Printf("      Primary: %v\n", r.Primary)
				fmt.Printf("      PrimaryRevealOrder: %v\n", r.PrimaryRevealOrder)
				fmt.Printf("      Winners: %d\n", r.Winners)
				fmt.Printf("      Payout: %d\n", r.Payout)
				fmt.Printf("      PrizeAmount: %d\n", r.PrizeAmount)
			}

			fmt.Printf("\nPrizeTiers (count: %d):\n", len(d.PrizeTiers))
			if len(d.PrizeTiers) == 0 {
				fmt.Println("  (empty)")
			}
			for i, pt := range d.PrizeTiers {
				fmt.Printf("  [%d] Tier: %s, Match: %s, Winners: %d\n", i, pt.Tier, pt.Match, pt.Winners)
				fmt.Printf("      PrizeAmount: %d, Prize: %d\n", pt.PrizeAmount, pt.Prize)
				fmt.Printf("      Description: %s\n", pt.Description)
			}

			fmt.Printf("\nPrizes (count: %d):\n", len(d.Prizes))
			if len(d.Prizes) == 0 {
				fmt.Println("  (empty)")
			}
			for i, p := range d.Prizes {
				fmt.Printf("  [%d] Level: %s, Winners: %d, Amount: %d\n", i, p.Level, p.Winners, p.Amount)
				fmt.Printf("      Description: %s\n", p.Description)
			}

			fmt.Printf("\nWinningNumbers: %v\n", d.WinningNumbers)
			fmt.Println("\n" + strings.Repeat("-", 60))
		}
	}

	if !found {
		fmt.Printf("No draw found for date %s\n", dateStr)
	}

	return nil
}

// displayOddsTable prints an odds table for 1 to maxCombos combos played
func displayOddsTable(maxCombos int) {
	const totalCombos = 1221759 // C(45,5)
	const ticketCost = 2

	// Try to get current jackpot for EV calculation
	var jackpotDollars int64
	jp, err := fetchCurrentJackpot()
	if err == nil && jp > 0 {
		jackpotDollars = jp / 100
	} else {
		// Fall back to latest draw's estimated jackpot
		draws, err := loadDraws()
		if err == nil && len(draws) > 0 {
			sort.Slice(draws, func(i, j int) bool { return draws[i].DrawTime < draws[j].DrawTime })
			est := draws[len(draws)-1].EstimatedJackpot
			if est > 0 {
				jackpotDollars = est / 100
			}
		}
	}

	fmt.Printf("NJ Cash 5 Odds Table. Total possible combinations: %s (C(45,5))\n",
		formatNumber(totalCombos))

	if jackpotDollars > 0 {
		fmt.Printf("Current jackpot: %s\n", formatCurrency(jackpotDollars))
	}

	if maxCombos > totalCombos {
		maxCombos = totalCombos
	}

	if jackpotDollars > 0 {
		fmt.Printf("%-9s  %5s    %-16s  %-14s  %6s\n", "COMBOS", "COST", "ODDS", "PROBABILITY", "EV")
	} else {
		fmt.Printf("%-9s  %5s    %-16s  %s\n", "COMBOS", "COST", "ODDS", "PROBABILITY")
	}

	for n := 1; n <= maxCombos; n++ {
		cost := n * ticketCost
		oneInX := (totalCombos + n - 1) / n // ceiling division
		prob := float64(n) / float64(totalCombos)

		if jackpotDollars > 0 {
			ev := prob*float64(jackpotDollars) - float64(cost)
			fmt.Printf("%6d     %5s    1 in %-11s  %-14s  %6s\n",
				n,
				formatCurrency(int64(cost)),
				formatNumber(oneInX),
				formatProbability(prob*100),
				formatEV(ev))
		} else {
			fmt.Printf("%6d     %5s    1 in %-11s  %s%%\n",
				n,
				formatCurrency(int64(cost)),
				formatNumber(oneInX),
				formatProbability(prob*100))
		}
	}
}

// formatEV formats expected value with a sign prefix
func formatEV(ev float64) string {
	if ev >= 0 {
		return fmt.Sprintf("+$%.2f", ev)
	}
	return fmt.Sprintf("-$%.2f", -ev)
}

// formatProbability formats a percentage with enough decimal places to be meaningful
func formatProbability(pct float64) string {
	if pct >= 1.0 {
		return fmt.Sprintf("%.4f%%", pct)
	}
	return fmt.Sprintf("%.6f%%", pct)
}

// buildGridLines returns the lines of an ASCII-bordered grid of all 45 numbers.
// transpose=false -> 5 rows x 9 cols; transpose=true -> 9 rows x 5 cols.
// Numbers in the highlight set are rendered in green.
func buildGridLines(transpose bool, highlight map[int]bool) []string {
	var rows, cols int
	if transpose {
		rows, cols = 9, 5
	} else {
		rows, cols = 5, 9
	}

	cell := "────"
	border := func(left, mid, right string) string {
		parts := make([]string, cols)
		for i := range parts {
			parts[i] = cell
		}
		return left + strings.Join(parts, mid) + right
	}

	var lines []string
	lines = append(lines, border("┌", "┬", "┐"))
	for r := range rows {
		row := ""
		for c := range cols {
			n := r*cols + c + 1
			cell := fmt.Sprintf("%02d", n)
			if highlight[n] {
				cell = color.GrnR(cell)
			}
			row += fmt.Sprintf("│ %s ", cell)
		}
		row += "│"
		lines = append(lines, row)
		if r < rows-1 {
			lines = append(lines, border("├", "┼", "┤"))
		}
	}
	lines = append(lines, border("└", "┴", "┘"))
	return lines
}

// displayGeometricGridSideBySide prints the 9-col and 5-col grids side by side,
// with winning numbers highlighted in green. indent is the left margin prefix.
func displayGeometricGridSideBySide(winners []int, indent string) {
	highlight := make(map[int]bool)
	for _, n := range winners {
		highlight[n] = true
	}

	left := buildGridLines(false, highlight) // 5x9
	right := buildGridLines(true, highlight) // 9x5

	// The left grid (5 rows of data, 9 cols) has 11 lines (5 data + 6 borders).
	// The right grid (9 rows of data, 5 cols) has 19 lines.
	// Pad the shorter one so they align at the top.
	maxLines := len(left)
	if len(right) > maxLines {
		maxLines = len(right)
	}
	for len(left) < maxLines {
		left = append(left, "")
	}
	for len(right) < maxLines {
		right = append(right, "")
	}

	// Left grid visible width: 9 cells x 4 chars + 10 borders = 46
	// But ANSI codes make len() unreliable, so use a fixed pad width.
	leftWidth := 9*4 + 10 // "│ XX │ XX │ ... │" = 46 chars visible

	for i := range maxLines {
		l := left[i]
		r := right[i]
		// Pad left line to fixed visible width
		visLen := visibleLen(l)
		pad := ""
		if visLen < leftWidth {
			pad = strings.Repeat(" ", leftWidth-visLen)
		}
		fmt.Printf("%s%s%s   %s\n", indent, l, pad, r)
	}
}

// visibleLen returns the visible length of a string, ignoring ANSI escape codes.
func visibleLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		n++
	}
	return n
}
