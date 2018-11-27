package main

import (
	"fmt"
	. "github.com/xnaveira/alertdelay/slclient"
	. "github.com/xnaveira/alertdelay/tlclient"
	"log"
	"os"
	"strconv"
	"time"
)



//type BaseClient interface {
//	newRequest(string, string, string, interface{}) (*http.Request, error)
//	do(req *http.Request, v interface{}) (*http.Response, error)
//}
//
//type SlackClient interface {
//	BaseClient
//	postToSlack(string) error
//}
//
//type TlClient interface {
//	BaseClient
//	runAlert(string, string, []int) error
//	getTrains(string, string) (*DepartureArray, error)
//	makeRoute(string, string, []int) func()
//}
//


//type DelayAlert struct {
//	isAlert bool
//	message string
//}

type notifier interface {
	Notify(string) error
}

const SODERTALJE_CENTRUM = "740000721"
const KNIVSTA = "740000559"
const STOCKHOLM_ODENPLAN = "740001618"
const STOCKHOLM_CITY = "740001617"
const UPPSALA = "740000005"

var stations = map[string]string{
	SODERTALJE_CENTRUM: "Sodertälje Centrum",
	KNIVSTA:            "Knivsta",
	STOCKHOLM_ODENPLAN: "Stockholm Odenplan",
	STOCKHOLM_CITY:     "Stockholm City",
	UPPSALA:            "Uppsala",
}

var timeLayout = "15:04:05"

func main() {

	interval := time.Second*time.Duration(300) //Defaults to 5 minutes

	if len(os.Args) > 1 {
		if os.Args[1] != "" {
			i, err := strconv.Atoi(os.Args[1])
			if err != nil {
				log.Fatal(err)
			}
			interval = time.Second*time.Duration(i)
		}
	}


	log.Printf("Initiating execution, interval is set to %d seconds",int(interval.Seconds()))


	go doEvery(interval, makeRoute(KNIVSTA, SODERTALJE_CENTRUM, []int{3, 33}))
	go doEvery(interval, makeRoute(STOCKHOLM_ODENPLAN, UPPSALA, []int{15, 45}))

	//Block forever
	select {}

}

func doEvery(d time.Duration, f func()) {
	f()
	for _ = range time.Tick(d) {
		f()
	}
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
			ontime:           ontime,
			trainNumber:      departure.TransportNumber,
			trainOrigin:      stations[departure.StopId],
			trainDestination: departure.Direction,
			newTime:          departureTime,
		})
		if !ontime {
			trainStatuses[len(trainStatuses)-1].newTime = departureTime
		}

		//fmt.Println(departureTime.Minute())
		//fmt.Println(departure.Time)
	}
	return trainStatuses, nil
}

func makeRoute(origin, destination string, minutes []int) func() {
	return func() {
		err := runAlert(origin, destination, minutes, SlackClient)
		if err != nil {
			log.Println(err)
		}
	}
}

func runAlert(origin, destination string, minutes []int, n notifier) error {

	trains, err := ResrobotClient.GetTrains(origin, destination)

	if err != nil {
		return fmt.Errorf("could not get trains: %s", err)
	}

	statuses, err := checkDepartures(trains.Departures, minutes)
	if err != nil {
		return fmt.Errorf("could not check depatures: %s", err)
	}

	for _, s := range statuses {
		log.Println(s.trainOrigin, s.trainDestination, s.newTime.Format(timeLayout), s.ontime)
		if !s.ontime {
			log.Println("Sending message to slack")
			err = n.Notify(fmt.Sprintf(
				"Train delayed: %s to %s. New time: %s",
				s.trainOrigin,
				s.trainDestination,
				s.newTime.Format(timeLayout)))
			if err != nil {
				return fmt.Errorf("couldn't send message to slack: %s",err)
			}
		}
	}
	err = n.Notify("Trains are checked")
	if err != nil {
		return err
	}

	return nil
}
