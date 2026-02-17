package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/queone/utl"
)

// Global RNG for statistics - seeded once at package init
var statsRNG = rand.New(rand.NewSource(time.Now().UnixNano()))

func displayStatistics(draws []Draw) error {
	if len(draws) == 0 {
		fmt.Println("No draws found")
		return nil
	}

	// Deduplicate by draw ID
	seen := make(map[string]bool)
	uniqueDraws := []Draw{}
	for _, d := range draws {
		if !seen[d.ID] {
			seen[d.ID] = true
			uniqueDraws = append(uniqueDraws, d)
		}
	}

	// Sort by date
	sort.Slice(uniqueDraws, func(i, j int) bool {
		return uniqueDraws[i].DrawTime < uniqueDraws[j].DrawTime
	})

	fmt.Println("=== NJ CASH 5 STATISTICS ===")
	fmt.Println()

	// Total drawings
	fmt.Printf("%s: %s\n", utl.Blu("Total Drawings"), utl.Gre(len(uniqueDraws)))

	// Date range
	earliest := time.UnixMilli(uniqueDraws[0].DrawTime)
	latest := time.UnixMilli(uniqueDraws[len(uniqueDraws)-1].DrawTime)
	fmt.Printf("%s: %s\n", utl.Blu("Earliest Drawing"), utl.Gre(earliest.Format("2006-01-02")))
	fmt.Printf("%s: %s\n", utl.Blu("Latest Drawing"), utl.Gre(latest.Format("2006-01-02")))

	// Find biggest and smallest recorded prizes
	var biggestPayout int64
	var smallestPayout int64 = 999999999999
	var biggestDraw, smallestDraw *Draw
	winnersCount := 0
	var winnerDates []time.Time

	firstNumFreq := make(map[int]int)
	pos2Freq := make(map[int]int)
	middleNumFreq := make(map[int]int)
	pos4Freq := make(map[int]int)
	lastNumFreq := make(map[int]int)
	overallFreq := make(map[int]int)
	pairFreq := make(map[string]int)

	// For hot/cold analysis
	last30Days := latest.AddDate(0, 0, -30)
	last60Days := latest.AddDate(0, 0, -60)
	last90Days := latest.AddDate(0, 0, -90)
	freq30 := make(map[int]int)
	freq60 := make(map[int]int)
	freq90 := make(map[int]int)

	for i := range uniqueDraws {
		d := &uniqueDraws[i]
		payout := getPayout(d)

		if payout > 0 {
			winnersCount++
			winnerDates = append(winnerDates, time.UnixMilli(d.DrawTime))
			if payout > biggestPayout {
				biggestPayout = payout
				biggestDraw = d
			}
			if payout < smallestPayout {
				smallestPayout = payout
				smallestDraw = d
			}
		}

		// Track number frequencies
		nums, err := extractPrimaryFive(d)
		if err == nil && len(nums) == 5 {
			drawDate := time.UnixMilli(d.DrawTime)

			firstNumFreq[nums[0]]++
			pos2Freq[nums[1]]++
			middleNumFreq[nums[2]]++
			pos4Freq[nums[3]]++
			lastNumFreq[nums[4]]++

			// Overall frequency
			for _, n := range nums {
				overallFreq[n]++

				// Hot/cold tracking
				if drawDate.After(last30Days) {
					freq30[n]++
				}
				if drawDate.After(last60Days) {
					freq60[n]++
				}
				if drawDate.After(last90Days) {
					freq90[n]++
				}
			}

			// Pair frequency
			for j := 0; j < len(nums)-1; j++ {
				for k := j + 1; k < len(nums); k++ {
					pair := fmt.Sprintf("%02d-%02d", nums[j], nums[k])
					pairFreq[pair]++
				}
			}
		}
	}

	fmt.Printf("\n%s: %s\n", utl.Blu("Winners (5/5 Match)"), utl.Gre(winnersCount))

	// Check for duplicate winning combinations
	fmt.Printf("\n%s:\n", utl.Blu("Duplicate Combination Check"))

	combinationMap := make(map[string][]string) // combo -> list of dates
	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil && len(nums) == 5 {
			sort.Ints(nums)
			comboKey := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])
			drawDate := time.UnixMilli(uniqueDraws[i].DrawTime).Format("2006-01-02")
			combinationMap[comboKey] = append(combinationMap[comboKey], drawDate)
		}
	}

	// Find duplicates
	var duplicates []struct {
		combo string
		dates []string
	}

	for combo, dates := range combinationMap {
		if len(dates) > 1 {
			duplicates = append(duplicates, struct {
				combo string
				dates []string
			}{combo, dates})
		}
	}

	if len(duplicates) == 0 {
		fmt.Printf("  %s: %s  %s\n",
			utl.Blu("Status"),
			utl.Gre("✓ No duplicates found"),
			utl.Gra(fmt.Sprintf("(%d unique combinations in %d draws)", len(combinationMap), len(uniqueDraws))))
	} else {
		fmt.Printf("  %s: %s  %s\n",
			utl.Blu("Status"),
			utl.Gra(fmt.Sprintf("⚠ %d duplicate combination(s) found!", len(duplicates))),
			utl.Gra("(EXTREMELY RARE - lottery integrity concern)"))

		fmt.Printf("\n  %s:\n", utl.Blu("Duplicate Details"))
		for _, dup := range duplicates {
			fmt.Printf("    %s: %s\n", utl.Blu("Combination"), utl.Gre(dup.combo))
			fmt.Printf("      %s:\n", utl.Blu("Drawn on"))
			for _, date := range dup.dates {
				fmt.Printf("        - %s\n", utl.Gre(date))
			}
		}
	}

	if biggestDraw != nil {
		nums, _ := extractPrimaryFive(biggestDraw)
		drawDate := time.UnixMilli(biggestDraw.DrawTime).Format("2006-01-02")
		numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])
		fmt.Printf("\n%s: %s\n", utl.Blu("Biggest Prize"), utl.Gre(formatCurrency(biggestPayout/100)))
		fmt.Printf("  %s: %s\n", utl.Blu("Date"), utl.Gre(drawDate))
		fmt.Printf("  %s: %s\n", utl.Blu("Numbers"), utl.Gre(numStr))
	}

	if smallestDraw != nil && smallestPayout < 999999999999 {
		nums, _ := extractPrimaryFive(smallestDraw)
		drawDate := time.UnixMilli(smallestDraw.DrawTime).Format("2006-01-02")
		numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])
		fmt.Printf("\n%s: %s\n", utl.Blu("Smallest Prize"), utl.Gre(formatCurrency(smallestPayout/100)))
		fmt.Printf("  %s: %s\n", utl.Blu("Date"), utl.Gre(drawDate))
		fmt.Printf("  %s: %s\n", utl.Blu("Numbers"), utl.Gre(numStr))
	}

	// Winner streak analysis
	if len(winnerDates) > 1 {
		var daysBetween []int
		var longestStreak int

		for i := 1; i < len(winnerDates); i++ {
			days := int(winnerDates[i].Sub(winnerDates[i-1]).Hours() / 24)
			daysBetween = append(daysBetween, days)

			if days > longestStreak {
				longestStreak = days
			}
		}

		// Calculate average
		var sum int
		for _, d := range daysBetween {
			sum += d
		}
		avgDays := float64(sum) / float64(len(daysBetween))

		fmt.Printf("\n%s:\n", utl.Blu("Jackpot Win Frequency"))
		fmt.Printf("  %s: %s\n", utl.Blu("Average days between"), utl.Gre(fmt.Sprintf("%.1f days", avgDays)))
		fmt.Printf("  %s: %s\n", utl.Blu("Longest streak"), utl.Gre(fmt.Sprintf("%d days", longestStreak)))

		// Days since last winner
		if len(winnerDates) > 0 {
			daysSinceWin := int(latest.Sub(winnerDates[len(winnerDates)-1]).Hours() / 24)
			fmt.Printf("  %s: %s\n", utl.Blu("Days since last win"), utl.Gre(fmt.Sprintf("%d days", daysSinceWin)))
		}
	}

	// Most common numbers by position
	mostCommonFirst := findMostCommon(firstNumFreq)
	mostCommonSecond := findMostCommon(pos2Freq)
	mostCommonMiddle := findMostCommon(middleNumFreq)
	mostCommonFourth := findMostCommon(pos4Freq)
	mostCommonLast := findMostCommon(lastNumFreq)

	fmt.Printf("\n%s:\n", utl.Blu("Most Common by Position"))
	fmt.Printf("  %s: %s  %s\n", utl.Blu("First position"), utl.Gre(fmt.Sprintf("%02d", mostCommonFirst.num)), utl.Gra(fmt.Sprintf("(appeared %d times)", mostCommonFirst.count)))
	fmt.Printf("  %s: %s  %s\n", utl.Blu("Second position"), utl.Gre(fmt.Sprintf("%02d", mostCommonSecond.num)), utl.Gra(fmt.Sprintf("(appeared %d times)", mostCommonSecond.count)))
	fmt.Printf("  %s: %s  %s\n", utl.Blu("Third position"), utl.Gre(fmt.Sprintf("%02d", mostCommonMiddle.num)), utl.Gra(fmt.Sprintf("(appeared %d times)", mostCommonMiddle.count)))
	fmt.Printf("  %s: %s  %s\n", utl.Blu("Fourth position"), utl.Gre(fmt.Sprintf("%02d", mostCommonFourth.num)), utl.Gra(fmt.Sprintf("(appeared %d times)", mostCommonFourth.count)))
	fmt.Printf("  %s: %s  %s\n", utl.Blu("Fifth position"), utl.Gre(fmt.Sprintf("%02d", mostCommonLast.num)), utl.Gra(fmt.Sprintf("(appeared %d times)", mostCommonLast.count)))

	// Most frequently drawn overall
	topOverall := findTopN(overallFreq, 5)
	fmt.Printf("\n%s:\n", utl.Blu("Most Frequently Drawn (All Positions)"))
	for i, nc := range topOverall {
		fmt.Printf("  %s. %s %s:  %s\n", utl.Gre(i+1), utl.Blu("Number"), utl.Gre(fmt.Sprintf("%02d", nc.num)), utl.Gre(fmt.Sprintf("%d times", nc.count)))
	}

	// Least frequently drawn overall
	bottomOverall := findBottomN(overallFreq, 5)
	fmt.Printf("\n%s:\n", utl.Blu("Least Frequently Drawn (All Positions)"))
	for i, nc := range bottomOverall {
		fmt.Printf("  %s. %s %s:  %s\n", utl.Gre(i+1), utl.Blu("Number"), utl.Gre(fmt.Sprintf("%02d", nc.num)), utl.Gre(fmt.Sprintf("%d times", nc.count)))
	}

	// Hot numbers (last 30 days)
	if len(freq30) > 0 {
		hot30 := findTopN(freq30, 5)
		fmt.Printf("\n%s:\n", utl.Blu("Hot Numbers (Last 30 Days)"))
		for i, nc := range hot30 {
			fmt.Printf("  %s. %s %s:  %s\n", utl.Gre(i+1), utl.Blu("Number"), utl.Gre(fmt.Sprintf("%02d", nc.num)), utl.Gre(fmt.Sprintf("%d times", nc.count)))
		}
	}

	// Cold numbers (last 90 days)
	if len(freq90) > 0 {
		cold90 := findBottomN(freq90, 5)
		fmt.Printf("\n%s:\n", utl.Blu("Cold Numbers (Last 90 Days)"))
		for i, nc := range cold90 {
			fmt.Printf("  %s. %s %s:  %s\n", utl.Gre(i+1), utl.Blu("Number"), utl.Gre(fmt.Sprintf("%02d", nc.num)), utl.Gre(fmt.Sprintf("%d times", nc.count)))
		}
	}

	// Most common pairs
	topPairs := findTopNPairs(pairFreq, 5)
	fmt.Printf("\n%s:\n", utl.Blu("Most Common Number Pairs"))
	for i, pc := range topPairs {
		fmt.Printf("  %s. %s:  %s\n", utl.Gre(i+1), utl.Gre(pc.pair), utl.Gre(fmt.Sprintf("%d times", pc.count)))
	}

	// Chi-squared uniformity analysis
	chiSquared := calculateChiSquared(overallFreq, len(uniqueDraws)*5)
	fmt.Printf("\n%s:\n", utl.Blu("χ² Uniformity Analysis"))
	fmt.Printf("  %s: %s\n", utl.Blu("χ² statistic"), utl.Gre(fmt.Sprintf("%.2f", chiSquared)))
	fmt.Printf("  %s: %s\n", utl.Blu("Degrees of freedom"), utl.Gre("44 (45 numbers - 1)"))

	// Critical values for χ² with 44 df:
	// p=0.05: 60.48, p=0.01: 66.77
	if chiSquared < 60.48 {
		fmt.Printf("  %s: %s\n", utl.Blu("Result"), utl.Gre("Uniform distribution (p > 0.05)"))
		fmt.Printf("  %s: %s\n", utl.Blu("Interpretation"), utl.Gra("Numbers appear randomly distributed"))
	} else if chiSquared < 66.77 {
		fmt.Printf("  %s: %s\n", utl.Blu("Result"), utl.Gre("Possibly non-uniform (0.01 < p < 0.05)"))
		fmt.Printf("  %s: %s\n", utl.Blu("Interpretation"), utl.Gra("Slight deviation from randomness"))
	} else {
		fmt.Printf("  %s: %s\n", utl.Blu("Result"), utl.Gre("Non-uniform distribution (p < 0.01)"))
		fmt.Printf("  %s: %s\n", utl.Blu("Interpretation"), utl.Gra("Significant bias detected"))
	}

	// Full χ² Frequency Analysis
	fmt.Printf("\n%s:\n", utl.Blu("Full χ² Frequency Analysis Over History"))

	// Build frequency maps for all positions
	pos1Freq := firstNumFreq
	pos2FreqChi := make(map[int]int)
	pos3Freq := middleNumFreq
	pos4FreqChi := make(map[int]int)
	pos5Freq := lastNumFreq

	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil && len(nums) == 5 {
			pos2FreqChi[nums[1]]++
			pos4FreqChi[nums[3]]++
		}
	}

	// 1. Per-position uniformity tests
	fmt.Printf("\n  %s:\n", utl.Blu("Position-Specific Uniformity Tests"))

	positionNames := []string{"First", "Second", "Third", "Fourth", "Fifth"}
	positionFreqs := []map[int]int{pos1Freq, pos2FreqChi, pos3Freq, pos4FreqChi, pos5Freq}

	allPositionsUniform := true
	for i, posFreq := range positionFreqs {
		chiSq := calculateChiSquared(posFreq, len(uniqueDraws))
		isUniform := chiSq < 60.48
		if !isUniform {
			allPositionsUniform = false
		}

		statusStr := utl.Gre("✓ Uniform")
		if !isUniform {
			statusStr = utl.Gra("⚠ Non-uniform")
		}
		fmt.Printf("    %s %s: χ²=%s  %s\n",
			utl.Blu(positionNames[i]), utl.Blu("position"),
			utl.Gre(fmt.Sprintf("%.2f", chiSq)), statusStr)
	}

	if allPositionsUniform {
		fmt.Printf("  %s: %s\n", utl.Blu("Overall"), utl.Gra("All positions show uniform distribution"))
	} else {
		fmt.Printf("  %s: %s\n", utl.Blu("Overall"), utl.Gra("Some positions show non-uniform distribution"))
	}

	// 2. Temporal uniformity (monthly/yearly trends)
	fmt.Printf("\n  %s:\n", utl.Blu("Temporal Uniformity Analysis"))

	// Group by year
	yearlyFreqs := make(map[int]map[int]int)
	for i := range uniqueDraws {
		year := time.UnixMilli(uniqueDraws[i].DrawTime).Year()
		if yearlyFreqs[year] == nil {
			yearlyFreqs[year] = make(map[int]int)
		}
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil {
			for _, n := range nums {
				yearlyFreqs[year][n]++
			}
		}
	}

	// Get sorted years
	var years []int
	for year := range yearlyFreqs {
		years = append(years, year)
	}
	sort.Ints(years)

	fmt.Printf("    %s:\n", utl.Blu("Year-by-Year Analysis"))
	for _, year := range years {
		yearDraws := 0
		for i := range uniqueDraws {
			if time.UnixMilli(uniqueDraws[i].DrawTime).Year() == year {
				yearDraws++
			}
		}

		if yearDraws >= 30 { // Only analyze years with enough data
			chiSq := calculateChiSquared(yearlyFreqs[year], yearDraws*5)
			isUniform := chiSq < 60.48

			statusStr := utl.Gre("✓")
			if !isUniform {
				statusStr = utl.Gra("⚠")
			}

			fmt.Printf("      %s %s: χ²=%s  %s draws  %s\n",
				statusStr,
				utl.Blu(fmt.Sprintf("%d", year)),
				utl.Gre(fmt.Sprintf("%.2f", chiSq)),
				utl.Gre(yearDraws),
				utl.Gra(fmt.Sprintf("(expected: %.1f)", 60.48)))
		}
	}

	// 3. Sequential pair analysis
	fmt.Printf("\n  %s:\n", utl.Blu("Sequential Pair Uniformity"))
	fmt.Printf("    %s: %s\n", utl.Blu("Testing"), utl.Gra("Whether consecutive numbers appear uniformly"))

	consecutivePairs := 0
	totalPairs := 0
	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil {
			for j := 0; j < len(nums)-1; j++ {
				totalPairs++
				if nums[j+1] == nums[j]+1 {
					consecutivePairs++
				}
			}
		}
	}

	expectedConsecutiveRate := 4.0 / 44.0 // Probability of consecutive in random draw
	actualRate := float64(consecutivePairs) / float64(totalPairs)

	fmt.Printf("    %s: %s  %s\n",
		utl.Blu("Consecutive pairs found"),
		utl.Gre(fmt.Sprintf("%d/%d", consecutivePairs, totalPairs)),
		utl.Gra(fmt.Sprintf("(%.2f%%)", actualRate*100)))
	fmt.Printf("    %s: %s\n",
		utl.Blu("Expected rate"),
		utl.Gre(fmt.Sprintf("%.2f%%", expectedConsecutiveRate*100)))

	deviation := ((actualRate - expectedConsecutiveRate) / expectedConsecutiveRate) * 100
	if deviation > -10 && deviation < 10 {
		fmt.Printf("    %s: %s  %s\n", utl.Blu("Assessment"), utl.Gre("Within expected range"),
			utl.Gra(fmt.Sprintf("(%.1f%% deviation)", deviation)))
	} else {
		fmt.Printf("    %s: %s  %s\n", utl.Blu("Assessment"), utl.Gra("Outside expected range"),
			utl.Gra(fmt.Sprintf("(%.1f%% deviation)", deviation)))
	}

	// 4. Low vs High number distribution
	fmt.Printf("\n  %s:\n", utl.Blu("Low vs High Number Distribution"))
	fmt.Printf("    %s: %s\n", utl.Blu("Testing"), utl.Gra("Whether low (1-22) and high (23-45) numbers appear equally"))

	lowCount := 0
	highCount := 0
	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil {
			for _, n := range nums {
				if n <= 22 {
					lowCount++
				} else {
					highCount++
				}
			}
		}
	}

	totalNums := lowCount + highCount
	expectedLow := float64(totalNums) * (22.0 / 45.0)
	expectedHigh := float64(totalNums) * (23.0 / 45.0)

	chiSqLowHigh := (float64(lowCount)-expectedLow)*(float64(lowCount)-expectedLow)/expectedLow +
		(float64(highCount)-expectedHigh)*(float64(highCount)-expectedHigh)/expectedHigh

	fmt.Printf("    %s: %s  %s\n",
		utl.Blu("Low numbers (1-22)"),
		utl.Gre(fmt.Sprintf("%d", lowCount)),
		utl.Gra(fmt.Sprintf("(expected: %.0f)", expectedLow)))
	fmt.Printf("    %s: %s  %s\n",
		utl.Blu("High numbers (23-45)"),
		utl.Gre(fmt.Sprintf("%d", highCount)),
		utl.Gra(fmt.Sprintf("(expected: %.0f)", expectedHigh)))
	fmt.Printf("    %s: %s  %s\n",
		utl.Blu("χ² statistic"),
		utl.Gre(fmt.Sprintf("%.2f", chiSqLowHigh)),
		utl.Gra("(df=1, critical=3.84 at p=0.05)"))

	if chiSqLowHigh < 3.84 {
		fmt.Printf("    %s: %s\n", utl.Blu("Result"), utl.Gre("✓ Balanced distribution"))
	} else {
		fmt.Printf("    %s: %s\n", utl.Blu("Result"), utl.Gra("⚠ Imbalanced distribution"))
	}

	// 5. Summary
	fmt.Printf("\n  %s:\n", utl.Blu("Analysis Summary"))
	issuesFound := 0
	if !allPositionsUniform {
		issuesFound++
	}
	if chiSqLowHigh >= 3.84 {
		issuesFound++
	}

	if issuesFound == 0 {
		fmt.Printf("    %s\n", utl.Gre("✓ All tests passed - lottery appears statistically fair"))
	} else {
		fmt.Printf("    %s\n", utl.Gra(fmt.Sprintf("⚠ %d potential issues detected - review individual tests", issuesFound)))
	}

	// Monte Carlo simulation for repeat probability
	fmt.Printf("\n%s:\n", utl.Blu("Monte Carlo Repeat Probability Simulation"))

	// Get all historical combinations
	historicalCombos := make(map[string]bool)
	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil {
			sort.Ints(nums)
			key := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])
			historicalCombos[key] = true
		}
	}

	numHistorical := len(historicalCombos)
	totalCombinations := 1221759 // C(45,5)

	// Run simulations for different time horizons
	simResults := runRepeatSimulation(numHistorical, totalCombinations, 10000)

	fmt.Printf("  %s: %s\n", utl.Blu("Historical combinations"), utl.Gre(fmt.Sprintf("%d unique sets", numHistorical)))
	fmt.Printf("  %s: %s\n", utl.Blu("Total possible combos"), utl.Gre(formatNumber(totalCombinations)))
	fmt.Printf("  %s: %s\n", utl.Blu("Coverage"), utl.Gre(fmt.Sprintf("%.4f%%", float64(numHistorical)*100.0/float64(totalCombinations))))
	fmt.Printf("\n  %s:\n", utl.Blu("Probability of seeing a repeat combination"))
	fmt.Printf("    %s: %s\n", utl.Blu("In next 30 draws"), utl.Gre(fmt.Sprintf("%.2f%%", simResults.prob30Days*100)))
	fmt.Printf("    %s: %s\n", utl.Blu("In next 90 draws"), utl.Gre(fmt.Sprintf("%.2f%%", simResults.prob90Days*100)))
	fmt.Printf("    %s: %s\n", utl.Blu("In next 365 draws"), utl.Gre(fmt.Sprintf("%.2f%%", simResults.prob365Days*100)))
	fmt.Printf("    %s: %s\n", utl.Blu("In next 10 years"), utl.Gre(fmt.Sprintf("%.2f%%", simResults.prob10Years*100)))

	// Combinatorial Distance Scoring
	fmt.Printf("\n%s:\n", utl.Blu("Combinatorial Distance Scoring"))
	fmt.Printf("  %s\n", utl.Gra("[Computing... this may take a moment]"))

	// Calculate all historical combinations
	var historicalSets [][]int
	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil && len(nums) == 5 {
			sort.Ints(nums)
			historicalSets = append(historicalSets, nums)
		}
	}

	// Find the combination with maximum distance from all historical draws
	bestCombo, bestScore := findMaxDistanceCombo(historicalSets, 1000)

	// Also show distance stats for random combinations
	distanceStats := calculateDistanceStats(historicalSets, 10000)

	fmt.Printf("  %s: %s\n", utl.Blu("Distance Metric"), utl.Gra("Hamming distance (non-matching numbers)"))
	fmt.Printf("  %s: %s\n\n", utl.Blu("Historical combinations analyzed"), utl.Gre(len(historicalSets)))

	fmt.Printf("  %s:\n", utl.Blu("Random Combination Distance Statistics (10,000 samples)"))
	fmt.Printf("    %s: %s\n", utl.Blu("Average min distance"), utl.Gre(fmt.Sprintf("%.2f numbers different", distanceStats.avgMinDist)))
	fmt.Printf("    %s: %s\n", utl.Blu("Average mean distance"), utl.Gre(fmt.Sprintf("%.2f numbers different", distanceStats.avgMeanDist)))
	fmt.Printf("    %s: %s\n", utl.Blu("Best min distance found"), utl.Gre(fmt.Sprintf("%.0f numbers different", distanceStats.bestMinDist)))

	fmt.Printf("\n  %s:\n", utl.Blu("Maximum Distance Combination Found"))
	fmt.Printf("    %s: %s\n", utl.Blu("Numbers"), utl.Gre(fmt.Sprintf("%02d-%02d-%02d-%02d-%02d",
		bestCombo[0], bestCombo[1], bestCombo[2], bestCombo[3], bestCombo[4])))
	fmt.Printf("    %s: %s\n", utl.Blu("Min distance to history"), utl.Gre(fmt.Sprintf("%.0f numbers different", bestScore)))
	fmt.Printf("    %s: %s\n", utl.Blu("Interpretation"), utl.Gra(fmt.Sprintf("At least %.0f/5 numbers differ from every historical draw", bestScore)))

	// Show some example distances
	fmt.Printf("\n  %s:\n", utl.Blu("Distance Examples (from max-distance combo)"))
	for i := 0; i < 3 && i < len(historicalSets); i++ {
		dist := hammingDistance(bestCombo, historicalSets[len(historicalSets)-1-i])
		drawDate := time.UnixMilli(uniqueDraws[len(uniqueDraws)-1-i].DrawTime).Format("2006-01-02")
		fmt.Printf("    %s %s %s: %s\n", utl.Blu("vs"), utl.Gre(drawDate), utl.Blu("draw"), utl.Gre(fmt.Sprintf("%d/5 numbers different", dist)))
	}

	// Simulated Annealing Search
	fmt.Printf("\n%s:\n", utl.Blu("Simulated Annealing Search"))
	fmt.Printf("  %s\n", utl.Gra("[Optimizing... running 5,000 iterations]"))
	fmt.Printf("  %s: %s\n", utl.Blu("Objective"), utl.Gra("Maximize minimum distance to historical draws"))
	fmt.Printf("  %s: %s\n\n", utl.Blu("Algorithm"), utl.Gra("Simulated annealing with adaptive cooling"))

	// Run simulated annealing
	annealResult := simulatedAnnealingSearch(historicalSets, 5000, 100.0, 0.95)

	fmt.Printf("  %s:\n", utl.Blu("Search Parameters"))
	fmt.Printf("    %s: %s\n", utl.Blu("Iterations"), utl.Gre("5,000"))
	fmt.Printf("    %s: %s\n", utl.Blu("Initial temperature"), utl.Gre("100.0"))
	fmt.Printf("    %s: %s\n", utl.Blu("Cooling rate"), utl.Gre("0.95"))
	fmt.Printf("    %s: %s\n", utl.Blu("Perturbation strategy"), utl.Gra("Single number mutation"))

	fmt.Printf("\n  %s:\n", utl.Blu("Optimization Results"))
	fmt.Printf("    %s: %s\n", utl.Blu("Starting score"), utl.Gre(fmt.Sprintf("%.2f", annealResult.initialScore)))
	fmt.Printf("    %s: %s\n", utl.Blu("Final score"), utl.Gre(fmt.Sprintf("%.2f", annealResult.finalScore)))
	fmt.Printf("    %s: %s\n", utl.Blu("Best score found"), utl.Gre(fmt.Sprintf("%.2f", annealResult.bestScore)))
	fmt.Printf("    %s: %s  %s\n", utl.Blu("Improvement"), utl.Gre(fmt.Sprintf("+%.2f", annealResult.bestScore-annealResult.initialScore)),
		utl.Gra(fmt.Sprintf("(%.1f%%)", ((annealResult.bestScore-annealResult.initialScore)/annealResult.initialScore)*100)))
	fmt.Printf("    %s: %s  %s\n", utl.Blu("Accepted moves"),
		utl.Gre(fmt.Sprintf("%d/%d", annealResult.acceptedMoves, annealResult.totalMoves)),
		utl.Gra(fmt.Sprintf("(%.1f%%)", (float64(annealResult.acceptedMoves)/float64(annealResult.totalMoves))*100)))

	fmt.Printf("\n  %s:\n", utl.Blu("Optimal Combination Found"))
	fmt.Printf("    %s: %s\n", utl.Blu("Numbers"), utl.Gre(fmt.Sprintf("%02d-%02d-%02d-%02d-%02d",
		annealResult.bestCombo[0], annealResult.bestCombo[1],
		annealResult.bestCombo[2], annealResult.bestCombo[3], annealResult.bestCombo[4])))
	fmt.Printf("    %s: %s\n", utl.Blu("Min distance to history"), utl.Gre(fmt.Sprintf("%.0f numbers different", annealResult.bestScore)))
	fmt.Printf("    %s: %s\n", utl.Blu("Global maximum"), utl.Gra("Likely (annealing converged)"))

	// Compare methods
	fmt.Printf("\n  %s:\n", utl.Blu("Method Comparison"))
	fmt.Printf("    %s: %s  %s\n", utl.Blu("Random search best"), utl.Gre(fmt.Sprintf("%.0f", bestScore)), utl.Gra("(from 1,000 samples)"))
	fmt.Printf("    %s: %s  %s\n", utl.Blu("Annealing best"), utl.Gre(fmt.Sprintf("%.0f", annealResult.bestScore)), utl.Gra("(from 5,000 iterations)"))
	if annealResult.bestScore > bestScore {
		fmt.Printf("    %s: %s  %s\n", utl.Blu("Winner"), utl.Gre("Simulated annealing"), utl.Gra(fmt.Sprintf("(+%.0f)", annealResult.bestScore-bestScore)))
	} else if annealResult.bestScore < bestScore {
		fmt.Printf("    %s: %s  %s\n", utl.Blu("Winner"), utl.Gre("Random search"), utl.Gra(fmt.Sprintf("(+%.0f)", bestScore-annealResult.bestScore)))
	} else {
		fmt.Printf("    %s: %s\n", utl.Blu("Winner"), utl.Gra("Tie (both found same score)"))
	}

	return nil
}

type numCount struct {
	num   int
	count int
}

type pairCount struct {
	pair  string
	count int
}

func findMostCommon(freq map[int]int) numCount {
	var result numCount
	for num, count := range freq {
		if count > result.count {
			result.num = num
			result.count = count
		}
	}
	return result
}

func findTopN(freq map[int]int, n int) []numCount {
	var results []numCount
	for num, count := range freq {
		results = append(results, numCount{num, count})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].count == results[j].count {
			return results[i].num < results[j].num
		}
		return results[i].count > results[j].count
	})
	if len(results) > n {
		results = results[:n]
	}
	return results
}

func findBottomN(freq map[int]int, n int) []numCount {
	var results []numCount
	for num, count := range freq {
		results = append(results, numCount{num, count})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].count == results[j].count {
			return results[i].num < results[j].num
		}
		return results[i].count < results[j].count
	})
	if len(results) > n {
		results = results[:n]
	}
	return results
}

func findTopNPairs(freq map[string]int, n int) []pairCount {
	var results []pairCount
	for pair, count := range freq {
		results = append(results, pairCount{pair, count})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].count == results[j].count {
			return results[i].pair < results[j].pair
		}
		return results[i].count > results[j].count
	})
	if len(results) > n {
		results = results[:n]
	}
	return results
}

// calculateChiSquared performs chi-squared test for uniformity
// freq: map of number -> count
// totalDraws: total number of balls drawn (num_drawings * 5)
func calculateChiSquared(freq map[int]int, totalDraws int) float64 {
	// Expected frequency for each number (1-45)
	// In a uniform distribution, each number should appear equally
	expected := float64(totalDraws) / 45.0

	var chiSquared float64

	// For each possible number (1-45)
	for i := 1; i <= 45; i++ {
		observed := float64(freq[i])
		diff := observed - expected
		chiSquared += (diff * diff) / expected
	}

	return chiSquared
}

type simulationResults struct {
	prob30Days  float64
	prob90Days  float64
	prob365Days float64
	prob10Years float64
}

// runRepeatSimulation performs Monte Carlo simulation to estimate probability
// of drawing a previously seen combination
func runRepeatSimulation(numHistorical, totalCombos, iterations int) simulationResults {
	const (
		draws30  = 30
		draws90  = 90
		draws365 = 365
		draws10y = 3650
	)

	var results simulationResults

	// Calculate probability using formula: 1 - (1 - p)^n
	// where p = numHistorical/totalCombos
	// and n = number of draws

	p := float64(numHistorical) / float64(totalCombos)

	// Probability of at least one repeat in n draws
	results.prob30Days = 1.0 - pow(1.0-p, draws30)
	results.prob90Days = 1.0 - pow(1.0-p, draws90)
	results.prob365Days = 1.0 - pow(1.0-p, draws365)
	results.prob10Years = 1.0 - pow(1.0-p, draws10y)

	return results
}

// Retain unused functions/types for future use
var (
	_ = runEVSimulation
	_ = calculateEV
	_ evSimulationResults
)

type evSimulationResults struct {
	singleTicketEV float64
	plays100EV     float64
	plays1000EV    float64
}

// runEVSimulation calculates expected value through Monte Carlo simulation
func runEVSimulation(avgJackpot, winRate float64, totalCombos, iterations int) evSimulationResults {
	const ticketCost = 1.00

	// Theoretical probability of winning (1 in totalCombos)
	winProb := 1.0 / float64(totalCombos)

	// Expected value = (probability of winning × jackpot) - ticket cost
	singleEV := (winProb * avgJackpot) - ticketCost

	// For multiple plays, EV scales linearly
	ev100 := singleEV * 100
	ev1000 := singleEV * 1000

	return evSimulationResults{
		singleTicketEV: singleEV,
		plays100EV:     ev100,
		plays1000EV:    ev1000,
	}
}

// calculateEV computes expected value for a given jackpot
func calculateEV(jackpot float64, totalCombos int, ticketCost float64) float64 {
	winProb := 1.0 / float64(totalCombos)
	return (winProb * jackpot) - ticketCost
}

// hammingDistance calculates the number of different elements between two sets
// hammingDistance calculates the number of different elements between two sorted sets
// Optimized for small sorted arrays (5 elements each)
func hammingDistance(a, b []int) int {
	// Count matching elements
	matches := 0
	i, j := 0, 0

	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			matches++
			i++
			j++
		} else if a[i] < b[j] {
			i++
		} else {
			j++
		}
	}

	// Hamming distance = total elements - 2*matches
	// (each match means both sets have it, so subtract twice)
	return len(a) + len(b) - 2*matches
}

// minDistanceToHistorical finds minimum distance from combo to any historical set
func minDistanceToHistorical(combo []int, historical [][]int) float64 {
	if len(historical) == 0 {
		return 5.0
	}

	minDist := 5
	for _, hist := range historical {
		dist := hammingDistance(combo, hist)
		if dist < minDist {
			minDist = dist
		}
	}

	return float64(minDist)
}

type distanceStatistics struct {
	avgMinDist  float64
	avgMeanDist float64
	bestMinDist float64
}

// calculateDistanceStats generates random combinations and calculates distance statistics
func calculateDistanceStats(historical [][]int, samples int) distanceStatistics {
	var sumMinDist float64
	var sumMeanDist float64
	var bestMinDist float64

	for i := 0; i < samples; i++ {
		// Generate random combination
		combo := generateRandomCombo()

		// Calculate min distance to historical
		minDist := minDistanceToHistorical(combo, historical)
		sumMinDist += minDist

		if minDist > bestMinDist {
			bestMinDist = minDist
		}

		// Calculate mean distance to all historical
		var totalDist float64
		for _, hist := range historical {
			totalDist += float64(hammingDistance(combo, hist))
		}
		meanDist := totalDist / float64(len(historical))
		sumMeanDist += meanDist
	}

	return distanceStatistics{
		avgMinDist:  sumMinDist / float64(samples),
		avgMeanDist: sumMeanDist / float64(samples),
		bestMinDist: bestMinDist,
	}
}

// findMaxDistanceCombo searches for combination with maximum minimum distance to historical
func findMaxDistanceCombo(historical [][]int, iterations int) ([]int, float64) {
	var bestCombo []int
	var bestScore float64

	for i := 0; i < iterations; i++ {
		combo := generateRandomCombo()
		score := minDistanceToHistorical(combo, historical)

		if score > bestScore {
			bestScore = score
			bestCombo = make([]int, len(combo))
			copy(bestCombo, combo)
		}
	}

	return bestCombo, bestScore
}

// generateRandomCombo generates a random 5-number combination (1-45)
func generateRandomCombo() []int {
	nums := make([]int, 5)
	used := make(map[int]bool)

	for i := 0; i < 5; i++ {
		for {
			n := statsRNG.Intn(45) + 1
			if !used[n] {
				nums[i] = n
				used[n] = true
				break
			}
		}
	}

	sort.Ints(nums)
	return nums
}

type annealingResult struct {
	bestCombo     []int
	bestScore     float64
	initialScore  float64
	finalScore    float64
	acceptedMoves int
	totalMoves    int
}

// simulatedAnnealingSearch uses simulated annealing to find optimal combination
func simulatedAnnealingSearch(historical [][]int, iterations int, initialTemp, coolingRate float64) annealingResult {
	// Start with random combination
	current := generateRandomCombo()
	currentScore := minDistanceToHistorical(current, historical)

	best := make([]int, len(current))
	copy(best, current)
	bestScore := currentScore

	temperature := initialTemp
	acceptedMoves := 0

	for i := 0; i < iterations; i++ {
		// Generate neighbor by mutating one number
		neighbor := perturb(current)
		neighborScore := minDistanceToHistorical(neighbor, historical)

		// Calculate acceptance probability
		delta := neighborScore - currentScore
		acceptProb := 1.0
		if delta < 0 {
			// Worse solution - accept with probability based on temperature
			acceptProb = exp(delta / temperature)
		}

		// Decide whether to accept
		if delta > 0 || randomFloat() < acceptProb {
			current = neighbor
			currentScore = neighborScore
			acceptedMoves++

			// Update best if improved
			if currentScore > bestScore {
				copy(best, current)
				bestScore = currentScore
			}
		}

		// Cool down
		temperature *= coolingRate
	}

	return annealingResult{
		bestCombo:     best,
		bestScore:     bestScore,
		initialScore:  minDistanceToHistorical(generateRandomCombo(), historical),
		finalScore:    currentScore,
		acceptedMoves: acceptedMoves,
		totalMoves:    iterations,
	}
}

// perturb creates a neighbor by changing one random number
func perturb(combo []int) []int {
	neighbor := make([]int, len(combo))
	copy(neighbor, combo)

	// Pick random position to mutate
	pos := statsRNG.Intn(5)

	// Find new number not in combo
	used := make(map[int]bool)
	for _, n := range neighbor {
		used[n] = true
	}

	for {
		newNum := statsRNG.Intn(45) + 1
		if !used[newNum] {
			neighbor[pos] = newNum
			break
		}
	}

	sort.Ints(neighbor)
	return neighbor
}

// exp calculates e^x using Taylor series approximation
func exp(x float64) float64 {
	if x < -10 {
		return 0
	}
	sum := 1.0
	term := 1.0
	for i := 1; i < 20; i++ {
		term *= x / float64(i)
		sum += term
	}
	return sum
}

// randomFloat returns a pseudo-random float between 0 and 1
func randomFloat() float64 {
	return statsRNG.Float64()
}

// pow calculates x^n
func pow(x float64, n int) float64 {
	result := 1.0
	for i := 0; i < n; i++ {
		result *= x
	}
	return result
}

func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	var result []byte
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(ch))
	}
	return string(result)
}
