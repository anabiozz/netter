package netter

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Since normal ways we would generate a Reader have special cases, use a
// custom type here
type custReader struct {
	val string
	pos int
}

func (c *custReader) Read(p []byte) (n int, err error) {
	if c.val == "" {
		c.val = "hello"
	}
	if c.pos >= len(c.val) {
		return 0, io.EOF
	}
	var i int
	for i = 0; i < len(p) && i+c.pos < len(c.val); i++ {
		p[i] = c.val[i+c.pos]
	}
	c.pos += i
	return i, nil
}

func TestClientDo(t *testing.T) {
	testBytes := []byte("hello")
	// Native func
	testClientDo(t, ReaderFunc(func() (io.Reader, error) {
		return bytes.NewReader(testBytes), nil
	}))
	testClientDo(t, func() (io.Reader, error) {
		return bytes.NewReader(testBytes), nil
	})
	testClientDo(t, testBytes)
	testClientDo(t, bytes.NewBuffer(testBytes))
	testClientDo(t, bytes.NewReader(testBytes))
	testClientDo(t, strings.NewReader(string(testBytes)))
	testClientDo(t, &custReader{})
}

func testClientDo(t *testing.T, body interface{}) {
	// Create a request
	req, err := NewRequest("PUT", "http://127.0.0.1:28934/v1/foo", body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	req.Header.Set("foo", "bar")

	// Track the number of times the logging hook was called
	retryCount := -1

	// Create the client. Use short retry windows.
	client := NewClient()
	client.Retry.WaitMin = 10 * time.Millisecond
	client.Retry.WaitMax = 50 * time.Millisecond
	client.Retry.Max = 50

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.NewSyncLogger(logger)
		logger = level.NewFilter(logger, level.AllowDebug())
		logger = log.With(logger,
			"svc", "uploader",
			"ts", log.DefaultTimestampUTC,
			"caller", log.DefaultCaller,
		)
	}

	// Send the request
	var resp *http.Response
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		var err error
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	select {
	case <-doneCh:
		t.Fatalf("should retry on error")
	case <-time.After(200 * time.Millisecond):
		// Client should still be retrying due to connection failure.
	}

	// Create the mock handler. First we return a 500-range response to ensure
	// that we power through and keep retrying in the face of recoverable
	// errors.
	code := int64(500)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request details
		if r.Method != "PUT" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/v1/foo" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}

		// Check the headers
		if v := r.Header.Get("foo"); v != "bar" {
			t.Fatalf("bad header: expect foo=bar, got foo=%v", v)
		}

		// Check the payload
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		expected := []byte("hello")
		if !bytes.Equal(body, expected) {
			t.Fatalf("bad: %v", body)
		}

		w.WriteHeader(int(atomic.LoadInt64(&code)))
	})

	// Create a test server
	list, err := net.Listen("tcp", ":28934")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer list.Close()
	go http.Serve(list, handler)

	// Wait again
	select {
	case <-doneCh:
		t.Fatalf("should retry on 500-range")
	case <-time.After(200 * time.Millisecond):
		// Client should still be retrying due to 500's.
	}

	// Start returning 200's
	atomic.StoreInt64(&code, 200)

	// Wait again
	select {
	case <-doneCh:
	case <-time.After(time.Second):
		t.Fatalf("timed out")
	}

	if resp.StatusCode != 200 {
		t.Fatalf("exected 200, got: %d", resp.StatusCode)
	}

	if retryCount < 0 {
		t.Fatal("request log hook was not called")
	}
}
