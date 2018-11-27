package tlclient

import (
	. "github.com/xnaveira/alertdelay/baseclient"
	"log"
	"net/http"
	"net/url"
	"os"
)

type TlClient struct {
	bClient BaseClient
	apiKey string
}

var ResrobotClient *TlClient

//const APIKEY = "732d8c4e-a795-4dcb-b291-6f3712f0f7a8"
const SL = "275"
const TRAINS = "16"


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

func init() {
	tburl, err := url.Parse("https://api.resrobot.se")
	if err != nil {
		log.Fatal(err)
	}

	resrobotApiKey := os.Getenv("RESROBOTCLIENT_API_KEY")

	if resrobotApiKey == "" {
		log.Fatal("RESROBOTCLIENT_API_KEY must be set")
	}

	ResrobotClient = &TlClient{
		BaseClient{
			BaseURL:    tburl,
			UserAgent:  USER_AGENT,
			HttpClient: &http.Client{},
		},
		resrobotApiKey,
	}
}

func (c *TlClient) GetTrains(origin, destination string) (*DepartureArray, error) {

	//url := "https://api.resrobot.se/v2/departureBoard?key=732d8c4e-a795-4dcb-b291-6f3712f0f7a8&id=740000559&maxJourneys=100&format=json&operator=275&products=16&direction=740000721"
	urlValues := url.Values{}
	urlValues.Set("key", c.apiKey)
	urlValues.Add("maxJourneys", "100")
	urlValues.Add("format", "json")
	urlValues.Add("operator", SL)
	urlValues.Add("products", TRAINS)
	urlValues.Add("direction", destination)
	urlValues.Add("id", origin)

	req, err := c.bClient.NewRequest("GET", "/v2/departureBoard", urlValues.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var trains DepartureArray

	_, err = c.bClient.Do(req, &trains)
	if err != nil {
		return nil,err
	}

	return &trains, err
}