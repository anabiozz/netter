package netter

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type logger interface {
	Printf(string, ...interface{})
}

// Client represents http client
type Client struct {
	httpclient *http.Client
	logger
	retry
}

// NewClient represents new http client
func NewClient() *Client {
	return defaultClient
}

var defaultClient = &Client{
	httpclient: &http.Client{
		Transport: defaultTransport,
	},
	logger: log.New(os.Stderr, "", log.LstdFlags),
	retry: retry{
		WaitMin: 1 * time.Second,
		WaitMax: 10 * time.Second,
		Max:     4,
	},
}

// Do sends an HTTP request and returns an HTTP response
func (c *Client) Do(req *Request) (resp *http.Response, err error) {

	for i := 0; ; i++ {

		var code int

		if req.body != nil {
			body, err := req.body()
			if err != nil {
				return resp, err
			}
			if c, ok := body.(io.ReadCloser); ok {
				req.Body = c
			} else {
				req.Body = ioutil.NopCloser(body)
			}
		}

		resp, err = c.httpclient.Do(req.Request)
		if resp != nil {
			code = resp.StatusCode
		}
		if err != nil {
			c.logger.Printf("ERROR: %s %s request failed: %v", req.Method, req.URL, err)
		}

		retryable, checkErr := c.retry.isRetry(req.Context(), resp, err)

		if !retryable {
			if checkErr != nil {
				err = checkErr
			}
			return resp, err
		}

		remain := c.retry.Max - i
		if remain <= 0 {
			break
		}

		if err == nil && resp != nil {
			c.drainBody(resp.Body)
		}

		wait := c.retry.backoff(c.retry.WaitMin, c.retry.WaitMax, i)
		desc := fmt.Sprintf("%s %s", req.Method, req.URL)
		if code > 0 {
			desc = fmt.Sprintf("%s (status: %d)", desc, code)
		}

		c.logger.Printf("RETRY %s retrying in %s (%d left)", desc, wait, remain)

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}

	if resp != nil {
		if err := resp.Body.Close(); err != nil {
			c.logger.Printf(err.Error())
		}
	}
	return nil, fmt.Errorf("ERROR: %s %s giving up after %d attempts", req.Method, req.URL, c.Max+1)
}

func (c *Client) drainBody(body io.ReadCloser) {
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, 4096))
	if err != nil {
		c.logger.Printf("ERROR: reading response body: %v", err)
	}

	err = body.Close()
	if err != nil {
		c.logger.Printf(err.Error())
	}
}

// Get sends get request
func Get(url string) (*http.Response, error) {
	return defaultClient.Get(url)
}

// Get sends get request
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post sends post request
func Post(url, bodyType string, body interface{}) (*http.Response, error) {
	return defaultClient.Post(url, bodyType, body)
}

// Post sends post request
func (c *Client) Post(url, bodyType string, body interface{}) (*http.Response, error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(req)
}
