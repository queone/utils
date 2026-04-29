package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/queone/governa-color"

	"github.com/spf13/cobra"
)

const (
	programName     = "cash5"
	programVersion  = "0.10.0"
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

	// Winning circle — inline image with winners highlighted (iTerm2 only)
	if isITerm2() {
		fmt.Printf("  %s:\n", color.Blu("WINNING CIRCLE"))
		displayCircleImage(lwn, "  ")
	}

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

func main() {
	runCLI()
}
