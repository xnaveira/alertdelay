package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	BaseURL   *url.URL
	UserAgent string

	httpClient *http.Client
}

type Departure struct {
	Product           interface{} `json:"Product"`
	Stops             interface{} `json:"Stops"`
	Name              string      `json:"name"`
	Type              string      `json:"type"`
	Stop              string      `json:"stop"`
	StopId            string      `json:"stopid"`
	StopExtId         string      `json:"stopExtId"`
	Time              string      `json:"time"`
	Date              string      `json:"date"`
	Direction         string      `json:"direction"`
	TransportNumber   string      `json:"transportNumber"`
	TransportCategory string      `json:"transportCategory"`

	//RtTime     string   `json:"rtTime"`
	//RtTrack    string   `json:"rtTrack"`
	//RtDepTrack string   `json:"rtDepTrack"`
	//RtDate string `json:"rt_date"`
}
type DepartureArray struct {
	Departures []Departure `json:"Departure"`
}

type DelayAlert struct {
	isAlert bool
	message string
}

const USER_AGENT = "alertdelay/0.0.1"
const APIKEY = "732d8c4e-a795-4dcb-b291-6f3712f0f7a8"
const SL = "275"
const TRAINS = "16"

const SODERTALJE_CENTRUM = "740000721"
const KNIVSTA = "740000559"
const STOCKHOLM_ODENPLAN = "740001618"
const STOCKHOLM_CITY = "740001617"
const UPPSALA = "740000005"


var stations = map[string]string{
SODERTALJE_CENTRUM: "SODERTALJE_CENTRUM",
KNIVSTA: "KNIVSTA",
STOCKHOLM_ODENPLAN: "STOCKHOLM_ODENPLAN",
STOCKHOLM_CITY: "STOCKHOLM_CITY",
UPPSALA: "UPPSALA",
}

var timeLayout = "15:04:05"


func main() {


	//_ = stations

	//url := "https://api.resrobot.se/v2/departureBoard?key=732d8c4e-a795-4dcb-b291-6f3712f0f7a8&id=740000559&maxJourneys=100&format=json&operator=275&products=16&direction=740000721"
	tburl, err := url.Parse("https://api.resrobot.se")
	if err != nil {
		log.Fatal(err)
	}

	resrobotClient := &Client{
		BaseURL:    tburl,
		UserAgent:  USER_AGENT,
		httpClient: &http.Client{},
	}


	//err = resrobotClient.runAlert(KNIVSTA,SODERTALJE_CENTRUM, []int{3,33})
	//if err != nil {
	//	panic(err)
	//}

	//s := func() func() {
	//	return func() {
	//		kk := "hi"
	//	fmt.Println(kk)
	//	}
	//	}
	//
	//_ = s

	go doEvery(2 * time.Second, resrobotClient.makeRoute(KNIVSTA,SODERTALJE_CENTRUM, []int{3,33}))
	go doEvery(2 * time.Second, resrobotClient.makeRoute(STOCKHOLM_ODENPLAN,UPPSALA, []int{15,45}))

	//Block forever
	select {}



}



func doEvery(d time.Duration, f func()) {
	for _ = range time.Tick(d){
		f()
	}
}

func (c *Client) makeRoute(origin, destination string, minutes []int) func() {
	return func() {
		err := c.runAlert(origin,destination, []int{15,45})
		if err != nil {
			log.Println(err)
		}
	}
}

func (c *Client) runAlert(origin, destination string, minutes []int) error {
	trains, err := c.getTrains(origin, destination)

	if err != nil {
		return fmt.Errorf("could not get trains: %s",err)
	}
	
	statuses, err := checkDepartures(trains.Departures, []int{3, 33})
	if err != nil {
		return fmt.Errorf("could not check depatures: %s", err)
	}
	
	for _,s := range statuses {
	//fmt.Println(s.newTime.Format(timeLayout))
		log.Println(s.trainOrigin,s.trainDestination,s.newTime.Format(timeLayout),s.ontime)
	}
	
	return nil
}

type trainStatus struct {
	ontime           bool
	trainNumber      string
	trainOrigin      string
	trainDestination string
	newTime          time.Time
}

func checkDepartures(departures []Departure, minutes []int) ([]trainStatus, error) {
	ontime := false
	var trainStatuses []trainStatus
	if len(departures) <= 0 {
		return nil, fmt.Errorf("departures seems to be empty: %v", departures)
	}
	for _, departure := range departures {
		departureTime, err := time.Parse(timeLayout, departure.Time)
		if err != nil {
			return nil, fmt.Errorf("error parsing departure time %v: %s", departure.Time, err)
		}
		for _, m := range minutes {
			if m == int(departureTime.Minute()) {
				ontime = true
			}
		}
		trainStatuses = append(trainStatuses, trainStatus{
			ontime:ontime,
			trainNumber:departure.TransportNumber,
			trainOrigin:stations[departure.StopId],
			trainDestination:departure.Direction,
			newTime:departureTime,
		})
		if !ontime {
			trainStatuses[len(trainStatuses)-1].newTime = departureTime
		}

		//fmt.Println(departureTime.Minute())
		//fmt.Println(departure.Time)
	}
	return trainStatuses, nil
}

func (c *Client) newRequest(method, path, query string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)
	if query != "" {
		u.RawQuery = query
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	return req, nil
}

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if 200 <= resp.StatusCode && resp.StatusCode <= 400 {
		err = json.NewDecoder(resp.Body).Decode(v)
	} else {
		err = fmt.Errorf("there was an error while requesting %s: %s", req.URL.String(), resp.Status)
	}
	return resp, err
}

//origin, destination, expected minute
func (c *Client) getTrains(origin, destination string) (*DepartureArray, error) {

	urlValues := url.Values{}
	urlValues.Set("key", APIKEY)
	urlValues.Add("maxJourneys", "100")
	urlValues.Add("format", "json")
	urlValues.Add("operator", SL)
	urlValues.Add("products", TRAINS)
	urlValues.Add("direction", destination)
	urlValues.Add("id", origin)

	req, err := c.newRequest("GET", "/v2/departureBoard", urlValues.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var trains DepartureArray

	_, err = c.do(req, &trains)

	return &trains, err
}
