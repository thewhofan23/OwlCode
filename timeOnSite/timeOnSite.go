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
}

// Sites - Create struct to unmarshal and hold array of Site data
type sites struct {
	Sites []site `json:"addresses"`
}

type siteData struct {
	Group sites
}

func main() {

	// TODO: Remove hardcoding when reading
	groupID := "3991"
	endTime := "1537472193229"
	duration := "8640000"

	tosData := tosQuery(groupID, endTime, duration)
	fmt.Println(tosData.Group.Devices[4].VAR.TripEntries[0].Start.Address.Name)
	siteData := siteQuery(groupID)
	fmt.Println(siteData.Group.Sites[2].Name)
}

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
	body, _ := ioutil.ReadAll(resp.Body)
	var data tosData
	json.Unmarshal(body, &data)
	return data
}

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
	body, _ := ioutil.ReadAll(resp.Body)
	var data siteData
	json.Unmarshal(body, &data)
	return data
}
