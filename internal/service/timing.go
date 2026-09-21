package service

import "time"

// Standard markets must start >= 1 hour out; breaking markets
// (eventInProgress) skip that delay but endTime must still be future-dated.
// Confirmed at docs.panta.market/api-reference/markets/quote.
const minStartDelaySec = 3600
const safetyBufferSec = 30 // clock-skew cushion so we don't land exactly on the boundary

type MarketTimestamps struct {
	StartTime      int64
	EndTime        int64
	ResolutionTime int64
}

func ComputeMarketTimestamps(eventInProgress bool, durationHours, resolutionBufferHours float64) MarketTimestamps {
	now := time.Now().Unix()

	startTime := now + minStartDelaySec + safetyBufferSec
	if eventInProgress {
		startTime = now + safetyBufferSec
	}

	if durationHours < 0.25 {
		durationHours = 0.25
	}
	if resolutionBufferHours < 0.25 {
		resolutionBufferHours = 0.25
	}

	endTime := startTime + int64(durationHours*3600)
	resolutionTime := endTime + int64(resolutionBufferHours*3600)

	return MarketTimestamps{StartTime: startTime, EndTime: endTime, ResolutionTime: resolutionTime}
}
