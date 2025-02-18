package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

/*
Internal Notes
Int value decoding
1: Recording
2: Not Recording Error
3: Not Recording Stopped
4: Camera Starting
5: Camera On

Will just need to find IntValue 1's then look to next status change, take difference. Aggregate the time

What we'll input:
deviceID
startTimeMs
endTimeMs (accept some input to mean live, perhaps 0)

What we'll output:

Segment list
-----
Start time
End time
Segment recording time
&
Total recording time

*/

// Structure to hold .json config data
type config struct {
	Token   string
	Timeout int
}

// Structure to format graphQL queries
type graphQL struct {
	Query     string
	Variables struct{}
}

// Structures to hold return information from graphQL
type recordData struct {
	Device device
}

type device struct {
	ObjectStat []recordOS
	DeviceName string `json:"name"`
	Group      groupName
}

type groupName struct {
	Name string
}

type recordOS struct {
	ChangedAtMs int
	IntValue    int
}

type cameraRecordElement struct {
	startTime int
	endTime   int
	duration  int
}

type cameraRecordElements struct {
	cameraElement []cameraRecordElement
	totalRecord   int
}

func main() {

	fmt.Println("\n Welcome to the camera recording time calculator!")

	input := os.Args

	if len(input) != 4 {
		fmt.Println("Format Invalid!: Please follow this format: ./recordingTime <deviceID> <startTimeMs> <endTimeMs>")
		return
	}

	deviceID := input[1]    // e.g. 212014918137973
	startTimeMs := input[2] // e.g. 1540397854230
	endTimeMs := input[3]   // e.g. 1540400526230

	// Check the inputs to see if they are valid integers
	_, err := strconv.Atoi(deviceID)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	startTimeMsInt, err := strconv.Atoi(startTimeMs)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	endTimeMsInt, err := strconv.Atoi(endTimeMs)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	if startTimeMsInt >= endTimeMsInt {
		fmt.Println("Start time is greater than or equal to end time! Please correct your times.")
		return
	}

	// Query for the recording data from graphQL
	cameraData, err := recordingQuery(deviceID, endTimeMs, strconv.Itoa(endTimeMsInt-startTimeMsInt))
	if err != nil {
		fmt.Println("Error encountered:", err)
		return
	}
	// Parse and calculate the queried data
	aggregateRecording := parseRecording(cameraData, startTimeMsInt, endTimeMsInt)
	// Display the results
	displayRecording(aggregateRecording, cameraData, startTimeMsInt, endTimeMsInt)
}

func displayRecording(records cameraRecordElements, data recordData, startTimeMs, endTimeMs int) {
	fmt.Printf("\n\n")
	for _, r := range records.cameraElement {
		fmt.Printf("Start: %s   End: %s    Duration: %s \n", time.Unix(int64(r.startTime/1000), 0), time.Unix(int64(r.endTime/1000), 0), secToHours(r.duration/1000))
	}
	fmt.Println("\nVehicle Name: ", data.Device.DeviceName)
	fmt.Println("Group Name: ", data.Device.Group.Name)
	fmt.Printf("\n Total recording time from %s to %s is: %s\n\n", time.Unix(int64(startTimeMs/1000), 0), time.Unix(int64(endTimeMs/1000), 0), secToHours(records.totalRecord/1000))
}

func parseRecording(data recordData, startTimeMs, endTimeMs int) cameraRecordElements {
	// Possible Edge Cases
	// Only one segment NOT POSSIBLE
	// No segments 	NOT POSSIBLE
	// In range, segment starts as recording
	// In range, segment ends as recording
	// In range, entirely one segment is recording
	elapsedTime, endTime, startTime := 0, 0, 0
	var cREs cameraRecordElements
	segmentList := data.Device.ObjectStat

	// Check through camera segments
	for i := 0; i < len(segmentList)-1; i++ {
		elapsedTime = 0

		// If the segment is recording
		if segmentList[i].IntValue == 1 {
			// Get startTime
			if i == 0 && segmentList[0].ChangedAtMs <= startTimeMs {
				startTime = startTimeMs
			} else {
				startTime = segmentList[i].ChangedAtMs
			}
			// Get endTime
			if segmentList[i+1].ChangedAtMs >= endTimeMs {
				endTime = endTimeMs
			} else {
				endTime = segmentList[i+1].ChangedAtMs
			}
			// Calculate total record time, add to totalRecord time, append segment to record list
			elapsedTime = endTime - startTime
			cREs.totalRecord += elapsedTime
			cREs.cameraElement = append(cREs.cameraElement, cameraRecordElement{startTime, endTime, elapsedTime})
		}
	}
	return cREs
}

func recordingQuery(deviceID, endTimeMs, durationMs string) (recordData, error) {
	// Read in the access token from an untracked local file
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("File open failed: ", err)
		return recordData{}, err
	}
	defer file.Close()
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Could not read the config file in site query", err)
		return recordData{}, err
	}
	var conf config
	json.Unmarshal(byteValue, &conf)

	client := &http.Client{}
	client.Timeout = time.Second * time.Duration(conf.Timeout)

	query := `{
		device(id:` + deviceID + `) {
			group{
				name
			}
			name
		  objectStat(statTypeEnum: osDDashcamState, endTime:` + endTimeMs + `, duration: ` + durationMs + `) {
			changedAtMs
			intValue
		  }
		}
	  }`

	q := graphQL{
		Query: query,
	}
	b, err := json.Marshal(q)
	if err != nil {
		fmt.Println("Error marshalling query information", err)
		return recordData{}, err
	}

	// Generate the API query
	url := "https://api.samsara.com/v1/admin/graphql"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		fmt.Printf("Error generating request: %s", err)
		return recordData{}, err
	}
	req.Header.Add("X-Access-Token", conf.Token)
	// Request data
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error getting response: %s", err)
		return recordData{}, err
	}
	defer resp.Body.Close()
	// Check if we get any page errors, this is not caught by err
	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return recordData{}, err
		}
		var data recordData
		json.Unmarshal(body, &data)
		return data, nil
	}
	return recordData{}, errors.New("Page error:" + strconv.Itoa(resp.StatusCode))
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
