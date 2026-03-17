package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"utils/internal/color"

	"github.com/spf13/cobra"
)

const (
	programName     = "cash5"
	programVersion  = "0.9.4"
	lottery_warning = "This is basically lighting money on fire! Play for fun, not profit 😀"
)

// narrativeDate formats a time as "2026-feb-17" for summary/narrative lines
func narrativeDate(t time.Time) string {
	return fmt.Sprintf("%d-%s-%02d", t.Year(), strings.ToLower(t.Format("Jan")), t.Day())
}

func runDailyWithRand() error {
	existing, err := loadDraws()
	if err != nil {
		return err
	}

	// Check connectivity once, up front. All fetch decisions key off this.
	online := checkInternet()
	if !online {
		fmt.Printf("%s\n", color.Red("Internet is unreachable — showing cached data"))
	}

	// Auto-fetch if no data or data is too old (more than 7 days)
	needsFetch := false
	if len(existing) == 0 {
		if online {
			fmt.Println("Empty local draws.json file. Fetching last 365 drawings...")
			needsFetch = true
		}
	} else {
		// Check if newest draw is more than 7 days old
		sort.Slice(existing, func(i, j int) bool { return existing[i].DrawTime < existing[j].DrawTime })
		newest := time.UnixMilli(existing[len(existing)-1].DrawTime)
		weekAgo := time.Now().AddDate(0, 0, -7)

		if newest.Before(weekAgo) && online {
			fmt.Printf("Data is outdated (newest draw: %s). Fetching recent data...\n",
				narrativeDate(newest))
			needsFetch = true
		}
	}

	if needsFetch {
		allDraws, err := fetchAllDrawsIncremental(existing, saveDrawsCallback)
		if err != nil {
			return fmt.Errorf("failed to fetch draws: %w", err)
		}
		existing = allDraws
		fmt.Println()
	}

	// Fetch all missing recent draws up to yesterday (only when online)
	if online && len(existing) > 0 {
		sort.Slice(existing, func(i, j int) bool { return existing[i].DrawTime < existing[j].DrawTime })
		newest := time.UnixMilli(existing[len(existing)-1].DrawTime)
		today := time.Now().Truncate(24 * time.Hour)
		yesterday := today.AddDate(0, 0, -1)

		// If newest draw is before yesterday, we're missing some recent draws
		if newest.Before(yesterday) {
			fmt.Printf("Missing recent draws (newest: %s, need up to: %s). Fetching...\n",
				narrativeDate(newest), narrativeDate(yesterday))

			dateFrom := newest.AddDate(0, 0, 1)
			dateTo := time.Now()

			recentDraws, err := fetchDrawsByDateRange(dateFrom, dateTo, existing, saveDrawsCallback)
			if err == nil {
				existing = recentDraws
				fmt.Printf("Fetched recent draws. Total in database: %d\n\n", len(existing))
			} else if is404Error(err) {
				// Primary returned 404 — try backup sources transparently
				fmt.Printf("%s\n", color.Red(fmt.Sprintf("Primary source unavailable (%v) — trying backup...", err)))
				existing = tryBackupFetchers(existing, dateFrom, dateTo)
			} else {
				fmt.Printf("Warning: failed to fetch recent draws: %v\n", err)
			}
		}
	}

	// Display last 10 draws
	if err := displayLastNDraws(existing, 10); err != nil {
		return err
	}

	// Deduplicate and sort for analysis
	sort.Slice(existing, func(i, j int) bool { return existing[i].DrawTime < existing[j].DrawTime })
	seen := make(map[string]bool)
	var uniqueDraws []Draw
	for _, d := range existing {
		if !seen[d.ID] {
			seen[d.ID] = true
			uniqueDraws = append(uniqueDraws, d)
		}
	}

	if len(uniqueDraws) == 0 {
		return fmt.Errorf("no draws available")
	}

	// Build combo -> dates map for repeat checking (narrative format)
	comboHistory := make(map[string][]string)
	for _, d := range uniqueDraws {
		nums, err := extractPrimaryFive(&d)
		if err != nil {
			continue
		}
		sort.Ints(nums)
		key := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])
		date := narrativeDate(time.UnixMilli(d.DrawTime))
		comboHistory[key] = append(comboHistory[key], date)
	}

	// Get last winning numbers (LWN)
	lastDraw := uniqueDraws[len(uniqueDraws)-1]
	lwn, err := extractPrimaryFive(&lastDraw)
	if err != nil {
		return fmt.Errorf("failed to extract last winning numbers: %w", err)
	}
	sort.Ints(lwn)
	lwnKey := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", lwn[0], lwn[1], lwn[2], lwn[3], lwn[4])
	lwnDate := narrativeDate(time.UnixMilli(lastDraw.DrawTime))

	// Current Jackpot (skip fetch if offline)
	if online {
		jackpot, err := fetchCurrentJackpot()
		if err == nil && jackpot > 0 {
			fmt.Printf("  %s: %s\n", color.Blu("CURRENT JACKPOT"), color.Grn(formatCurrency(jackpot/100)))
		} else if len(uniqueDraws) > 0 {
			// Fall back to latest draw's estimated jackpot
			jp := uniqueDraws[len(uniqueDraws)-1].EstimatedJackpot
			if jp > 0 {
				fmt.Printf("  %s: %s\n", color.Blu("CURRENT JACKPOT"), color.Grn(formatCurrency(jp/100)))
			}
		}
	} else if len(uniqueDraws) > 0 {
		jp := uniqueDraws[len(uniqueDraws)-1].EstimatedJackpot
		if jp > 0 {
			fmt.Printf("  %s: %s %s\n", color.Blu("CURRENT JACKPOT"),
				color.Grn(formatCurrency(jp/100)), color.Gra("(cached)"))
		}
	}

	// LWN repeat check
	fmt.Printf("  %s: %s", color.Blu("LAST WINNING NUMBERS"), color.Grn(lwnKey))
	lwnDates := comboHistory[lwnKey]
	if len(lwnDates) > 1 {
		// Filter out the last draw date itself to find prior occurrences
		var priorDates []string
		for _, d := range lwnDates {
			if d != lwnDate {
				priorDates = append(priorDates, d)
			}
		}
		if len(priorDates) > 0 {
			fmt.Printf("  %s", color.Blu("REPEATED: "+strings.Join(priorDates, ", ")))
		} else {
			fmt.Printf("  %s", color.Gra("Never repeated"))
		}
	} else {
		fmt.Printf("  %s", color.Gra("Never repeated"))
	}
	fmt.Println()

	// Winning geometries — side-by-side grids with winners highlighted
	fmt.Printf("  %s:\n", color.Blu("WINNING GEOMETRIES"))
	displayGeometricGridSideBySide(lwn, "  ")

	// Closest matches to LWN (3+ matching numbers)
	type closeMatch struct {
		drawTime int64
		date     string
		nums     []int
		matches  int
	}
	var closeMatches []closeMatch
	for _, d := range uniqueDraws {
		dt := d.DrawTime
		dDate := narrativeDate(time.UnixMilli(dt))
		if dDate == lwnDate {
			continue
		}
		nums, err := extractPrimaryFive(&d)
		if err != nil {
			continue
		}
		sort.Ints(nums)
		mc := countMatches(lwn, nums)
		if mc >= 3 {
			closeMatches = append(closeMatches, closeMatch{dt, dDate, nums, mc})
		}
	}

	// Sort by match count desc, then by drawTime desc (most recent first within same match count)
	sort.Slice(closeMatches, func(i, j int) bool {
		if closeMatches[i].matches != closeMatches[j].matches {
			return closeMatches[i].matches > closeMatches[j].matches
		}
		return closeMatches[i].drawTime > closeMatches[j].drawTime
	})

	fmt.Printf("  %s:\n", color.Blu("CLOSEST 5 PREVIOUS WINNING MATCHES"))
	if len(closeMatches) == 0 {
		fmt.Printf("  %s\n", color.Gra("No previous draws with 3+ matching numbers"))
	} else {
		limit := min(len(closeMatches), 5)
		for _, cm := range closeMatches[:limit] {
			numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d",
				cm.nums[0], cm.nums[1], cm.nums[2], cm.nums[3], cm.nums[4])
			fmt.Printf("    %s  %s  %s\n",
				color.Grn(numStr), color.Grn(cm.date),
				color.Gra(fmt.Sprintf("(%d/5 match)", cm.matches)))
		}
	}

	// Generate intelligent recommendations
	recommendations := generateRecommendations(uniqueDraws)

	fmt.Printf("  %s:\n", color.Blu("RECOMMENDATION"))
	for _, rec := range recommendations {
		numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d",
			rec.numbers[0], rec.numbers[1], rec.numbers[2], rec.numbers[3], rec.numbers[4])
		fmt.Printf("    %s  %s\n", color.Grn(numStr), color.Gra(rec.strategy))
	}

	fmt.Printf("\n  %s\n", color.Red(lottery_warning))

	return nil
}

type recommendation struct {
	numbers  []int
	strategy string
}

// generateRecommendations creates 5 intelligent recommendations based on statistical analysis
func generateRecommendations(uniqueDraws []Draw) []recommendation {
	// Build frequency maps
	overallFreq := make(map[int]int)
	firstNumFreq := make(map[int]int)
	pos2Freq := make(map[int]int)
	middleNumFreq := make(map[int]int)
	pos4Freq := make(map[int]int)
	lastNumFreq := make(map[int]int)

	// Last 30 days for hot numbers
	latest := time.UnixMilli(uniqueDraws[len(uniqueDraws)-1].DrawTime)
	last30Days := latest.AddDate(0, 0, -30)
	freq30 := make(map[int]int)

	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil && len(nums) == 5 {
			drawDate := time.UnixMilli(uniqueDraws[i].DrawTime)

			firstNumFreq[nums[0]]++
			pos2Freq[nums[1]]++
			middleNumFreq[nums[2]]++
			pos4Freq[nums[3]]++
			lastNumFreq[nums[4]]++

			for _, n := range nums {
				overallFreq[n]++
				if drawDate.After(last30Days) {
					freq30[n]++
				}
			}
		}
	}

	var recs []recommendation

	// 1. Most Common by Position
	mostCommonFirst := findMostCommon(firstNumFreq)
	mostCommonSecond := findMostCommon(pos2Freq)
	mostCommonMiddle := findMostCommon(middleNumFreq)
	mostCommonFourth := findMostCommon(pos4Freq)
	mostCommonLast := findMostCommon(lastNumFreq)

	positionCombo := []int{mostCommonFirst.num, mostCommonSecond.num, mostCommonMiddle.num, mostCommonFourth.num, mostCommonLast.num}
	sort.Ints(positionCombo)
	recs = append(recs, recommendation{positionCombo, "Most common by position"})

	// 2. Most Frequent Overall (expansion-normalized)
	// Use robust pool expansion detection to normalize frequencies
	pe := detectPoolExpansion(uniqueDraws, overallFreq)
	normFreqInt := make(map[int]int)
	for n, count := range overallFreq {
		expected := expectedFreqForNumber(n, len(uniqueDraws), pe)
		if expected > 0 {
			// Ratio of observed/expected, scaled for integer ranking
			normFreqInt[n] = int(float64(count) / expected * 1000000)
		}
	}
	topNorm := findTopN(normFreqInt, 10)
	if len(topNorm) >= 5 {
		freqCombo := []int{topNorm[0].num, topNorm[1].num, topNorm[2].num, topNorm[3].num, topNorm[4].num}
		sort.Ints(freqCombo)
		recs = append(recs, recommendation{freqCombo, "Most frequent (expansion-adjusted)"})
	}

	// 3. Hot Numbers (most frequent in last 30 days)
	topHot := findTopN(freq30, 10)
	if len(topHot) >= 5 {
		hotCombo := []int{topHot[0].num, topHot[1].num, topHot[2].num, topHot[3].num, topHot[4].num}
		sort.Ints(hotCombo)
		recs = append(recs, recommendation{hotCombo, "Hot numbers last 30 days"})
	}

	// 4. Least Common by Position (expansion-normalized)
	normPos := [5]map[int]int{}
	posFreqs := [5]map[int]int{firstNumFreq, pos2Freq, middleNumFreq, pos4Freq, lastNumFreq}
	for p := range 5 {
		normPos[p] = make(map[int]int)
		for n, count := range posFreqs[p] {
			expected := expectedFreqForNumber(n, len(uniqueDraws), pe)
			if expected > 0 {
				normPos[p][n] = int(float64(count) / expected * 1000000)
			}
		}
	}
	leastNorm := [5]numCount{}
	for p := range 5 {
		leastNorm[p] = findLeastCommon(normPos[p])
	}
	leastPositionCombo := []int{leastNorm[0].num, leastNorm[1].num, leastNorm[2].num, leastNorm[3].num, leastNorm[4].num}
	sort.Ints(leastPositionCombo)
	recs = append(recs, recommendation{leastPositionCombo, "Least common by position (expansion-adjusted)"})

	// 5. Consecutive pair avoidance — the one statistically grounded signal
	consecCombo := generateConsecAvoidCombo()
	recs = append(recs, recommendation{consecCombo, "Consecutive pair avoidance"})

	return recs
}

func printUsage() {
	n := color.Whi2(programName)
	v := programVersion
	usage := fmt.Sprintf("%s v%s\n"+
		"NJ Cash 5 daily numbers recommender\n"+
		"\n"+
		"%s\n"+
		"  %s [options]\n"+
		"\n"+
		"%s\n"+
		"  -f             Fetch new draws since last run (within last year)\n"+
		"  -a             Display all previous drawings\n"+
		"  -s             Show statistics about historical data\n"+
		"  -g             Display geometric number grids\n"+
		"  -m             Show closest-match analysis for all drawings\n"+
		"  -o [N]         Show odds table for 1 to N combos played (default: 30)\n"+
		"  -d DATE        Show raw JSON for draws on DATE (format: 2026-02-06)\n"+
		"  -v             Show this help message and exit\n"+
		"  -h, -?         Show this help message and exit\n"+
		"\n"+
		"%s\n"+
		"  1. Display the last 10 draws\n"+
		"  2. Show current jackpot, last winning numbers, and closest matches\n"+
		"  3. Recommend 5 sets of numbers based on statistics\n"+
		"\n"+
		"%s\n"+
		"  %s\n"+
		"  %s -f\n"+
		"  %s -s\n"+
		"  %s -o 100\n"+
		"  %s -o\n",
		n, v, color.Whi2("Usage"), n, color.Whi2("Options"),
		color.Whi2("Running without switches will"), color.Whi2("Examples"),
		n, n, n, n, n)
	usage += "\n" + color.Red(lottery_warning) + "\n"
	fmt.Print(usage)
}

func runCLI() {
	// Handle -?, -h, --help before cobra — -? isn't a valid pflag character
	for _, arg := range os.Args[1:] {
		if arg == "-?" || arg == "-h" || arg == "--help" {
			printUsage()
			return
		}
	}

	// Handle -g before cobra
	for _, arg := range os.Args[1:] {
		if arg == "-g" {
			displayGeometricGridSideBySide(nil, "  ")
			return
		}
	}

	// Handle -o before cobra — cobra can't do optional-value flags properly
	for i, arg := range os.Args[1:] {
		if arg == "-o" {
			n := 30 // default
			if i+1 < len(os.Args[1:]) {
				if val, err := strconv.Atoi(os.Args[i+2]); err == nil && val > 0 {
					n = val
				}
			}
			displayOddsTable(n)
			return
		}
	}

	var fetchAll bool
	var showVersion bool
	var showAll bool
	var showStats bool
	var matchAnalysis bool
	var debugDate string

	root := &cobra.Command{
		Use:   programName,
		Short: "NJ Cash 5 daily numbers recommender",
		Run: func(cmd *cobra.Command, args []string) {
			if showVersion {
				printUsage()
				return
			}

			if debugDate != "" {
				existingDraws, err := loadDraws()
				if err != nil {
					log.Fatal(err)
				}

				if err := debugDrawByDate(existingDraws, debugDate); err != nil {
					log.Fatal(err)
				}
				return
			}

			if fetchAll {
				fmt.Println("Fetching all historical draws...")
				fmt.Printf("%-33s  %7s  %12s\n", "PERIOD", "DRAWS", "GRAND TOTAL")

				existingDraws, err := loadDraws()
				if err != nil {
					log.Fatal(err)
				}

				for {
					beforeCount := len(existingDraws)

					allDraws, err := fetchAllDrawsIncremental(existingDraws, saveDrawsCallback)
					if err != nil {
						log.Fatal(err)
					}

					newDrawsCount := len(allDraws) - beforeCount
					if newDrawsCount == 0 {
						fmt.Println("\nNo more historical data available.")
						break
					}

					existingDraws = allDraws
				}

				fmt.Printf("\nFetch complete! Total draws in database: %d\n", len(existingDraws))
				return
			}

			if showAll {
				existingDraws, err := loadDraws()
				if err != nil {
					log.Fatal(err)
				}

				if err := displayAllDraws(existingDraws); err != nil {
					log.Fatal(err)
				}
				return
			}

			if matchAnalysis {
				existingDraws, err := loadDraws()
				if err != nil {
					log.Fatal(err)
				}

				if err := displayMatchAnalysis(existingDraws); err != nil {
					log.Fatal(err)
				}
				return
			}

			if showStats {
				existingDraws, err := loadDraws()
				if err != nil {
					log.Fatal(err)
				}

				if err := displayStatistics(existingDraws); err != nil {
					log.Fatal(err)
				}
				return
			}

			if err := runDailyWithRand(); err != nil {
				log.Fatal(err)
			}
		},
	}

	root.Flags().BoolVarP(&fetchAll, "fetch-all", "f", false, "Fetch new draws since last run (within last year)")
	root.Flags().BoolVarP(&showAll, "all", "a", false, "Display all previous drawings")
	root.Flags().BoolVarP(&showStats, "stats", "s", false, "Show statistics about historical data")
	root.Flags().BoolVarP(&matchAnalysis, "match-analysis", "m", false, "Show closest-match analysis for all historical drawings")
	root.Flags().BoolVarP(&showVersion, "version", "v", false, "Show program version and usage")
	root.Flags().StringVarP(&debugDate, "debug", "d", "", "Show raw JSON for draws on specified date")

	// Disable default help flag
	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.CompletionOptions.DisableDefaultCmd = true
	root.Flags().BoolP("help", "h", false, "")
	root.Flags().Lookup("help").Hidden = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

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
// transpose=false → 5 rows × 9 cols; transpose=true → 9 rows × 5 cols.
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
			var n int
			if transpose {
				n = c*9 + r + 1
			} else {
				n = r*cols + c + 1
			}
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

// displayGeometricGrid prints a single grid to stdout.
func displayGeometricGrid(transpose bool) {
	for _, line := range buildGridLines(transpose, nil) {
		fmt.Println(line)
	}
}

// displayGeometricGridSideBySide prints the 9-col and 5-col grids side by side,
// with winning numbers highlighted in green. indent is the left margin prefix.
func displayGeometricGridSideBySide(winners []int, indent string) {
	highlight := make(map[int]bool)
	for _, n := range winners {
		highlight[n] = true
	}

	left := buildGridLines(false, highlight) // 5×9
	right := buildGridLines(true, highlight) // 9×5

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

	// Left grid visible width: 9 cells × 4 chars + 10 borders = 46
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

// is404Error reports whether an error originated from an HTTP 404 response.
func is404Error(err error) bool {
	return err != nil && strings.Contains(err.Error(), "404")
}

// tryBackupFetchers attempts each backup source in order, merges any draws found
// into existing, saves, and returns the result. Prints status for each attempt.
func tryBackupFetchers(existing []Draw, dateFrom, dateTo time.Time) []Draw {
	for _, fetcher := range backupFetchers {
		fmt.Printf("  Trying %s...\n", fetcher.Name())
		draws, err := fetcher.FetchRecent(dateFrom, dateTo)
		if err != nil {
			fmt.Printf("  %s failed: %v\n", fetcher.Name(), err)
			continue
		}
		if len(draws) == 0 {
			fmt.Printf("  %s: no new draws found\n", fetcher.Name())
			continue
		}

		// Merge: add draws not already in existing (match by date, since IDs differ
		// between primary and backup sources)
		existingDates := make(map[string]bool)
		for _, d := range existing {
			existingDates[time.UnixMilli(d.DrawTime).Format("2006-01-02")] = true
		}
		added := 0
		for _, d := range draws {
			dateKey := time.UnixMilli(d.DrawTime).Format("2006-01-02")
			if !existingDates[dateKey] {
				existing = append(existing, d)
				added++
			}
		}

		sort.Slice(existing, func(i, j int) bool { return existing[i].DrawTime < existing[j].DrawTime })

		if added > 0 {
			fmt.Printf("  %s: added %d draw(s) via backup\n", fetcher.Name(), added)
			if err := saveDrawsCallback(existing); err != nil {
				fmt.Printf("  Warning: failed to save after backup fetch: %v\n", err)
			}
		} else {
			fmt.Printf("  %s: draws already in cache\n", fetcher.Name())
		}
		return existing
	}

	fmt.Printf("%s\n", color.Red("All backup sources failed — showing cached data"))
	return existing
}

func main() {
	runCLI()
}
