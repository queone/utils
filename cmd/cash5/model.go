package main

import "encoding/json"

type APIResponse struct {
	Draws []Draw `json:"draws"`
}

type Draw struct {
	GameName         string      `json:"gameName"`
	ID               string      `json:"id"`
	Status           string      `json:"status"`
	DrawTime         int64       `json:"drawTime"`
	EstimatedJackpot int64       `json:"estimatedJackpot"`
	Jackpot          int64       `json:"jackpot,omitempty"`
	ActualPayout     int64       `json:"actualPayout,omitempty"` // Manual field for actual 5/5 winner payout (in cents)
	Results          []Result    `json:"results"`
	PrizeTiers       []PrizeTier `json:"prizeTiers,omitempty"`
	Prizes           []Prize     `json:"prizes,omitempty"`
	WinningNumbers   interface{} `json:"winningNumbers,omitempty"`
}

// UnmarshalJSON custom unmarshaler to filter out empty prize tiers
func (d *Draw) UnmarshalJSON(data []byte) error {
	type Alias Draw
	aux := &struct {
		PrizeTiers []PrizeTier `json:"prizeTiers"`
		Prizes     []Prize     `json:"prizes"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Filter out empty prize tiers
	var filteredTiers []PrizeTier
	for _, tier := range aux.PrizeTiers {
		if tier.Tier != "" || tier.Winners > 0 || tier.PrizeAmount > 0 ||
			tier.Name != "" || tier.ShareCount > 0 || tier.ShareAmount > 0 {
			filteredTiers = append(filteredTiers, tier)
		}
	}
	d.PrizeTiers = filteredTiers

	// Filter out empty prizes
	var filteredPrizes []Prize
	for _, prize := range aux.Prizes {
		if prize.Level != "" || prize.Winners > 0 || prize.Amount > 0 {
			filteredPrizes = append(filteredPrizes, prize)
		}
	}
	d.Prizes = filteredPrizes

	return nil
}

type Result struct {
	Primary            []string `json:"primary"`
	PrimaryRevealOrder []string `json:"primaryRevealOrder,omitempty"`
	DrawType           string   `json:"drawType,omitempty"`
	Winners            int      `json:"winners,omitempty"`
	Payout             int64    `json:"payout,omitempty"`
	PrizeAmount        int64    `json:"prizeAmount,omitempty"`
}

type PrizeTier struct {
	Tier        string `json:"tier,omitempty"`
	Winners     int    `json:"winners,omitempty"`
	PrizeAmount int64  `json:"prizeAmount,omitempty"`
	Description string `json:"description,omitempty"`
	Match       string `json:"match,omitempty"`
	Prize       int64  `json:"prize,omitempty"`
	// Actual fields from NJ Lottery API
	ShareCount  int    `json:"shareCount,omitempty"`
	ShareAmount int64  `json:"shareAmount,omitempty"`
	Name        string `json:"name,omitempty"`
	ID          string `json:"id,omitempty"`
	PrizeType   string `json:"prizeType,omitempty"`
	DrawType    string `json:"drawType,omitempty"`
}

type Prize struct {
	Level       string `json:"level,omitempty"`
	Winners     int    `json:"winners,omitempty"`
	Amount      int64  `json:"amount,omitempty"`
	Description string `json:"description,omitempty"`
}
