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
}

type recordOS struct {
	ChangedAtMs int
	IntValue    int
}

type cameraElement struct {
	startTime int
	endTime   int
	duration  int
}

type cameraElements []cameraElement

func main() {
	deviceID := "212014918137973"
	endTimeMs := "1540400526230"
	startTimeMs := "1540397854230"
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
	cameraData, err := recordingQuery(deviceID, endTimeMs, strconv.Itoa(endTimeMsInt-startTimeMsInt))
	if err != nil {
		fmt.Println("Error encountered:", err)
		return
	}
	aggregateRecording := parseRecording(cameraData, startTimeMsInt, endTimeMsInt)
	displayRecording(aggregateRecording)
}

func displayRecording(records cameraElements) {
	total := 0
	fmt.Printf("\n\n\n")
	for _, r := range records {
		total += r.duration
		fmt.Printf("Start: %s   End: %s    Duration: %s \n", time.Unix(int64(r.startTime/1000), 0), time.Unix(int64(r.endTime/1000), 0), secToHours(r.duration/1000))
	}
	fmt.Println("\n Total recording Time: ", secToHours(total/1000))
}

func parseRecording(data recordData, startTimeMs, endTimeMs int) cameraElements {
	// Possible Edge Cases
	// Only one segment NOT POSSIBLE
	// No segments 	NOT POSSIBLE
	// In range, segment starts as recording
	// In range, segment ends as recording
	// In range, entirely one segment is recording
	elapsedTime, endTime, startTime := 0, 0, 0
	var cE cameraElements
	segmentList := data.Device.ObjectStat

	for i := 0; i < len(segmentList)-1; i++ {
		elapsedTime = 0

		// If segment starts as recording
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

			elapsedTime = endTime - startTime
			cE = append(cE, cameraElement{startTime, endTime, elapsedTime})
		}
	}
	return cE
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
