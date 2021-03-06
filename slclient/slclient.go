package slclient

import (
	"encoding/json"
	"fmt"
	. "github.com/xnaveira/alertdelay/baseclient"
	"log"
	"net/http"
	"net/url"
	"os"
)

type slClient struct {
	bClient BaseClient
	channel string
}

var SlackClient *slClient

func init() {

	slurl, err := url.Parse("https://hooks.slack.com")
	if err != nil {
		log.Fatal(err)
	}

	slackChannel := os.Getenv("SLACK_CHANNEL")

	if slackChannel == "" {
		log.Fatal("SLACK_CHANNEL must be set")
	}

	SlackClient = &slClient{
		BaseClient{
			BaseURL:    slurl,
			UserAgent:  USER_AGENT,
			HttpClient: &http.Client{},
		},
		slackChannel,
	}

}

func (c *slClient) Notify(message string) error {

	payload := map[string]string{
		"text": message,
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := c.bClient.NewRequest(
		"POST",
		fmt.Sprintf("/services%s", c.channel),
		"",
		payloadJson)
	if err != nil {
		return err
	}

	var posted interface{}

	_, err = c.bClient.Do(req, &posted)

	log.Printf("message posted to slack: %s, status: %s", message, string(posted.([]byte)))

	if err != nil {
		return err
	}

	return nil

}
