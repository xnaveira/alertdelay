package slclient

import (
	"encoding/json"
	. "github.com/xnaveira/alertdelay/baseclient"
	"log"
	"net/http"
	"net/url"
)

type slClient struct {
	bClient BaseClient
}

var SlackClient *slClient

func init()  {

	slurl, err := url.Parse("https://hooks.slack.com")
	if err != nil {
		log.Fatal(err)
	}
	SlackClient = &slClient{
		BaseClient{
			BaseURL:    slurl,
			UserAgent:  USER_AGENT,
			HttpClient: &http.Client{},
		},
	}

}

func (c *slClient) PostToSlack(message string) error {

	payload := map[string]string{
		"text": message,
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	//https://hooks.slack.com/services/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
	req, err := c.bClient.NewRequest(
		"POST",
		"/services/T0SJTEHD3/B0SK4FWM8/2Lr9ljyaAxOcOeHeRWNqafVc",
		"",
		payloadJson)
	if err != nil {
		return err
	}

	//bytes, err := httputil.DumpRequest(req, true)

	//_ = bytes

	var posted interface{}

	_, err = c.bClient.Do(req,&posted)

	log.Printf("message posted to slack: %s, status: %s",message,string(posted.([]byte)))

	if err != nil {
		return err
	}

	return nil

}
