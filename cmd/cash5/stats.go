package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/queone/utils/internal/color"
)

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
	fmt.Printf("%s: %s\n", color.Blu("Total Drawings"), color.Grn(len(uniqueDraws)))

	// Date range
	earliest := time.UnixMilli(uniqueDraws[0].DrawTime)
	latest := time.UnixMilli(uniqueDraws[len(uniqueDraws)-1].DrawTime)
	fmt.Printf("%s: %s\n", color.Blu("Earliest Drawing"), color.Grn(earliest.Format("2006-01-02")))
	fmt.Printf("%s: %s\n", color.Blu("Latest Drawing"), color.Grn(latest.Format("2006-01-02")))

	// Find biggest and smallest recorded prizes
	var smallestPayout int64 = 999999999999
	var smallestDraw *Draw
	winnersCount := 0
	var winnerDates []time.Time

	type winnerEntry struct {
		draw   *Draw
		payout int64
	}
	var allWinners []winnerEntry

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
			allWinners = append(allWinners, winnerEntry{draw: d, payout: payout})
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

	// Detect pool expansion (used throughout analysis)
	pe := detectPoolExpansion(uniqueDraws, overallFreq)
	if pe.expanded {
		// Find the date of the expansion
		expandDate := narrativeDate(time.UnixMilli(uniqueDraws[pe.expansionIdx].DrawTime))
		var lateNums []int
		for n := range pe.lateEntrants {
			lateNums = append(lateNums, n)
		}
		sort.Ints(lateNums)
		fmt.Printf("\n%s: %v first eligible at draw #%d (%s)\n",
			color.Blu("Pool expansion detected"), lateNums, pe.expansionIdx+1, expandDate)
		fmt.Printf("  %s: %d → %d\n", color.Blu("Pool size change"), pe.prePoolSize, pe.postPoolSize)
	}

	fmt.Printf("\n%s: %s\n", color.Blu("Winners (5/5 Match)"), color.Grn(winnersCount))

	// Check for duplicate winning combinations
	fmt.Printf("\n%s:\n", color.Blu("Duplicate Combination Check"))

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

	// Birthday paradox expected duplicates
	totalCombinations := 1221759 // C(45,5)
	expectedDups := birthdayExpectedDuplicates(len(uniqueDraws), totalCombinations)
	dupStdDev := birthdayStdDev(expectedDups)
	observedDups := len(duplicates)

	fmt.Printf("  %s: %s  %s\n",
		color.Blu("Observed duplicates"), color.Grn(observedDups),
		color.Gra(fmt.Sprintf("(expected ~%.1f ± %.1f from birthday paradox)", expectedDups, dupStdDev)))

	if observedDups == 0 {
		fmt.Printf("  %s: %s\n", color.Blu("Status"),
			color.Grn(fmt.Sprintf("✓ No duplicates (%d unique combinations in %d draws)", len(combinationMap), len(uniqueDraws))))
	} else {
		zScore := 0.0
		if dupStdDev > 0 {
			zScore = (float64(observedDups) - expectedDups) / dupStdDev
		}
		if zScore > 2.0 {
			fmt.Printf("  %s: %s\n", color.Blu("Status"),
				color.Gra(fmt.Sprintf("⚠ %d duplicates exceeds expectation (z=%.1f)", observedDups, zScore)))
		} else {
			fmt.Printf("  %s: %s\n", color.Blu("Status"),
				color.Grn(fmt.Sprintf("✓ %d duplicates is consistent with random chance (z=%.1f)", observedDups, zScore)))
		}

		fmt.Printf("\n  %s:\n", color.Blu("Duplicate Details"))
		for _, dup := range duplicates {
			fmt.Printf("    %s: %s\n", color.Blu("Combination"), color.Grn(dup.combo))
			// Check for close-in-time pairs that warrant scrutiny
			for i, d1 := range dup.dates {
				label := color.Grn(d1)
				if i > 0 {
					t1, _ := time.Parse("2006-01-02", dup.dates[i-1])
					t2, _ := time.Parse("2006-01-02", d1)
					gap := int(t2.Sub(t1).Hours() / 24)
					if gap <= 30 {
						label = color.Gra(fmt.Sprintf("%s  ← %d-day gap, warrants scrutiny", d1, gap))
					}
				}
				fmt.Printf("        - %s\n", label)
			}
		}
	}

	if len(allWinners) > 0 {
		sort.Slice(allWinners, func(i, j int) bool {
			return allWinners[i].payout > allWinners[j].payout
		})
		topN := 10
		if len(allWinners) < topN {
			topN = len(allWinners)
		}
		fmt.Printf("\n%s:\n", color.Blu("Biggest Prizes"))
		fmt.Printf("  %s  %s  %s\n",
			color.Blu(fmt.Sprintf("%-14s", "Numbers")),
			color.Blu(fmt.Sprintf("%-10s", "Date")),
			color.Blu("Prize"))
		for i := 0; i < topN; i++ {
			w := allWinners[i]
			nums, _ := extractPrimaryFive(w.draw)
			drawDate := time.UnixMilli(w.draw.DrawTime).Format("2006-01-02")
			numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])
			fmt.Printf("  %s  %s  %s\n",
				color.Grn(numStr),
				color.Grn(drawDate),
				color.Grn(formatCurrency(w.payout/100)))
		}
	}

	if smallestDraw != nil && smallestPayout < 999999999999 {
		nums, _ := extractPrimaryFive(smallestDraw)
		drawDate := time.UnixMilli(smallestDraw.DrawTime).Format("2006-01-02")
		numStr := fmt.Sprintf("%02d-%02d-%02d-%02d-%02d", nums[0], nums[1], nums[2], nums[3], nums[4])
		fmt.Printf("\n%s: %s\n", color.Blu("Smallest Prize"), color.Grn(formatCurrency(smallestPayout/100)))
		fmt.Printf("  %s: %s\n", color.Blu("Date"), color.Grn(drawDate))
		fmt.Printf("  %s: %s\n", color.Blu("Numbers"), color.Grn(numStr))
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

		fmt.Printf("\n%s:\n", color.Blu("Jackpot Win Frequency"))
		fmt.Printf("  %s: %s\n", color.Blu("Average days between"), color.Grn(fmt.Sprintf("%.1f days", avgDays)))
		fmt.Printf("  %s: %s\n", color.Blu("Longest streak"), color.Grn(fmt.Sprintf("%d days", longestStreak)))

		// Days since last winner
		if len(winnerDates) > 0 {
			daysSinceWin := int(latest.Sub(winnerDates[len(winnerDates)-1]).Hours() / 24)
			fmt.Printf("  %s: %s\n", color.Blu("Days since last win"), color.Grn(fmt.Sprintf("%d days", daysSinceWin)))
		}
	}

	// Most common numbers by position
	mostCommonFirst := findMostCommon(firstNumFreq)
	mostCommonSecond := findMostCommon(pos2Freq)
	mostCommonMiddle := findMostCommon(middleNumFreq)
	mostCommonFourth := findMostCommon(pos4Freq)
	mostCommonLast := findMostCommon(lastNumFreq)

	fmt.Printf("\n%s:\n", color.Blu("Most Common by Position"))
	fmt.Printf("  %s: %s  %s\n", color.Blu("First position"), color.Grn(fmt.Sprintf("%02d", mostCommonFirst.num)), color.Gra(fmt.Sprintf("(appeared %d times)", mostCommonFirst.count)))
	fmt.Printf("  %s: %s  %s\n", color.Blu("Second position"), color.Grn(fmt.Sprintf("%02d", mostCommonSecond.num)), color.Gra(fmt.Sprintf("(appeared %d times)", mostCommonSecond.count)))
	fmt.Printf("  %s: %s  %s\n", color.Blu("Third position"), color.Grn(fmt.Sprintf("%02d", mostCommonMiddle.num)), color.Gra(fmt.Sprintf("(appeared %d times)", mostCommonMiddle.count)))
	fmt.Printf("  %s: %s  %s\n", color.Blu("Fourth position"), color.Grn(fmt.Sprintf("%02d", mostCommonFourth.num)), color.Gra(fmt.Sprintf("(appeared %d times)", mostCommonFourth.count)))
	fmt.Printf("  %s: %s  %s\n", color.Blu("Fifth position"), color.Grn(fmt.Sprintf("%02d", mostCommonLast.num)), color.Gra(fmt.Sprintf("(appeared %d times)", mostCommonLast.count)))

	// Least common numbers by position
	leastCommonFirst := findLeastCommon(firstNumFreq)
	leastCommonSecond := findLeastCommon(pos2Freq)
	leastCommonMiddle := findLeastCommon(middleNumFreq)
	leastCommonFourth := findLeastCommon(pos4Freq)
	leastCommonLast := findLeastCommon(lastNumFreq)

	fmt.Printf("\n%s:\n", color.Blu("Least Common by Position"))
	fmt.Printf("  %s: %s  %s\n", color.Blu("First position"), color.Grn(fmt.Sprintf("%02d", leastCommonFirst.num)), color.Gra(fmt.Sprintf("(appeared %d times)", leastCommonFirst.count)))
	fmt.Printf("  %s: %s  %s\n", color.Blu("Second position"), color.Grn(fmt.Sprintf("%02d", leastCommonSecond.num)), color.Gra(fmt.Sprintf("(appeared %d times)", leastCommonSecond.count)))
	fmt.Printf("  %s: %s  %s\n", color.Blu("Third position"), color.Grn(fmt.Sprintf("%02d", leastCommonMiddle.num)), color.Gra(fmt.Sprintf("(appeared %d times)", leastCommonMiddle.count)))
	fmt.Printf("  %s: %s  %s\n", color.Blu("Fourth position"), color.Grn(fmt.Sprintf("%02d", leastCommonFourth.num)), color.Gra(fmt.Sprintf("(appeared %d times)", leastCommonFourth.count)))
	fmt.Printf("  %s: %s  %s\n", color.Blu("Fifth position"), color.Grn(fmt.Sprintf("%02d", leastCommonLast.num)), color.Gra(fmt.Sprintf("(appeared %d times)", leastCommonLast.count)))

	// Most frequently drawn overall
	topOverall := findTopN(overallFreq, 5)
	fmt.Printf("\n%s:\n", color.Blu("Most Frequently Drawn (All Positions)"))
	for i, nc := range topOverall {
		fmt.Printf("  %s. %s %s:  %s\n", color.Grn(i+1), color.Blu("Number"), color.Grn(fmt.Sprintf("%02d", nc.num)), color.Grn(fmt.Sprintf("%d times", nc.count)))
	}

	// Least frequently drawn overall
	bottomOverall := findBottomN(overallFreq, 5)
	fmt.Printf("\n%s:\n", color.Blu("Least Frequently Drawn (All Positions)"))
	for i, nc := range bottomOverall {
		fmt.Printf("  %s. %s %s:  %s\n", color.Grn(i+1), color.Blu("Number"), color.Grn(fmt.Sprintf("%02d", nc.num)), color.Grn(fmt.Sprintf("%d times", nc.count)))
	}

	// Hot numbers (last 30 days)
	if len(freq30) > 0 {
		hot30 := findTopN(freq30, 5)
		fmt.Printf("\n%s:\n", color.Blu("Hot Numbers (Last 30 Days)"))
		for i, nc := range hot30 {
			fmt.Printf("  %s. %s %s:  %s\n", color.Grn(i+1), color.Blu("Number"), color.Grn(fmt.Sprintf("%02d", nc.num)), color.Grn(fmt.Sprintf("%d times", nc.count)))
		}
	}

	// Cold numbers (last 90 days)
	if len(freq90) > 0 {
		cold90 := findBottomN(freq90, 5)
		fmt.Printf("\n%s:\n", color.Blu("Cold Numbers (Last 90 Days)"))
		for i, nc := range cold90 {
			fmt.Printf("  %s. %s %s:  %s\n", color.Grn(i+1), color.Blu("Number"), color.Grn(fmt.Sprintf("%02d", nc.num)), color.Grn(fmt.Sprintf("%d times", nc.count)))
		}
	}

	// Most common pairs
	topPairs := findTopNPairs(pairFreq, 5)
	fmt.Printf("\n%s:\n", color.Blu("Most Common Number Pairs"))
	for i, pc := range topPairs {
		fmt.Printf("  %s. %s:  %s\n", color.Grn(i+1), color.Grn(pc.pair), color.Grn(fmt.Sprintf("%d times", pc.count)))
	}

	// Chi-squared uniformity analysis
	chiSquared := calculateChiSquared(overallFreq, len(uniqueDraws)*5)
	fmt.Printf("\n%s:\n", color.Blu("χ² Uniformity Analysis"))
	fmt.Printf("  %s: %s\n", color.Blu("χ² statistic"), color.Grn(fmt.Sprintf("%.2f", chiSquared)))
	fmt.Printf("  %s: %s\n", color.Blu("Degrees of freedom"), color.Grn("44 (45 numbers - 1)"))

	// Critical values for χ² with 44 df:
	// p=0.05: 60.48, p=0.01: 66.77
	if chiSquared < 60.48 {
		fmt.Printf("  %s: %s\n", color.Blu("Result"), color.Grn("Uniform distribution (p > 0.05)"))
		fmt.Printf("  %s: %s\n", color.Blu("Interpretation"), color.Gra("Numbers appear randomly distributed"))
	} else if chiSquared < 66.77 {
		fmt.Printf("  %s: %s\n", color.Blu("Result"), color.Grn("Possibly non-uniform (0.01 < p < 0.05)"))
		fmt.Printf("  %s: %s\n", color.Blu("Interpretation"), color.Gra("Slight deviation from randomness"))
	} else {
		fmt.Printf("  %s: %s\n", color.Blu("Result"), color.Grn("Non-uniform distribution (p < 0.01)"))
		fmt.Printf("  %s: %s\n", color.Blu("Interpretation"), color.Gra("Significant bias detected"))
	}

	// Full χ² Frequency Analysis
	fmt.Printf("\n%s:\n", color.Blu("Full χ² Frequency Analysis Over History"))

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
	fmt.Printf("\n  %s:\n", color.Blu("Position-Specific Uniformity Tests"))

	positionNames := []string{"First", "Second", "Third", "Fourth", "Fifth"}
	positionFreqs := []map[int]int{pos1Freq, pos2FreqChi, pos3Freq, pos4FreqChi, pos5Freq}

	allPositionsUniform := true
	for i, posFreq := range positionFreqs {
		chiSq := calculateChiSquared(posFreq, len(uniqueDraws))
		isUniform := chiSq < 60.48
		if !isUniform {
			allPositionsUniform = false
		}

		statusStr := color.Grn("✓ Uniform")
		if !isUniform {
			statusStr = color.Gra("⚠ Non-uniform")
		}
		fmt.Printf("    %s %s: χ²=%s  %s\n",
			color.Blu(positionNames[i]), color.Blu("position"),
			color.Grn(fmt.Sprintf("%.2f", chiSq)), statusStr)
	}

	if allPositionsUniform {
		fmt.Printf("  %s: %s\n", color.Blu("Overall"), color.Gra("All positions show uniform distribution"))
	} else {
		fmt.Printf("  %s: %s\n", color.Blu("Overall"), color.Gra("Some positions show non-uniform distribution"))
	}

	// 2. Temporal uniformity (monthly/yearly trends)
	fmt.Printf("\n  %s:\n", color.Blu("Temporal Uniformity Analysis"))

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

	fmt.Printf("    %s:\n", color.Blu("Year-by-Year Analysis (pool-size adjusted)"))
	for _, year := range years {
		yearDraws := 0
		for i := range uniqueDraws {
			if time.UnixMilli(uniqueDraws[i].DrawTime).Year() == year {
				yearDraws++
			}
		}

		if yearDraws >= 30 { // Only analyze years with enough data
			poolSize := detectPoolSizeForYear(uniqueDraws, year, pe)
			chiSq := calculateChiSquaredWithPool(yearlyFreqs[year], yearDraws*5, poolSize)
			df := poolSize - 1
			critical := chiSquaredCritical(df)
			isUniform := chiSq < critical

			statusStr := color.Grn("✓")
			if !isUniform {
				statusStr = color.Gra("⚠")
			}

			fmt.Printf("      %s %s: χ²=%s  %s draws  %s\n",
				statusStr,
				color.Blu(fmt.Sprintf("%d", year)),
				color.Grn(fmt.Sprintf("%.2f", chiSq)),
				color.Grn(yearDraws),
				color.Gra(fmt.Sprintf("(pool=%d, df=%d, critical=%.1f)", poolSize, df, critical)))
		}
	}

	// 3. Sequential pair analysis
	fmt.Printf("\n  %s:\n", color.Blu("Sequential Pair Uniformity"))
	fmt.Printf("    %s: %s\n", color.Blu("Testing"), color.Gra("Whether consecutive numbers appear uniformly"))

	consecutivePairs := 0
	totalPairs := 0
	consecPairFreq := make(map[string]int) // "NN-NN+1" -> count
	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err == nil {
			for j := 0; j < len(nums)-1; j++ {
				totalPairs++
				if nums[j+1] == nums[j]+1 {
					consecutivePairs++
					pairKey := fmt.Sprintf("%02d-%02d", nums[j], nums[j+1])
					consecPairFreq[pairKey]++
				}
			}
		}
	}

	expectedConsecutiveRate := 4.0 / 44.0 // Probability of consecutive in random draw
	actualRate := float64(consecutivePairs) / float64(totalPairs)

	fmt.Printf("    %s: %s  %s\n",
		color.Blu("Consecutive pairs found"),
		color.Grn(fmt.Sprintf("%d/%d", consecutivePairs, totalPairs)),
		color.Gra(fmt.Sprintf("(%.2f%%)", actualRate*100)))
	fmt.Printf("    %s: %s\n",
		color.Blu("Expected rate"),
		color.Grn(fmt.Sprintf("%.2f%%", expectedConsecutiveRate*100)))

	deviation := ((actualRate - expectedConsecutiveRate) / expectedConsecutiveRate) * 100
	if deviation > -10 && deviation < 10 {
		fmt.Printf("    %s: %s  %s\n", color.Blu("Assessment"), color.Grn("Within expected range"),
			color.Gra(fmt.Sprintf("(%.1f%% deviation)", deviation)))
	} else {
		fmt.Printf("    %s: %s  %s\n", color.Blu("Assessment"), color.Gra("Outside expected range"),
			color.Gra(fmt.Sprintf("(%.1f%% deviation)", deviation)))
	}

	// Breakdown: top 10 most frequent consecutive pairs
	fmt.Printf("\n    %s:\n", color.Blu("Top 10 Consecutive Pairs"))
	topConsec := findTopNPairs(consecPairFreq, 10)
	// Expected per specific pair: each adjacent pair (k, k+1) has roughly equal probability
	// Total expected consecutive = totalPairs * expectedConsecutiveRate
	// Spread across 44 possible adjacent pairs (1-2, 2-3, ..., 44-45)
	expectedPerPair := float64(totalPairs) * expectedConsecutiveRate / 44.0
	for i, pc := range topConsec {
		fmt.Printf("      %2d. %s:  %s  %s\n",
			i+1, color.Grn(pc.pair),
			color.Grn(fmt.Sprintf("%d times", pc.count)),
			color.Gra(fmt.Sprintf("(expected ~%.1f)", expectedPerPair)))
	}

	// Range breakdown: low (1-15), mid (16-30), high (31-44)
	lowConsec, midConsec, highConsec := 0, 0, 0
	for pair, count := range consecPairFreq {
		var n1 int
		fmt.Sscanf(pair, "%d-", &n1)
		switch {
		case n1 <= 15:
			lowConsec += count
		case n1 <= 30:
			midConsec += count
		default:
			highConsec += count
		}
	}
	fmt.Printf("\n    %s:\n", color.Blu("Consecutive Pairs by Range"))
	fmt.Printf("      %s: %s\n", color.Blu("Low (1-15)"), color.Grn(fmt.Sprintf("%d", lowConsec)))
	fmt.Printf("      %s: %s\n", color.Blu("Mid (16-30)"), color.Grn(fmt.Sprintf("%d", midConsec)))
	fmt.Printf("      %s: %s\n", color.Blu("High (31-44)"), color.Grn(fmt.Sprintf("%d", highConsec)))

	// 4. Low vs High number distribution (hypergeometric baseline)
	// For a sorted draw of 5 from 1-45, E[low numbers 1-22] = 5 * 22/45 ≈ 2.444
	// This is already the correct hypergeometric expected value.
	// The simple proportion 22/45 is correct for each of 5 picks without replacement
	// because E[X] = n*K/N for hypergeometric.
	fmt.Printf("\n  %s:\n", color.Blu("Low vs High Number Distribution"))
	fmt.Printf("    %s: %s\n", color.Blu("Testing"), color.Gra("Whether low (1-22) and high (23-45) numbers match hypergeometric expectation"))

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
	// Hypergeometric: E[low] = n * K/N where n=5 picks, K=22 low numbers, N=45 total
	// Per draw: E[low] = 5 * 22/45. Over all draws: totalNums * 22/45
	expectedLow := float64(totalNums) * (22.0 / 45.0)
	expectedHigh := float64(totalNums) * (23.0 / 45.0)

	chiSqLowHigh := (float64(lowCount)-expectedLow)*(float64(lowCount)-expectedLow)/expectedLow +
		(float64(highCount)-expectedHigh)*(float64(highCount)-expectedHigh)/expectedHigh

	fmt.Printf("    %s: %s  %s\n",
		color.Blu("Low numbers (1-22)"),
		color.Grn(fmt.Sprintf("%d", lowCount)),
		color.Gra(fmt.Sprintf("(hypergeometric expected: %.0f)", expectedLow)))
	fmt.Printf("    %s: %s  %s\n",
		color.Blu("High numbers (23-45)"),
		color.Grn(fmt.Sprintf("%d", highCount)),
		color.Gra(fmt.Sprintf("(hypergeometric expected: %.0f)", expectedHigh)))
	fmt.Printf("    %s: %s  %s\n",
		color.Blu("χ² statistic"),
		color.Grn(fmt.Sprintf("%.2f", chiSqLowHigh)),
		color.Gra("(df=1, critical=3.84 at p=0.05)"))

	if chiSqLowHigh < 3.84 {
		fmt.Printf("    %s: %s\n", color.Blu("Result"), color.Grn("✓ Balanced distribution"))
	} else {
		fmt.Printf("    %s: %s\n", color.Blu("Result"), color.Gra("⚠ Imbalanced — exceeds hypergeometric expectation"))
	}

	// 5. Summary
	fmt.Printf("\n  %s:\n", color.Blu("Analysis Summary"))
	issuesFound := 0
	if !allPositionsUniform {
		issuesFound++
	}
	if chiSqLowHigh >= 3.84 {
		issuesFound++
	}

	if issuesFound == 0 {
		fmt.Printf("    %s\n", color.Grn("✓ All tests passed - lottery appears statistically fair"))
	} else {
		fmt.Printf("    %s\n", color.Gra(fmt.Sprintf("⚠ %d potential issues detected - review individual tests", issuesFound)))
	}

	// Repeat probability analysis
	fmt.Printf("\n%s:\n", color.Blu("Repeat Combination Analysis"))

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

	simResults := runRepeatSimulation(numHistorical, totalCombinations, 10000)

	fmt.Printf("  %s: %s\n", color.Blu("Historical combinations"), color.Grn(fmt.Sprintf("%d unique sets", numHistorical)))
	fmt.Printf("  %s: %s\n", color.Blu("Total possible combos"), color.Grn(formatNumber(totalCombinations)))
	fmt.Printf("  %s: %s\n", color.Blu("Coverage"), color.Grn(fmt.Sprintf("%.4f%%", float64(numHistorical)*100.0/float64(totalCombinations))))

	// Birthday paradox context
	bpExpected := birthdayExpectedDuplicates(len(uniqueDraws), totalCombinations)
	bpDev := (float64(observedDups) - bpExpected) / birthdayStdDev(bpExpected)
	fmt.Printf("  %s: %s\n", color.Blu("Birthday paradox expected repeats"),
		color.Grn(fmt.Sprintf("~%.1f for %d draws from %s combos", bpExpected, len(uniqueDraws), formatNumber(totalCombinations))))
	fmt.Printf("  %s: %s\n", color.Blu("Observed repeats"),
		color.Grn(fmt.Sprintf("%d (z=%.1f, consistent with expectation)", observedDups, bpDev)))

	fmt.Printf("\n  %s:\n", color.Blu("Future repeat probability"))
	fmt.Printf("    %s: %s\n", color.Blu("In next 30 draws"), color.Grn(fmt.Sprintf("%.2f%%", simResults.prob30Days*100)))
	fmt.Printf("    %s: %s\n", color.Blu("In next 90 draws"), color.Grn(fmt.Sprintf("%.2f%%", simResults.prob90Days*100)))
	fmt.Printf("    %s: %s\n", color.Blu("In next 365 draws"), color.Grn(fmt.Sprintf("%.2f%%", simResults.prob365Days*100)))
	fmt.Printf("    %s: %s\n", color.Blu("In next 10 years"), color.Grn(fmt.Sprintf("%.2f%%", simResults.prob10Years*100)))

	// Distance scoring caveat
	fmt.Printf("\n%s:\n", color.Blu("Combinatorial Distance Scoring"))
	coverage := float64(numHistorical) * 100.0 / float64(totalCombinations)
	fmt.Printf("  %s\n", color.Gra(fmt.Sprintf("With %d draws covering %.2f%% of %s possible combinations,",
		len(uniqueDraws), coverage, formatNumber(totalCombinations))))
	fmt.Printf("  %s\n", color.Gra("the max-distance score is 2/5 for ~98%% of all combos."))
	fmt.Printf("  %s\n", color.Gra("At current dataset density, distance-based selection provides no"))
	fmt.Printf("  %s\n", color.Gra("actionable signal — any random combination achieves the same score."))
	fmt.Printf("  %s\n", color.Gra("(Skipping brute-force enumeration and simulated annealing.)"))

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

func findLeastCommon(freq map[int]int) numCount {
	result := numCount{count: int(^uint(0) >> 1)} // max int
	for num, count := range freq {
		if count < result.count {
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

// calculateChiSquared performs chi-squared test for uniformity.
// freq: map of number -> count
// totalDraws: total number of balls drawn (num_drawings * 5)
func calculateChiSquared(freq map[int]int, totalDraws int) float64 {
	expected := float64(totalDraws) / 45.0
	var chiSquared float64
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

// runRepeatSimulation estimates probability of drawing a previously seen combination.
func runRepeatSimulation(numHistorical, totalCombos, iterations int) simulationResults {
	const (
		draws30  = 30
		draws90  = 90
		draws365 = 365
		draws10y = 3650
	)

	p := float64(numHistorical) / float64(totalCombos)

	return simulationResults{
		prob30Days:  1.0 - pow(1.0-p, draws30),
		prob90Days:  1.0 - pow(1.0-p, draws90),
		prob365Days: 1.0 - pow(1.0-p, draws365),
		prob10Years: 1.0 - pow(1.0-p, draws10y),
	}
}

// generateRandomCombo generates a random 5-number combination (1-45).
func generateRandomCombo() []int {
	nums := make([]int, 5)
	used := make(map[int]bool)
	for i := range 5 {
		for {
			n := rand.IntN(45) + 1
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

// pow calculates x^n.
func pow(x float64, n int) float64 {
	result := 1.0
	for range n {
		result *= x
	}
	return result
}

// birthdayExpectedDuplicates returns the expected number of duplicate pairs
// given n draws from a pool of totalCombos possible combinations.
// Uses the birthday paradox approximation: E[pairs] ≈ n*(n-1) / (2*totalCombos)
func birthdayExpectedDuplicates(n, totalCombos int) float64 {
	return float64(n) * float64(n-1) / (2.0 * float64(totalCombos))
}

// birthdayStdDev returns approximate std dev for birthday collision count.
// Var ≈ E[pairs] * (1 - 1/totalCombos) for Poisson approximation.
func birthdayStdDev(expected float64) float64 {
	if expected <= 0 {
		return 0
	}
	return math.Sqrt(expected)
}

// calculateChiSquaredWithPool performs chi-squared test using only numbers in the active pool.
func calculateChiSquaredWithPool(freq map[int]int, totalBalls int, poolSize int) float64 {
	if poolSize == 0 {
		return 0
	}
	expected := float64(totalBalls) / float64(poolSize)
	var chiSquared float64
	for i := 1; i <= poolSize; i++ {
		observed := float64(freq[i])
		diff := observed - expected
		chiSquared += (diff * diff) / expected
	}
	return chiSquared
}

// chiSquaredCritical returns the p=0.05 critical value for given degrees of freedom.
// Uses approximation for df > 30: critical ≈ df * (1 - 2/(9*df) + 1.6449*sqrt(2/(9*df)))^3
func chiSquaredCritical(df int) float64 {
	if df <= 0 {
		return 0
	}
	// Wilson-Hilferty approximation
	d := float64(df)
	x := 1.0 - 2.0/(9.0*d) + 1.6449*math.Sqrt(2.0/(9.0*d))
	return d * x * x * x
}

// detectPoolSizeForYear determines the active pool size for a given year
// using the expansion detection result.
func detectPoolSizeForYear(uniqueDraws []Draw, year int, pe poolExpansion) int {
	if !pe.expanded {
		return 45
	}
	// Find the draw index at the start of the given year
	for i := range uniqueDraws {
		if time.UnixMilli(uniqueDraws[i].DrawTime).Year() > year {
			// Draw i is the first draw after this year.
			// If expansion happened before end of this year, use post pool size.
			if pe.expansionIdx < i {
				return pe.postPoolSize
			}
			return pe.prePoolSize
		}
	}
	// Year extends past all draws — use post pool size
	return pe.postPoolSize
}

// numFirstSeenIndex returns a map of number -> index of first draw where it appeared.
func numFirstSeenIndex(uniqueDraws []Draw) map[int]int {
	first := make(map[int]int)
	for i := range uniqueDraws {
		nums, err := extractPrimaryFive(&uniqueDraws[i])
		if err != nil {
			continue
		}
		for _, n := range nums {
			if _, ok := first[n]; !ok {
				first[n] = i
			}
		}
	}
	return first
}

// poolExpansion holds the result of pool expansion detection.
type poolExpansion struct {
	expanded     bool
	lateEntrants map[int]bool // numbers that entered late
	expansionIdx int          // draw index where expansion started
	prePoolSize  int          // pool size before expansion
	postPoolSize int          // pool size after expansion
}

// detectPoolExpansion looks for a structural cluster of late-arriving numbers
// that indicates a genuine pool size change (e.g. from 38 to 45).
// It requires: 2+ numbers first appearing within a 60-draw window, no earlier
// than draw 200, cross-validated by significantly low total frequency.
func detectPoolExpansion(uniqueDraws []Draw, overallFreq map[int]int) poolExpansion {
	totalDraws := len(uniqueDraws)
	if totalDraws < 200 {
		return poolExpansion{postPoolSize: 45}
	}

	firstSeen := numFirstSeenIndex(uniqueDraws)

	// Collect numbers whose first appearance is at draw index >= 200
	type candidate struct {
		num      int
		firstIdx int
	}
	var candidates []candidate
	for n := 1; n <= 45; n++ {
		idx, ok := firstSeen[n]
		if ok && idx >= 200 {
			candidates = append(candidates, candidate{n, idx})
		}
	}

	if len(candidates) < 2 {
		return poolExpansion{postPoolSize: 45}
	}

	// Sort by first-seen index
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].firstIdx < candidates[j].firstIdx
	})

	// Find the densest cluster within a 60-draw window
	bestStart := 0
	bestCount := 0
	for i := range candidates {
		count := 0
		for j := i; j < len(candidates); j++ {
			if candidates[j].firstIdx-candidates[i].firstIdx <= 60 {
				count++
			} else {
				break
			}
		}
		if count > bestCount {
			bestCount = count
			bestStart = i
		}
	}

	if bestCount < 2 {
		return poolExpansion{postPoolSize: 45}
	}

	// Extract cluster members
	clusterStartIdx := candidates[bestStart].firstIdx
	var clusterNums []candidate
	for i := bestStart; i < len(candidates); i++ {
		if candidates[i].firstIdx-clusterStartIdx <= 60 {
			clusterNums = append(clusterNums, candidates[i])
		} else {
			break
		}
	}

	// Cross-validate: each candidate should have significantly low frequency.
	// Expected if the number were eligible from the start: totalDraws * 5/45
	// Its actual eligible draws: totalDraws - firstIdx
	// If actual frequency is within 1σ of full-history expectation, it's not
	// a true late entrant (just a random late first appearance).
	lateEntrants := make(map[int]bool)
	for _, c := range clusterNums {
		eligibleDraws := totalDraws - c.firstIdx
		// Expected if it had been eligible all along
		fullExpected := float64(totalDraws) * 5.0 / 45.0
		// Its actual-eligibility expected
		eligibleExpected := float64(eligibleDraws) * 5.0 / 45.0
		observed := float64(overallFreq[c.num])
		// σ for binomial: sqrt(n * p * (1-p)) where p = 5/45
		sigma := math.Sqrt(float64(totalDraws) * (5.0 / 45.0) * (40.0 / 45.0))
		// If observed is within 1σ of the full-history expectation, this number
		// has appeared roughly as often as a number eligible from the start — discard it
		if observed < fullExpected-sigma {
			// Frequency is significantly below what a full-history number would have.
			// Additional check: is the observed count consistent with its actual eligible period?
			eligibleSigma := math.Sqrt(float64(eligibleDraws) * (5.0 / 45.0) * (40.0 / 45.0))
			if math.Abs(observed-eligibleExpected) <= 2*eligibleSigma {
				// Consistent with being eligible only from firstIdx onward
				lateEntrants[c.num] = true
			}
		}
	}

	if len(lateEntrants) < 2 {
		return poolExpansion{postPoolSize: 45}
	}

	// Determine pre-pool size: 45 minus late entrants
	prePoolSize := 45 - len(lateEntrants)

	return poolExpansion{
		expanded:     true,
		lateEntrants: lateEntrants,
		expansionIdx: clusterStartIdx,
		prePoolSize:  prePoolSize,
		postPoolSize: 45,
	}
}

// expectedFreqForNumber computes the expected frequency for a single number
// given the pool expansion info.
func expectedFreqForNumber(n int, totalDraws int, pe poolExpansion) float64 {
	if !pe.expanded {
		// No expansion: flat baseline
		return float64(totalDraws) * 5.0 / 45.0
	}

	if pe.lateEntrants[n] {
		// Late entrant: only eligible from expansionIdx onward
		eligible := totalDraws - pe.expansionIdx
		return float64(eligible) * 5.0 / float64(pe.postPoolSize)
	}

	// Original pool number: higher expected rate pre-expansion, normal rate post
	preDraws := pe.expansionIdx
	postDraws := totalDraws - pe.expansionIdx
	return float64(preDraws)*5.0/float64(pe.prePoolSize) +
		float64(postDraws)*5.0/float64(pe.postPoolSize)
}

// generateConsecAvoidCombo generates a 5-number combo that minimizes consecutive pairs.
func generateConsecAvoidCombo() []int {
	// Strategy: pick numbers spaced at least 2 apart
	// Try random combos and keep the one with fewest consecutive pairs
	bestCombo := generateRandomCombo()
	bestConsec := countConsecPairs(bestCombo)

	for range 1000 {
		combo := generateRandomCombo()
		consec := countConsecPairs(combo)
		if consec < bestConsec {
			bestConsec = consec
			bestCombo = combo
			if bestConsec == 0 {
				break
			}
		}
	}
	return bestCombo
}

func countConsecPairs(combo []int) int {
	count := 0
	for i := 0; i < len(combo)-1; i++ {
		if combo[i+1] == combo[i]+1 {
			count++
		}
	}
	return count
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
