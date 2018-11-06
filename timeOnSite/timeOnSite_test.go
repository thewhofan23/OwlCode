package main

import (
	"testing"
)

// TestSecToHours Test the time formatting
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

func TestGreatCircleDist(t *testing.T) {
	// Calculate distance from SF to Paris. Switches from postive to negative longitude
	// Paris
	var lat1, long1 float32 = 48.8566, 2.349014
	// SF
	var lat2, long2 float32 = 37.733795, -122.446747

	result1 := greatCircleDist(lat1, long1, lat2, long2)
	var dist1 float32 = 8958379.0
	if result1 != dist1 {
		t.Errorf("Paris to SF distance did not work, got: %f, want: %f", result1, dist1)
	}

	// Calculate distance from Sioux Falls to Sioux City. Common region for freight travel
	// Sioux Falls
	var lat3, long3 float32 = 43.5445959, -96.7311034
	// Sioux City
	var lat4, long4 float32 = 42.4921646, -96.3908317

	result2 := greatCircleDist(lat3, long3, lat4, long4)
	var dist2 float32 = 120250.125
	if result2 != dist2 {
		t.Errorf("Sioux Falls to Sioux City distance did not work, got: %f, want: %f", result2, dist2)
	}

	// Calculate distance from Bogota, Colombia to Santiago, Chile. Switches from positive to negative latitude
	// Bogota, Colombia
	var lat5, long5 float32 = 4.624335, -74.063644
	// Santiago, Chile
	var lat6, long6 float32 = -33.45694, -70.64827

	result3 := greatCircleDist(lat5, long5, lat6, long6)
	var dist3 float32 = 4249678.0
	if result3 != dist3 {
		t.Errorf("Sioux Falls to Sioux City distance did not work, got: %f, want: %f", result3, dist3)
	}

}

func TestSiteQuery(t *testing.T) {
	// Testing with my own org
	groupID := "4656"
	expected1 := 4
	result, err := siteQuery(groupID)
	if err != nil {
		t.Errorf("Received an error: %s", err)
	}
	if len(result.Group.Sites) != expected1 {
		t.Errorf("Did not return correct numbers of sites, got: %d, want: %d", len(result.Group.Sites), expected1)
	}
}

func TestGetGPSBound(t *testing.T) {
	// Test when valid lat, long with a radius of 500m
	var lat1, long1, r1 float32 = 37.733795, -122.446747, 500
	result1, err1 := getGPSBound(lat1, long1, r1)
	expected1 := latLongRange{-122.45574, -122.43775, 37.7248, 37.74279}
	if err1 != nil {
		t.Errorf("Received an error: %s", err1)
	}
	if result1 != expected1 {
		t.Errorf("Did not get correct bounding box, got: %v, want: %v", result1, expected1)
	}

	// Test when radius is negative
	var lat2, long2, r2 float32 = 37.733795, -122.446747, -500
	result2, err2 := getGPSBound(lat2, long2, r2)
	expected2 := latLongRange{-122.45574, -122.43775, 37.7248, 37.74279}
	if err2 != nil {
		t.Errorf("Received an error: %s", err2)
	}
	if result2 != expected2 {
		t.Errorf("Negative radius broke function, got: %v, want: %v", result2, expected2)
	}

	// Test when radius is zero
	var lat3, long3, r3 float32 = 37.733795, -122.446747, 0
	result3, err3 := getGPSBound(lat3, long3, r3)
	expected3 := latLongRange{-122.446747, -122.446747, 37.733795, 37.733795}
	if err3 != nil {
		t.Errorf("Received an error: %s", err3)
	}
	if result3 != expected3 {
		t.Errorf("Zero radius broke function, got: %v, want: %v", result3, expected3)
	}

	// Test when bounds overlap over longitude range
	var lat4, long4, r4 float32 = 37.733795, -179.999, 1000
	result4, err4 := getGPSBound(lat4, long4, r4)
	expected4 := latLongRange{-180, 180, 37.71581, 37.75178}
	if err4 != nil {
		t.Errorf("Received an error: %s", err4)
	}
	if result4 != expected4 {
		t.Errorf("Overlap at longitude unexpected value, got: %v, want: %v", result4, expected4)
	}

	// Test when bounds overlap over longitude range
	var lat5, long5, r5 float32 = 89.9999, -122.446747, 1000
	expected5 := latLongRange{}
	result5, _ := getGPSBound(lat5, long5, r5)
	if result5 != expected5 {
		t.Errorf("Overlap at latitude unexpected value, got: %v, want: %v", result5, expected5)
	}

}
