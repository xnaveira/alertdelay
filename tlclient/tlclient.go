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
	apiKey  string
}

var ResrobotClient *TlClient

//Filtrerar svaret på produktsnivå
//Anger trafikslag som trafikerar hållplatsen, summerar ihop om fler än ett. Möjliga värden:
//2 - Snabbtåg, Expresståg, Arlanda Express
//4 - Regionaltåg, InterCitytåg
//8 - Expressbuss, Flygbussar
//16 - Lokaltåg, Pågatåg, Öresundståg
//32 - Tunnelbana
//64 – Spårvagn
//128 – Buss
//256 – Färja, Utrikes Färja
//(inte en fullständig lista)
//Exempel: 6 = 2 (Snabbtåg, Expresståg, Arlanda Express) + 4 (Regionaltåg, InterCitytåg)

const SL = "275"
const TRAINS = "20" //sl + sj

type Stop struct {
	Name     string `json:"name"`
	Id       string `json:"id"`
	ExtId    string `json:"extId"`
	RouteIdx int    `json:"routeIdx"`
	//Lon        string `json:"lon"`
	//Lat        string `json:"lat"`
	DepTime    string `json:"depTime"`
	DepDate    string `json:"depDate"`
	RtDepTime  string `json:"rtDepTime,omitempty"`
	RtDepDate  string `json:"rtDepdate,omitempty"`
	RtDepTrack string `json:"rtDepTrack,omitempty"`
}

type StopArray struct {
	Stops []Stop `json:"Stop"`
}

type Departure struct {
	Product           interface{} `json:"Product"`
	Stops             StopArray   `json:"Stops"`
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

	urlValues := url.Values{}
	urlValues.Set("key", c.apiKey)
	urlValues.Add("maxJourneys", "100")
	urlValues.Add("format", "json")
	//urlValues.Add("operator", SL)
	urlValues.Add("products", TRAINS)
	//urlValues.Add("direction", destination)
	urlValues.Add("id", origin)

	req, err := c.bClient.NewRequest("GET", "/v2/departureBoard", urlValues.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var trains DepartureArray

	_, err = c.bClient.Do(req, &trains)
	if err != nil {
		return nil, err
	}

	return &trains, err
}
