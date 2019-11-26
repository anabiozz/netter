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

// Logger ..
type Logger interface {
	Printf(string, ...interface{})
}

// Client ..
type Client struct {
	httpclient *http.Client
	Logger
	Retry
}

// NewClient ..
func NewClient() *Client {
	return defaultClient
}

var defaultClient = &Client{
	httpclient: &http.Client{
		Transport: DefaultTransport,
	},
	Logger: log.New(os.Stderr, "", log.LstdFlags),
	Retry: Retry{
		WaitMin: 1 * time.Second,
		WaitMax: 10 * time.Second,
		Max:     4,
	},
}

// Do ..
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

		checkOK, checkErr := c.Retry.isRetry(req.Context(), resp, err)

		if err != nil {
			if c.Logger != nil {
				c.Logger.Printf("ERROR: %s %s request failed: %v", req.Method, req.URL, err)
			}
		}

		if !checkOK {
			if checkErr != nil {
				err = checkErr
			}
			return resp, err
		}

		remain := c.Retry.Max - i
		if remain <= 0 {
			break
		}

		if err == nil && resp != nil {
			c.drainBody(resp.Body)
		}

		wait := c.Retry.Backoff(c.Retry.WaitMin, c.Retry.WaitMax, i, resp)
		desc := fmt.Sprintf("%s %s", req.Method, req.URL)
		if code > 0 {
			desc = fmt.Sprintf("%s (status: %d)", desc, code)
		}
		if c.Logger != nil {
			c.Logger.Printf("RETRY %s retrying in %s (%d left)", desc, wait, remain)
		}
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}

	if resp != nil {
		resp.Body.Close()
	}
	return nil, fmt.Errorf("ERROR: %s %s giving up after %d attempts", req.Method, req.URL, c.Max+1)
}

func (c *Client) drainBody(body io.ReadCloser) {
	defer body.Close()
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, 4096))
	if err != nil {
		if c.Logger != nil {
			c.Logger.Printf("ERROR: reading response body: %v", err)
		}
	}
}

// Get ..
func Get(url string) (*http.Response, error) {
	return defaultClient.Get(url)
}

// Get ..
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post ..
func Post(url, bodyType string, body interface{}) (*http.Response, error) {
	return defaultClient.Post(url, bodyType, body)
}

// Post ..
func (c *Client) Post(url, bodyType string, body interface{}) (*http.Response, error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(req)
}
