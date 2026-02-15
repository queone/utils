package main

import "fmt"

func extractPrimaryFive(d *Draw) ([]int, error) {

	if len(d.Results) == 0 {
		return nil, fmt.Errorf("no results in draw")
	}

	raw := d.Results[0].Primary
	if len(raw) < 5 {
		return nil, fmt.Errorf("not enough primary numbers")
	}

	numbers := make([]int, 0, 5)

	for i := 0; i < 5; i++ {
		var n int
		_, err := fmt.Sscanf(raw[i], "%d", &n)
		if err != nil {
			return nil, err
		}
		numbers = append(numbers, n)
	}

	return numbers, nil
}

// getPayout returns the actual 5/5 payout amount in cents, or 0 if no winner
func getPayout(currentDraw *Draw) int64 {
	// First check if ActualPayout is set (manual winner data)
	if currentDraw.ActualPayout > 0 {
		return currentDraw.ActualPayout
	}

	// Check prize tiers for actual payout
	for _, tier := range currentDraw.PrizeTiers {
		hasWinners := tier.Winners > 0 || tier.ShareCount > 0
		is5of5 := tier.Tier == "1" || tier.Match == "5" || tier.Match == "5/5" ||
			tier.Description == "5/5" || tier.Name == "5/5" || tier.ID == "1"

		if hasWinners && is5of5 {
			// Return shareAmount or prizeAmount, whichever is set
			if tier.ShareAmount > 0 {
				return tier.ShareAmount
			}
			if tier.PrizeAmount > 0 {
				return tier.PrizeAmount
			}
		}
	}

	return 0
}

func formatPayout(draws []Draw, currentDraw *Draw) string {
	payout := getPayout(currentDraw)
	if payout > 0 {
		dollars := payout / 100
		return formatCurrency(dollars)
	}
	return "$0"
}

func formatCurrency(amount int64) string {
	// Convert to string and add commas
	s := fmt.Sprintf("%d", amount)

	// Add commas every 3 digits from right
	var result []byte
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(ch))
	}

	return "$" + string(result)
}

func formatWinner(draws []Draw, currentDraw *Draw) string {
	return formatPayout(draws, currentDraw)
}
