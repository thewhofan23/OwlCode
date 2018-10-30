package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"
)

//TODO: Add tests

// **** HTTP Structs *****

// Query - Used to marshal graphQL query
type graphQL struct {
	Query     string
	Variables struct{}
}

// Config - Importing configs from config.json
type config struct {
	Token      string  // Access token
	Timeout    int     // HTTP timeout
	BoundMulti float32 // Bound Multiplier
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
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")

func main() {

	// Used to profile the program
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	programStart := time.Now()

	// TODO: Remove hardcoding when reading, ask eric about best way to manage

	groupID := "3991"
	endTime := "1540341729936"
	duration := "3600000"
	expanded := false

	fmt.Println("Running Time on Site Report...")
	// Grab vehicle and driver data from graphQL
	start := time.Now()
	tosData, err := tosQuery(groupID, endTime, duration)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Total time to fetch vehicle/location data: ", time.Since(start))

	start1 := time.Now()

	// Grab site data from graphQL
	siteData, err := siteQuery(groupID)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Total time to fetch site data: ", time.Since(start1))

	// Format input arguments
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

	// Run the time on site report using the data from earlier graphQL queries
	report := checkSite(siteData, tosData, intEndTime, intDuration)
	// Format and print the results of checkSite
	printSite(report, expanded)
	fmt.Println("Total Program Runtime: ", time.Since(programStart))
}

// **** SUPPORTING FUNCTIONS ****

// Requests driver and vehicle information from graphQL
// Nearly all runtime of program happens here when requesting data from the server.
func tosQuery(id, end, duration string) (tosData, error) {
	// Read in the access token from an untracked local file
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("File open failed: ", err)
		return tosData{}, err
	}
	defer file.Close()
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Could not read the config file in site query", err)
		return tosData{}, err
	}
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
	if err != nil {
		fmt.Println("Error marshalling query information", err)
		return tosData{}, err
	}

	// Generate the API query
	url := "https://api.samsara.com/v1/admin/graphql"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		fmt.Printf("Error generating request: %s", err)
		return tosData{}, err
	}
	req.Header.Add("X-Access-Token", conf.Token)
	// Request data
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error getting response: %s", err)
		return tosData{}, err
	}
	defer resp.Body.Close()
	// Check if we get any page errors, this is not caught by err
	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return tosData{}, err
		}
		var data tosData
		json.Unmarshal(body, &data)
		return data, nil
	}
	return tosData{}, errors.New("Page error:" + strconv.Itoa(resp.StatusCode))

}

// Requests address information from graphQL
func siteQuery(id string) (siteData, error) {
	// Read in the access token from an untracked local file
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("File open failed: ", err)
		return siteData{}, err
	}
	defer file.Close()
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Could not read the config file in site query", err)
		return siteData{}, err
	}
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
	if err != nil {
		fmt.Println("Error marshalling query information", err)
		return siteData{}, err
	}

	// Generate the API query
	url := "https://api.samsara.com/v1/admin/graphql"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		fmt.Printf("Error generating request: %s", err)
		return siteData{}, err
	}
	req.Header.Add("X-Access-Token", conf.Token)
	// Request data from server
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error getting response: %s", err)
		return siteData{}, err
	}
	defer resp.Body.Close()
	// Check if we get any page errors, this is not caught by err
	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return siteData{}, err
		}
		var data siteData
		json.Unmarshal(body, &data)
		return data, nil
	}
	return siteData{}, errors.New("Page error:" + strconv.Itoa(resp.StatusCode))
}

// Creates the bounding rectangle used to quickly condition if GPS coordinate is within a site
func getGPSBound(lat, long, r float32) (latLongRange, error) {
	// If no radius, just return initial coordinates
	if r == 0 {
		return latLongRange{long, long, lat, lat}, nil
	}
	// If radius is negative, just flip sign
	if r < 0 {
		r = -r
	}

	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("File open failed: ", err)
		return latLongRange{}, err
	}
	defer file.Close()
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Could not read the json config file in!", err)
		return latLongRange{}, err
	}
	var conf config
	json.Unmarshal(byteValue, &conf)

	// Assuming radius is sufficently small, and vehicles are not driving north or south enough,
	// such that we have to check for latitude overlapping at the poles
	var bound latLongRange
	multi := conf.BoundMulti
	bound.latMin = lat - (multi*r/6371000)*180/math.Pi
	bound.latMax = lat + (multi*r/6371000)*180/math.Pi
	// // Check if latitude bounds overlap over north or south poles, used to catch very large bounds
	if bound.latMax > 90 || bound.latMin < -90 {
		return latLongRange{}, errors.New("error: bounds overlap over north or south pole")
	}
	bound.longMin = long - (multi*r/6371000)*180/math.Pi
	bound.longMax = long + (multi*r/6371000)*180/math.Pi
	// If longitude min/max cuts over 180th Meridian, create belt around earth whose width is difference of min and max latitude. This will likely pass more candidate GPS coordinates to haversine,
	// but currently preferable to introducing more complex logic and conditionals
	if bound.longMin < -180 || bound.longMax > 180 {
		bound.longMin = -180
		bound.longMax = 180
	}

	return bound, nil
}

func siteVehicle(wg *sync.WaitGroup, siteReport *siteOverall, s site, td tosData, startTime, endTime, duration int) {
	defer wg.Done()
	lineEntry := make([]siteReportLine, 0)
	bound, err := getGPSBound(s.Latitude, s.Longitude, s.Radius)
	if err != nil {
		fmt.Println("Could not define the GPS bounds for "+s.Name, err)
		return
	}
	var siteReportElem siteOverall
	totalTimeAtSite := 0
	totalUniqVehicles := 0
	totalUniqVisits := 0
	// For each site, check each vehicle
	for _, vehicle := range td.Group.Devices {
		didVisit := false
		// Check each end of trip for each vehicle for each site to figure out if vehicle ended within a site
		for i, trip := range vehicle.VAR.TripEntries {

			if i == 0 && greatCircleDist(trip.Start.Lat, trip.Start.Lng, s.Latitude, s.Longitude) <= s.Radius {
				arrivalTime := startTime
				departureTime := trip.Start.Time
				if departureTime >= arrivalTime {
					sRL := siteReportLine{trip.Driver.Name, arrivalTime, departureTime, vehicle.Name, trip.Start.Lat, trip.Start.Lng}
					lineEntry = append(lineEntry, sRL)
					totalTimeAtSite += (departureTime - arrivalTime) / 1000
					totalUniqVisits++
					didVisit = true
				}
			}

			// Check if this is the end of the recorded trips, if so, use user inputted endTime as the departureTime
			var departureTime int
			if i >= len(vehicle.VAR.TripEntries)-1 {
				departureTime = endTime
			} else {
				departureTime = vehicle.VAR.TripEntries[i+1].Start.Time
			}
			// Check if point is within bounds and within time frame before using greatCircleDist (heavy computation)
			if trip.End.Lat >= bound.latMin && trip.End.Lat <= bound.latMax &&
				trip.End.Lng >= bound.longMin && trip.End.Lng <= bound.longMax && departureTime-trip.End.Time > 0 && trip.End.Time >= startTime {

				// Calculate if distance is within radius using great circle formula
				if greatCircleDist(trip.End.Lat, trip.End.Lng, s.Latitude, s.Longitude) <= s.Radius {
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
		siteReportElem = siteOverall{lineEntry, s.Name, totalUniqVehicles, totalUniqVisits, totalTimeAtSite}
		*siteReport = siteReportElem
	}
}

// Iterates through site, vehicle, and driver information to
func checkSite(sd siteData, td tosData, endTime int, duration int) []siteOverall {
	wg := &sync.WaitGroup{}
	siteReport := make([]siteOverall, len(sd.Group.Sites))
	startTime := endTime - duration
	// Check at each site
	for i, site := range sd.Group.Sites {
		wg.Add(1)
		go siteVehicle(wg, &siteReport[i], site, td, startTime, endTime, duration)
	}

	wg.Wait()

	return siteReport
}

// Prints the time on site information in a presentable way
func printSite(siteReports []siteOverall, expanded bool) {
	fmt.Printf("\n\n")
	// Iterate through sites
	for _, siteReport := range siteReports {
		// If site was visited, display information
		if siteReport.totalVisits > 0 {
			fmt.Printf("%-40s %-5d %d %s \n", siteReport.siteName, siteReport.totalVehicles, siteReport.totalVisits,
				secToHours(siteReport.totalTime/siteReport.totalVisits))
			// If user would like detailed trip information for the sites
			if expanded {
				for _, visit := range siteReport.lineEntry {
					fmt.Printf("%-6s %-25s %-35s %-35s %12s %f %f \n", visit.vehicleName, visit.driverName, time.Unix(int64(visit.arrival/1000), 0),
						time.Unix(int64(visit.departure/1000), 0), secToHours((visit.departure-visit.arrival)/1000), visit.lat, visit.long)
				}
				fmt.Printf("\n")
			}
		}
	}
	fmt.Printf("\n")
}

// Formats seconds into the time on site format of Xh Ym, or Xm Ys
func secToHours(seconds int) string {
	if seconds/3600 >= 1 {
		hours := seconds / 3600
		min := seconds % 60
		return strconv.Itoa(hours) + "h " + strconv.Itoa(min) + "m"
	} else if seconds < 0 {
		return "negative"
	}
	min := seconds / 60
	sec := seconds % 60
	return strconv.Itoa(min) + "m " + strconv.Itoa(sec) + "s"
}

// Calculates the great circle distance between two GPS coordinates on an geometrical approximation of Earth (Ellipsoid)
func greatCircleDist(lat1, long1, lat2, long2 float32) float32 {
	// Assuming radius of earth is 6371000 m
	R := 6371000

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
