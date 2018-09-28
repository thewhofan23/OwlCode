package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// Query - Used to marshal graphQL query
type Query struct {
	Query     string
	Variables struct{}
}

// Config - Importing configs from config.json
type Config struct {
	Token   string // Access token
	Timeout int    // HTTP timeout
}

// Address - Create struct to unmarshal and hold the address
type Address struct {
	Name string
}

// Segment - Create struct to unmarshal and hold the start/end segments
type Segment struct {
	Address Address
	Time    int
}

// Driver - Create struct to unmarshal and hold the driver name
type Driver struct {
	Name string
}

// TripEntry - Create struct to unmarshal and hold a trip entry
type TripEntry struct {
	Driver Driver
	End    Segment
	Start  Segment
}

// VehicleActivityReport - Create struct to unmarshal and hold vehicleActivityReport
type VehicleActivityReport struct {
	TripEntries []TripEntry
}

// Devices - Create struct to unmarshal and hold each device
type Devices struct {
	Name                  string
	VehicleActivityReport VehicleActivityReport
}

// Group - Create struct to unmarshal and hold Devices
type Group struct {
	Devices []Devices
}

// Data - Create struct to unmarshal and hold Group
type Data struct {
	Group Group
}

func main() {

	// TODO: Remove hardcoding when reading
	groupID := "3991"
	endTime := "1537472193229"
	duration := "8640000"

	// Read in the access token from an untracked local file
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("File open failed: ", err)
	}
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)
	var config Config
	json.Unmarshal(byteValue, &config)

	client := &http.Client{}
	client.Timeout = time.Second * time.Duration(config.Timeout)

	query := `
	{
		group(id: ` + groupID + `) {
		  devices {
			name
			vehicleActivityReport(endTime:` + endTime + `, duration:` + duration + `) {
			  tripEntries {
				start {
				  time
				  address {
					name
				  }
				}
				end {
				  time
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

	q := Query{
		Query: query,
	}
	b, err := json.Marshal(q)

	// Generate the API query
	url := "https://api.samsara.com/v1/admin/graphql"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		fmt.Printf("Error generating request: %s", err)
	}
	req.Header.Add("X-Access-Token", config.Token)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error getting response: %s", err)
	}
	// Check if we get any page errors, this is not caught by err
	if resp.StatusCode != 200 {
		fmt.Println("Error Code")
		b, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(b))
		return
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var data Data
	json.Unmarshal(body, &data)
	fmt.Println(data.Group.Devices[4].VehicleActivityReport.TripEntries[0].Start.Address.Name)
}

// func tosQuery(id, end, duration string) *Data{
// 	return
// }
