package main

import (
	"fmt"
	"github.com/go-yaml/yaml"
	. "github.com/xnaveira/alertdelay/slclient"
	. "github.com/xnaveira/alertdelay/tlclient"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

type notifier interface {
	Notify(string) error
}

type section struct {
	Name        string `yaml:"name"`
	Origin      string `yaml:"origin"`
	Destination string `yaml:"destination"`
}

type conf struct {
	Stations map[string]string `yaml:"stations"`
	Sections []section         `yaml:"sections"`
}

var c conf

func main() {

	err := c.getConf("alertdelay.yaml")
	if err != nil {
		log.Fatal("problem parsin the config: %s", err)
	}

	interval := time.Second * time.Duration(300) //Defaults to 5 minutes

	if len(os.Args) > 1 {
		if os.Args[1] != "" {
			i, err := strconv.Atoi(os.Args[1])
			if err != nil {
				log.Fatal(err)
			}
			interval = time.Second * time.Duration(i)
		}
	}

	log.Printf("Initiating execution, interval is set to %d seconds", int(interval.Seconds()))

	for _, s := range c.Sections {
		go doEvery(interval, makeRoute(s.Origin, s.Destination))
	}

	//Block forever
	select {}

}

func (c *conf) getConf(file string) error {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		return err
	}

	return nil
}

func doEvery(d time.Duration, f func()) {
	f()
	for range time.Tick(d) {
		f()
	}
}

type trainStatus struct {
	ontime           bool
	trainNumber      string
	trainOrigin      string
	trainDestination string
	oldTime          string
	newTime          string
}

func checkDepartures(departures []Departure, destination string) ([]trainStatus, error) {
	var trainStatuses []trainStatus
	var ontime bool
	if len(departures) <= 0 {
		return nil, fmt.Errorf("departures seems to be empty: %v", departures)
	}
	for _, departure := range departures {
		if departure.Direction == c.Stations[destination] {
			departureTime := departure.Stops.Stops[0].DepTime
			departureRtTime := departure.Stops.Stops[0].RtDepTime
			if departureRtTime != "" && departureTime != departureRtTime {
				ontime = false
			} else {
				ontime = true
			}
			trainStatuses = append(trainStatuses, trainStatus{
				ontime:           ontime,
				trainNumber:      departure.TransportNumber,
				trainOrigin:      c.Stations[departure.StopId],
				trainDestination: departure.Direction,
				oldTime:          departureTime,
				newTime:          departureRtTime,
			})
		}
	}
	if len(trainStatuses) <= 0 {
		return nil, fmt.Errorf("got no statuses, probably destination is spelled inccorrectly")
	}
	return trainStatuses, nil
}

func makeRoute(origin, destination string) func() {
	return func() {
		err := runAlert(origin, destination, SlackClient)
		if err != nil {
			log.Println(err)
		}
	}
}

func runAlert(origin, destination string, n notifier) error {

	trains, err := ResrobotClient.GetTrains(origin, destination)

	if err != nil {
		return fmt.Errorf("could not get trains: %s", err)
	}

	statuses, err := checkDepartures(trains.Departures, destination)
	if err != nil {
		return fmt.Errorf("could not check depatures: %s", err)
	}

	for _, s := range statuses {
		log.Println(s.trainOrigin, s.trainDestination, s.oldTime, s.newTime, s.ontime)
		if !s.ontime {
			log.Println("Sending message to slack")
			err = n.Notify(fmt.Sprintf(
				"Train delayed: %s to %s. Original time: %s, New time: %s",
				s.trainOrigin,
				s.trainDestination,
				s.oldTime,
				s.newTime))
			if err != nil {
				return fmt.Errorf("couldn't send message to slack: %s", err)
			}
		}
	}

	err = n.Notify("Trains are checked")
	if err != nil {
		return err
	}

	return nil
}
