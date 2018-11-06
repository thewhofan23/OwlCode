package main

import (
	s "strconv"
	"testing"
)

func TestRecordingQueryAndParseRecording(t *testing.T) {
	// Testing the recordingQuery
	expect1 := 9
	endTime := "1541168971265"
	duration := "3123000"
	test1, err := recordingQuery("212014918236538", endTime, duration)
	if err != nil {
		t.Errorf("Page err, %d", err)
	}
	if len(test1.Device.ObjectStat) != expect1 {
		t.Errorf("Test Query 1 did not work, got: %v, want: %v", len(test1.Device.ObjectStat), expect1)
	}
	expect2 := 5 // Camera On status
	test2 := test1.Device.ObjectStat[0].IntValue
	if test2 != expect2 {
		t.Errorf("Did not retrieve the proper camera status, got: %v, want: %v", test2, expect2)
	}
	// Testing the parseRecording
	endTimeInt, _ := s.Atoi(endTime)
	durationInt, _ := s.Atoi(duration)
	startTimeInt := endTimeInt - durationInt
	cameraRecordElements := parseRecording(test1, startTimeInt, endTimeInt)
	expect3 := 2458150
	test3 := cameraRecordElements.totalRecord
	// Testing the total recording time
	if cameraRecordElements.totalRecord != expect3 {
		t.Errorf("Did not get correct total recording time, got: %v, want: %v", test3, expect3)
	}

}

func TestSecToHours(t *testing.T) {
	// Test if zero value decodes correctly
	zero := 0
	expect1 := "0m 0s"
	time1 := secToHours(zero)
	if time1 != expect1 {
		t.Errorf("Zero second did not work, got: %s, want: %s", time1, expect1)
	}

	// Test if hour decodes correctly
	hour := 3600
	time2 := secToHours(hour)
	expect2 := "1h 0m"
	if time2 != expect2 {
		t.Errorf("Zero second did not work, got: %s, want: %s", time2, expect2)
	}

	// Test negative handling
	negative := -3600
	time3 := secToHours(negative)
	expect3 := "negative"
	if time3 != expect3 {
		t.Errorf("Zero second did not work, got: %s, want: %s", time3, expect3)
	}

}
