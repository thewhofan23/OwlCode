package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

// **** HTTP Structs *****

// Query - Used to marshal graphQL query
type graphQL struct {
	Query     string
	Variables struct{}
}

// Config - Importing configs from config.json
type config struct {
	Token   string // Access token
	Timeout int    // HTTP timeout
}

// **** Device Data Structs *****

// Address - Create struct to unmarshal and hold the address
type address struct {
	Name string
}

// Segment - Create struct to unmarshal and hold the start/end segments
type segment struct {
	Address address
	Lat     float32
	Lng     float32
	Time    int
}

// Driver - Create struct to unmarshal and hold the driver name
type driver struct {
	Name string
}

// TripEntry - Create struct to unmarshal and hold a trip entry
type tripEntry struct {
	Driver driver
	End    segment
	Start  segment
}

// VAR - Create struct to unmarshal and hold vehicleActivityReport
type vehicleActivityReport struct {
	TripEntries []tripEntry
}

// Devices - Create struct to unmarshal and hold each device
type devices struct {
	Name string
	VAR  vehicleActivityReport `json:"vehicleActivityReport"`
}

// Group - Create struct to unmarshal and hold Devices
type group struct {
	Devices []devices
}

type tosData struct {
	Group group
}

// **** Site/Address Structs *****

// Site - Create struct to unmarshal and hold Site data
type site struct {
	Latitude  float32
	Longitude float32
	Name      string
	Radius    float32
}

// Sites - Create struct to unmarshal and hold array of Site data
type sites struct {
	Sites []site `json:"addresses"`
}

type siteData struct {
	Group sites
}

// Structs to contain the resulting site information after processing
type siteReportLine struct {
	driverName  string
	arrival     int
	departure   int
	vehicleName string
	lat         float32
	long        float32
}

type siteOverall struct {
	lineEntry     []siteReportLine
	siteName      string
	totalVehicles int
	totalVisits   int
	totalTime     int
}

// Struct to hold the bound of a GPS rectangle
type latLongRange struct {
	longMin float32
	longMax float32
	latMin  float32
	latMax  float32
}

// **** Main *****

func main() {

	// TODO: Remove hardcoding when reading
	start := time.Now()
	groupID := "3991"
	endTime := "1535871599999"
	duration := "8380799000"
	expanded := false

	fmt.Println("Running Time on Site Report...")

	tosData := tosQuery(groupID, endTime, duration)
	siteData := siteQuery(groupID)
	intEndTime, err := strconv.Atoi(endTime)
	if err != nil {
		fmt.Println("Could not convert endTime to an integer")
		return
	}
	intDuration, err := strconv.Atoi(duration)
	if err != nil {
		fmt.Println("Could not convert duration to an integer")
		return
	}
	report := checkSite(siteData, tosData, intEndTime, intDuration)
	printSite(report, expanded)

	fmt.Println("Runtime: ", time.Since(start))
}

// **** SUPPORTING FUNCTIONS ****

// Requests driver and vehicle information from graphQL
func tosQuery(id, end, duration string) tosData {
	// Read in the access token from an untracked local file
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("File open failed: ", err)
	}
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)
	var conf config
	json.Unmarshal(byteValue, &conf)

	client := &http.Client{}
	client.Timeout = time.Second * time.Duration(conf.Timeout)

	query := `
		{
			group(id: ` + id + `) {
				devices {
				name
				vehicleActivityReport(endTime:` + end + `, duration:` + duration + `) {
					tripEntries {
					start {
						time
						lat
						lng
						address {
						name
						}
					}
					end {
						time
						lat
						lng
						address {
						name
						}
					}
					driver {
						name
					}
					}
				}
				}
			}
			}
			
		`

	q := graphQL{
		Query: query,
	}
	b, err := json.Marshal(q)

	// Generate the API query
	url := "https://api.samsara.com/v1/admin/graphql"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		fmt.Printf("Error generating request: %s", err)
	}
	req.Header.Add("X-Access-Token", conf.Token)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error getting response: %s", err)
	}
	// Check if we get any page errors, this is not caught by err
	if resp.StatusCode != 200 {
		fmt.Println("Error Code")
		b, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(b))
		//return nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var data tosData
	json.Unmarshal(body, &data)
	return data
}

// Requests address information from graphQL
func siteQuery(id string) siteData {
	// Read in the access token from an untracked local file
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("File open failed: ", err)
	}
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)
	var conf config
	json.Unmarshal(byteValue, &conf)

	client := &http.Client{}
	client.Timeout = time.Second * time.Duration(conf.Timeout)

	query := `
	{
		group(id:` + id + `) {
			addresses {
				name
				latitude
				longitude
				radius
			}
		}
	}
	`

	q := graphQL{
		Query: query,
	}
	b, err := json.Marshal(q)

	// Generate the API query
	url := "https://api.samsara.com/v1/admin/graphql"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		fmt.Printf("Error generating request: %s", err)
	}
	req.Header.Add("X-Access-Token", conf.Token)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error getting response: %s", err)
	}
	// Check if we get any page errors, this is not caught by err
	if resp.StatusCode != 200 {
		fmt.Println("Error Code")
		b, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(b))
		//return nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var data siteData
	json.Unmarshal(body, &data)
	return data
}

// Creates the bounding rectangle used to quickly condition if GPS coordinate is within a site
func getGPSBound(lat, long, r float32) latLongRange {
	// Assuming r is in meters
	// First calculate the min and max latitude, since that doesn't change much as you move along range
	// TODO: Find better way to store (config?)
	// Assuming radius is sufficently small, and vehicles not driving north or south enough, such that we have to check for latitude overlapping poles
	var bound latLongRange
	bound.latMin = lat - (r/6371000)*180/math.Pi
	bound.latMax = lat + (r/6371000)*180/math.Pi
	bound.longMin = long - (r/6371000)*180/math.Pi
	bound.longMax = long + (r/6371000)*180/math.Pi
	return bound
}

// Iterates through site, vehicle, and driver information to
func checkSite(sd siteData, td tosData, endTime int, duration int) []siteOverall {
	siteReport := make([]siteOverall, 0)
	startTime := endTime - duration
	// Check at each site
	for _, site := range sd.Group.Sites {
		totalTimeAtSite := 0
		bound := getGPSBound(site.Latitude, site.Longitude, site.Radius)
		lineEntry := make([]siteReportLine, 0)
		var siteReportElem siteOverall
		totalUniqVehicles := 0
		totalUniqVisits := 0
		// For each site, check each vehicle
		for _, vehicle := range td.Group.Devices {
			didVisit := false
			// Check each end of trip for each vehicle for each site to figure out if vehicle ended within a site
			for i, trip := range vehicle.VAR.TripEntries {
				// Check if this is the end of the recorded trips, if so, use user inputted endTime as the departureTime
				var departureTime int
				if i >= len(vehicle.VAR.TripEntries)-1 {
					departureTime = endTime
				} else {
					departureTime = vehicle.VAR.TripEntries[i+1].Start.Time
				}
				// Check if point is within bounds and within time frame before using greatCircleDist (heavy computation)
				if trip.End.Lat > bound.latMin && trip.End.Lat < bound.latMax &&
					trip.End.Lng > bound.longMin && trip.End.Lng < bound.longMax && departureTime-trip.End.Time > 0 && trip.End.Time >= startTime {

					// Calculate if distance is within radius using great circle formula
					if greatCircleDist(trip.End.Lat, trip.End.Lng, site.Latitude, site.Longitude) <= site.Radius {
						arrivalTime := trip.End.Time
						sRL := siteReportLine{trip.Driver.Name, arrivalTime, departureTime, vehicle.Name, trip.End.Lat, trip.End.Lng}
						lineEntry = append(lineEntry, sRL)
						totalTimeAtSite += (departureTime - arrivalTime) / 1000
						totalUniqVisits++
						didVisit = true
					}
				}
			}
			// If the vehicle visited one site during time range, increment number of vehicles that visited by one
			if didVisit {
				totalUniqVehicles++
			}
		}
		// Append this site's information to the total site list, if one vehicle has visited
		if totalUniqVehicles > 0 {
			siteReportElem = siteOverall{lineEntry, site.Name, totalUniqVehicles, totalUniqVisits, totalTimeAtSite}
			siteReport = append(siteReport, siteReportElem)
		}
	}
	return siteReport
}

func printSite(siteReports []siteOverall, expanded bool) {
	fmt.Printf("\n\n")
	for i, siteReport := range siteReports {
		fmt.Printf("%d %-40s %-5d %d %s \n", i, siteReport.siteName, siteReport.totalVehicles, siteReport.totalVisits,
			secToHours(siteReport.totalTime/siteReport.totalVisits))
		if expanded {
			for _, visit := range siteReport.lineEntry {
				fmt.Printf("%-6s %-25s %-35s %-35s %12s %f %f \n", visit.vehicleName, visit.driverName, time.Unix(int64(visit.arrival/1000), 0),
					time.Unix(int64(visit.departure/1000), 0), secToHours((visit.departure-visit.arrival)/1000), visit.lat, visit.long)
			}
			fmt.Printf("\n")
		}
	}
	fmt.Printf("\n")
}

// Formats seconds into the time on site format of Xh Ym, or Xm Ys
func secToHours(seconds int) string {
	if seconds/3600 > 0 {
		hours := seconds / 3600
		min := seconds % 60
		return strconv.Itoa(hours) + "h " + strconv.Itoa(min) + "m"
	} else {
		min := seconds / 60
		sec := seconds % 60
		return strconv.Itoa(min) + "m " + strconv.Itoa(sec) + "s"
	}
}

// Calculates the great circle distance between two GPS coordinates on an geometrical approximation of Earth (Ellipsoid)
func greatCircleDist(lat1, long1, lat2, long2 float32) float32 {
	R := 6371 * 1000 // meters

	// TODO: should variables all be float64, or is it fine to cast here? Speed (unknown) vs space (doubled)
	radDifLat := degToRad(lat2 - lat1)
	radDifLong := degToRad(long2 - long1)
	radLat1 := degToRad(lat1)
	radLat2 := degToRad(lat2)
	// The haversine formula
	a := math.Sin(radDifLat/2)*math.Sin(radDifLat/2) + math.Cos(radLat1)*
		math.Cos(radLat2)*math.Sin(radDifLong/2)*math.Sin(radDifLong/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := float64(R) * c

	return float32(d)
}

// Returning float64 since golang math package uses float64, instead of float32
func degToRad(x float32) float64 {
	return float64(x) * math.Pi / 180
}
