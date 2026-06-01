package system

import "testing"

func TestGetCurrentTimeReturnsNowDateAndTimezone(t *testing.T) {
	output := GetCurrentTime()
	if output.Now.IsZero() || output.Date == "" || output.TimeZone == "" {
		t.Fatalf("unexpected current time: %#v", output)
	}
}
