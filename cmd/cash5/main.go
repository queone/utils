package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/queone/utl"
	"github.com/spf13/cobra"
)

const (
	program_name    = "cash5"
	program_version = "1.1.0"
)

func runDailyWithRand(r *rand.Rand) error {
	existing, err := loadDraws()
	if err != nil {
		return err
	}

	// Auto-fetch if no data or data is too old (more than 7 days)
	needsFetch := false
	if len(existing) == 0 {
		fmt.Println("Empty local draws.json file. Fetching last 365 drawings...")
		needsFetch = true
	} else {
		// Check if newest draw is more than 7 days old
		sort.Slice(existing, func(i, j int) bool { return existing[i].DrawTime < existing[j].DrawTime })
		newest := time.UnixMilli(existing[len(existing)-1].DrawTime)
		weekAgo := time.Now().AddDate(0, 0, -7)

		if newest.Before(weekAgo) {
			fmt.Printf("Data is outdated (newest draw: %s). Fetching recent data...\n",
				newest.Format("2006-01-02"))
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

	// Fetch all missing recent draws up to yesterday
	if len(existing) > 0 {
		sort.Slice(existing, func(i, j int) bool { return existing[i].DrawTime < existing[j].DrawTime })
		newest := time.UnixMilli(existing[len(existing)-1].DrawTime)
		today := time.Now().Truncate(24 * time.Hour)
		yesterday := today.AddDate(0, 0, -1)

		// If newest draw is before yesterday, we're missing some recent draws
		if newest.Before(yesterday) {
			fmt.Printf("Missing recent draws (newest: %s, need up to: %s). Fetching...\n",
				newest.Format("2006-01-02"), yesterday.Format("2006-01-02"))

			// Fetch from day after newest to today
			dateFrom := newest.AddDate(0, 0, 1)
			dateTo := time.Now()

			recentDraws, err := fetchDrawsByDateRange(dateFrom, dateTo, existing, saveDrawsCallback)
			if err == nil {
				existing = recentDraws
				fmt.Printf("Fetched recent draws. Total in database: %d\n\n", len(existing))
			} else {
				fmt.Printf("Warning: failed to fetch recent draws: %v\n", err)
			}
		}
	}

	// Display last 10 draws
	if err := displayLastNDraws(existing, 10); err != nil {
		return err
	}

	// Generate intelligent recommendations
	fmt.Printf("\n%s\n", utl.Gra("Please wait while calculating suggestions..."))
	recommendations := generateTop3Recommendations(existing)

	fmt.Printf("\n%s:\n", utl.Blu("RECOMMENDED SETS"))
	for _, rec := range recommendations {
		numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d",
			rec.numbers[0], rec.numbers[1], rec.numbers[2], rec.numbers[3], rec.numbers[4])
		fmt.Printf("  %s  %s\n", utl.Gre(numStr), utl.Gra(rec.strategy))
	}

	return nil
}

type recommendation struct {
	numbers  []int
	strategy string
}

// generateTop3Recommendations creates intelligent recommendations based on statistical analysis
func generateTop3Recommendations(draws []Draw) []recommendation {
	// Deduplicate
	seen := make(map[string]bool)
	uniqueDraws := []Draw{}
	for _, d := range draws {
		if !seen[d.ID] {
			seen[d.ID] = true
			uniqueDraws = append(uniqueDraws, d)
		}
	}

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

	// Build historical combinations for maximum distance (do this once)
	var historicalSets [][]int
	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil && len(nums) == 5 {
			sort.Ints(nums)
			historicalSets = append(historicalSets, nums)
		}
	}

	var recs []recommendation

	// 1. Most Frequent Overall Strategy
	topOverall := findTopN(overallFreq, 10)
	if len(topOverall) >= 5 {
		freqCombo := []int{topOverall[0].num, topOverall[1].num, topOverall[2].num, topOverall[3].num, topOverall[4].num}
		sort.Ints(freqCombo)
		recs = append(recs, recommendation{freqCombo, "(Most frequent all-time)"})
	}

	// 2. Maximum Distance Strategy (simulated annealing - run 3 times with fewer iterations)
	if len(historicalSets) > 0 {
		var bestResult annealingResult
		bestResult.bestScore = 0

		// Run annealing 3 times with 1000 iterations each (faster than 5x2000)
		for run := 0; run < 3; run++ {
			result := simulatedAnnealingSearch(historicalSets, 1000, 100.0, 0.95)
			if result.bestScore > bestResult.bestScore {
				bestResult = result
			}
		}

		recs = append(recs, recommendation{bestResult.bestCombo, "(Maximum distance)"})
	}

	// 3. Position-Based Strategy (most common in each position)
	mostCommonFirst := findMostCommon(firstNumFreq)
	mostCommonSecond := findMostCommon(pos2Freq)
	mostCommonMiddle := findMostCommon(middleNumFreq)
	mostCommonFourth := findMostCommon(pos4Freq)
	mostCommonLast := findMostCommon(lastNumFreq)

	positionCombo := []int{mostCommonFirst.num, mostCommonSecond.num, mostCommonMiddle.num, mostCommonFourth.num, mostCommonLast.num}
	sort.Ints(positionCombo)
	recs = append(recs, recommendation{positionCombo, "(Most common by position)"})

	// 4. Hot Numbers Strategy (most frequent in last 30 days)
	topHot := findTopN(freq30, 10)
	if len(topHot) >= 5 {
		hotCombo := []int{topHot[0].num, topHot[1].num, topHot[2].num, topHot[3].num, topHot[4].num}
		sort.Ints(hotCombo)
		recs = append(recs, recommendation{hotCombo, "(Hot numbers last 30 days)"})
	}

	return recs
}

func runCLI() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	var fetchAll bool
	var showVersion bool
	var showAll bool
	var showStats bool
	var debugDate string

	root := &cobra.Command{
		Use:   program_name,
		Short: "NJ Cash 5 daily numbers recommender",
		Run: func(cmd *cobra.Command, args []string) {
			if showVersion {
				fmt.Printf("%s v%s\n", program_name, program_version)
				fmt.Println("NJ Cash 5 daily numbers recommender.")
				fmt.Println("\nUsage:")
				fmt.Println("  -f             Fetch new draws since last run (within last year)")
				fmt.Println("  -a             Display all previous drawings")
				fmt.Println("  -s             Show statistics about historical data")
				fmt.Println("  -v             Show program version and usage")
				fmt.Println("  -d DATE        Show raw JSON for draws on DATE (format: 2026-02-06)")
				fmt.Println("\nRunning without switches will:")
				fmt.Println("  1. Display the last 10 draws")
				fmt.Println("  2. Recommend 4 sets of number based on statistics")
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

			if err := runDailyWithRand(r); err != nil {
				log.Fatal(err)
			}
		},
	}

	root.Flags().BoolVarP(&fetchAll, "fetch-all", "f", false, "Fetch new draws since last run (within last year)")
	root.Flags().BoolVarP(&showAll, "all", "a", false, "Display all previous drawings")
	root.Flags().BoolVarP(&showStats, "stats", "s", false, "Show statistics about historical data")
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

		fmt.Printf("%-12s  %-20s  %15s\n", drawDate, numStr, payout)
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
	fmt.Printf("%-12s  %-20s  %15s\n", "DATE", "WINNING NUMBERS", "5/5 PAYOUT")

	for _, d := range lastNDraws {
		nums, err := extractPrimaryFive(&d)
		if err != nil {
			continue
		}

		drawDate := time.UnixMilli(d.DrawTime).Format("2006-01-02")
		numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])

		// Pass full uniqueDraws array so we can find the next draw
		payout := formatWinner(uniqueDraws, &d)

		fmt.Printf("%-12s  %-20s  %15s\n", drawDate, numStr, payout)
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

func main() {
	runCLI()
}
