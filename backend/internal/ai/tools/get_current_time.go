package tools

import "time"

func GetCurrentTime() CurrentTimeOutput {
	now := time.Now()
	zone, _ := now.Zone()
	return CurrentTimeOutput{
		Now:      now,
		Date:     now.Format("2006-01-02"),
		TimeZone: zone,
	}
}
