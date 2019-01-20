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
	"strings"
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

type hintervals struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type ban struct {
	Bdays  []string     `yaml:"days"`
	BHours []hintervals `yaml:"hours"`
}

type conf struct {
	Interval int               `yaml:"intervalinseconds,omitempty"`
	Stations map[string]string `yaml:"stations"`
	Sections []section         `yaml:"sections"`
	Ban      ban               `yaml:"ban"`
}

var c conf

type notification struct {
	msg       string
	timestamp time.Time
}

var notifications []notification

var confFile = "alertdelay.yaml"

const apiLimit = 10000

func main() {

	if len(os.Args) > 1 {
		if os.Args[1] != "" {
			confFile = os.Args[1]
		}
	}

	err := c.getConf(confFile)
	if err != nil {
		log.Fatal("problem parsing the config: ", err)
	}

	intervalInMinutes, nApiCalls := c.Ban.getInterval(len(c.Sections))
	interval := time.Second * time.Duration(intervalInMinutes*60)

	log.Printf("Initiating execution")
	log.Printf("interval is set to %d seconds", int(interval.Seconds()))
	//log.Printf("calculated executions per month: %d", c.Ban.hoursPerMonth(int(interval.Seconds()))*len(c.Sections))
	log.Printf("monitoring %d sections every %d minutes. Total api calls/month: %d", len(c.Sections), intervalInMinutes, nApiCalls)

	for _, s := range c.Sections {
		log.Println("monitoring route ", s.Name, " ", s.Origin, " ", s.Destination)
		go doEvery(interval, c.Ban, makeRoute(s.Origin, s.Destination))
	}

	//Block forever
	select {}

}

func (b *ban) isNowBanned() (bool, error) {
	now := time.Now()
	isBanned := false

	for _, day := range b.Bdays {
		if day == now.Format("Monday") {
			isBanned = true
			log.Printf("%s is banned", now.Format("Monday"))
		}
	}

	for _, i := range b.BHours {

		from, err := time.Parse("15:04", i.From)
		if err != nil {
			return true, err
		}
		to, err := time.Parse("15:04", i.To)
		if err != nil {
			return true, err
		}

		if now.After(from) && now.Before(to) {
			isBanned = true
			log.Printf("%s is in the banned interval.", now.Format("15:04"))
		}
	}

	return isBanned, nil

}

func (b *ban) getInterval(nsections int) (int, int) {

	nbDays := 0
	nbHours := 0

	if len(b.Bdays) > 0 {
		nbDays = len(b.Bdays)
	}

	if len(b.BHours) > 0 {
		for _, h := range b.BHours {
			f := strings.Split(h.From, ":")[0]
			t := strings.Split(h.To, ":")[0]
			fint, _ := strconv.Atoi(f)
			tint, _ := strconv.Atoi(t)
			tominusfrom := tint - fint
			nbHours = nbHours + tominusfrom
		}
	}

	//nnonbanneddays * nnonbannedhours * minutesinanhour / ( apiLimit / nofmonitoredsections )
	intevalInMinutes := (31 - nbDays) * (24 - nbHours) * 60 / (apiLimit / nsections)
	nApiCalls := (((31 - nbDays) * (24 - nbHours) * 60) / intevalInMinutes) * nsections

	return intevalInMinutes, nApiCalls

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

func doEvery(d time.Duration, b ban, f func()) {
	banned, err := b.isNowBanned()
	if err != nil {
		log.Fatal("error checking ban:", err)
	}
	if !banned {
		f()
	}
	for range time.Tick(d) {
		if !banned {
			f()
		}
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

	//Clean up old notifications
	for i, n := range notifications {
		if time.Since(n.timestamp).Hours() > 24 {
			notifications = append(notifications[:i], notifications[i+1:]...)
		}
	}

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
			msg := fmt.Sprintf(
				"Train delayed: %s to %s. Original time: %s, New time: %s",
				s.trainOrigin,
				s.trainDestination,
				s.oldTime,
				s.newTime)

			send := true
			for _, n := range notifications {
				//If message already sent
				if (n.msg) == msg {
					log.Println("skipping notify again: ", msg)
					send = false
				}
			}
			if send == true {
				err = n.Notify(msg)
				notifications = append(notifications, notification{msg, time.Now()})
			}
			if err != nil {
				return fmt.Errorf("couldn't send message to slack: %s", err)
			}
		}
	}

	//err = n.Notify("Trains are checked")
	if err != nil {
		return err
	}

	return nil
}
