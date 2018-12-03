package baseclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const USER_AGENT = "alertdelay/0.0.1"

type BaseClient struct {
	BaseURL   *url.URL
	UserAgent string

	HttpClient *http.Client
}

func (c *BaseClient) NewRequest(method, path, query string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)
	if query != "" {
		u.RawQuery = query
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		//err := json.NewEncoder(buf).Encode(body)
		_, err := buf.Write(body.([]byte))
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
	req.Header.Set("User-Agent", c.UserAgent)

	return req, nil
}

func (c *BaseClient) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("could not read the response body: %s", err)
	}

	if 200 <= resp.StatusCode && resp.StatusCode < 400 {
		if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
			//err = json.NewDecoder(bytes.NewReader(body)).Decode(v)
			err = json.Unmarshal(body, v)
		} else {
			*v.(*interface{}) = body
		}
	} else {
		err = fmt.Errorf("there was an error while requesting %s: %s %s", req.URL.String(), resp.Status, string(body))
	}
	return resp, err
}
