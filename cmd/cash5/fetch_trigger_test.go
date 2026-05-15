package main

import (
	"testing"
	"time"
)

// edt is the New Jersey timezone the cash5 operator typically runs in.
// Used to exercise the regression case where stored drawTime is at local
// midnight (which encodes as a UTC instant 4 hours later) and the operator
// runs cash5 the next local calendar day.
var edt = time.FixedZone("EDT", -4*60*60)

// localMidnightMillis builds a UnixMilli for the start of the given calendar
// date in loc. NJ Cash 5 stores drawTimes at NJ-local midnight of the draw
// date, encoded as the corresponding UTC instant. For example, EDT midnight
// on 2026-05-13 is 2026-05-13 04:00 UTC.
func localMidnightMillis(y int, m time.Month, d int, loc *time.Location) int64 {
	return time.Date(y, m, d, 0, 0, 0, 0, loc).UnixMilli()
}

func TestNeedsRecentFetchTodaysDraw(t *testing.T) {
	// Newest cached draw is dated today (local). The bug regression case:
	// operator runs cash5 in the evening, after local midnight has carried
	// time.Now() past UTC midnight too. No fetch should fire.
	now := time.Date(2026, 5, 13, 20, 47, 0, 0, edt)
	got, newest, _ := needsRecentFetch(localMidnightMillis(2026, 5, 13, edt), now)
	if got {
		t.Errorf("needsRecentFetch fired on today's draw")
	}
	if newest.Day() != 13 || newest.Month() != 5 {
		t.Errorf("newest in local TZ wrong: %v", newest)
	}
}

func TestNeedsRecentFetchYesterdaysDraw(t *testing.T) {
	// Newest draw is local-yesterday; today's draw not yet published.
	now := time.Date(2026, 5, 14, 14, 0, 0, 0, edt)
	got, _, _ := needsRecentFetch(localMidnightMillis(2026, 5, 13, edt), now)
	if got {
		t.Errorf("needsRecentFetch fired on local-yesterday newest")
	}
}

func TestNeedsRecentFetchTwoDaysOldFires(t *testing.T) {
	// Newest draw was 2 local days ago — a real miss.
	now := time.Date(2026, 5, 15, 14, 0, 0, 0, edt)
	got, newest, yesterday := needsRecentFetch(localMidnightMillis(2026, 5, 13, edt), now)
	if !got {
		t.Errorf("needsRecentFetch did not fire on 2-days-old newest")
	}
	if newest.Location() != edt {
		t.Errorf("newest not in operator's local TZ: %v", newest.Location())
	}
	if yesterday.Location() != edt {
		t.Errorf("yesterday not in operator's local TZ: %v", yesterday.Location())
	}
	if yesterday.Year() != 2026 || yesterday.Month() != 5 || yesterday.Day() != 14 {
		t.Errorf("yesterday wrong: %v", yesterday)
	}
}

func TestNeedsRecentFetchHonorsOperatorTZWest(t *testing.T) {
	// PDT (UTC-7): a drawTime stored at PDT-midnight encodes 2026-05-13 07:00
	// UTC. "now" later that same local day must not fire.
	pdt := time.FixedZone("PDT", -7*60*60)
	now := time.Date(2026, 5, 13, 9, 0, 0, 0, pdt)
	got, _, _ := needsRecentFetch(localMidnightMillis(2026, 5, 13, pdt), now)
	if got {
		t.Errorf("needsRecentFetch fired for west-of-UTC operator on same-local-day newest")
	}
}

func TestNeedsRecentFetchHonorsOperatorTZEast(t *testing.T) {
	// CEST (UTC+2): drawTime at CEST midnight encodes 2026-05-12 22:00 UTC.
	// Same-local-day "now" → no fetch.
	cest := time.FixedZone("CEST", 2*60*60)
	now := time.Date(2026, 5, 13, 10, 0, 0, 0, cest)
	got, _, _ := needsRecentFetch(localMidnightMillis(2026, 5, 13, cest), now)
	if got {
		t.Errorf("needsRecentFetch fired for east-of-UTC operator on same-local-day newest")
	}
}
