package netter

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var robotsTxtHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Last-Modified", "sometime")
	_, err := fmt.Fprintf(w, "User-agent: go\nDisallow: /something/")
	if err != nil {
		fmt.Println(err)
	}
})

func TestClient(t *testing.T) {
	ts := httptest.NewServer(robotsTxtHandler)
	defer ts.Close()

	client := NewClient()
	client.Max = 4
	client.WaitMin = 1 * time.Second
	client.WaitMax = 10 * time.Second

	res, err := client.Get(ts.URL)
	var bytes []byte

	if err == nil {
		bytes, err = pedanticReadAll(res.Body)
		t.Log(string(bytes))
		err := res.Body.Close()
		if err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Error(err)
	} else if s := string(bytes); !strings.HasPrefix(s, "User-agent:") {
		t.Errorf("incorrect page body (did not begin with User-agent: %q", s)
	}
}

func pedanticReadAll(r io.Reader) (b []byte, err error) {
	var buffer [64]byte
	buf := buffer[:]
	for {
		n, err := r.Read(buf)
		if n == 0 && err == nil {
			return nil, fmt.Errorf("read: n=0 with err=nil")
		}
		b = append(b, buf[:n]...)
		if err == io.EOF {
			n, err := r.Read(buf)
			if n != 0 || err != io.EOF {
				return nil, fmt.Errorf("read: n=%d err=%#v after EOF", n, err)
			}
			return b, nil
		}
		if err != nil {
			return b, nil
		}
	}
}

//type clientServerTest struct {
//	t  *testing.T
//	h2 bool
//	h  http.Handler
//	ts *httptest.Server
//	tr *http.Transport
//	c  *Client
//}
//
//func newClientServerTest() {
//
//}
//
//type reader struct {
//	val string
//	pos int
//}
//
//func (c *reader) Read(p []byte) (n int, err error) {
//	if c.val == "" {
//		c.val = "hello"
//	}
//	if c.pos >= len(c.val) {
//		return 0, io.EOF
//	}
//	var i int
//	for i = 0; i < len(p) && i+c.pos < len(c.val); i++ {
//		p[i] = c.val[i+c.pos]
//	}
//	c.pos += i
//	return i, nil
//}
//
//func TestClientDo(t *testing.T) {
//
//	testBytes := []byte("hello")
//
//	var testCases = []struct {
//		body interface{}
//	}{
//		{
//			ReaderFunc(func() (io.Reader, error) {
//				return bytes.NewReader(testBytes), nil
//			}),
//		},
//		{
//			func() (io.Reader, error) {
//				return bytes.NewReader(testBytes), nil
//			},
//		},
//		{
//			testBytes
//		},
//		{
//			bytes.NewBuffer(testBytes)
//		},
//		{
//			bytes.NewReader(testBytes)
//		},
//		{
//			strings.NewReader(string(testBytes))
//		},
//		{
//			strings.NewReader(string(testBytes))
//		},
//		{
//			&reader{}
//		}
//	}
//
//	for _, test := range testCases {
//		t.Run(test.desc, func(t *testing.T) {
//			t.Parallel()
//
//		})
//	}
//
//}
//
//func testClientDo(t *testing.T, body interface{}) {
//	request, err := NewRequest("PUT", "http://127.0.0.1:28934/v1/foo", body)
//	if err != nil {
//		t.Fatalf("err: %v", err)
//	}
//	request.Header.Set("foo", "bar")
//
//	retryCount := -1
//
//	client := NewClient()
//	client.Retry.WaitMin = 1 * time.Second
//	client.Retry.WaitMax = 10 * time.Second
//	client.Retry.Max = 10
//
//	var response *http.Response
//	donech := make(chan struct{})
//
//	go func() {
//		defer close(donech)
//		var err error
//		response, err = client.Do(request)
//		if err != nil {
//			t.Fatalf("err: %v", err)
//		}
//	}()
//
//	select {
//	case <-donech:
//		t.Fatalf("should retry on error")
//	case <-time.After(200 * time.Millisecond):
//	}
//
//	code := int64(500)
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//
//		if r.Method != "PUT" {
//			t.Fatalf("bad method: %s", r.Method)
//		}
//
//		if r.RequestURI != "/v1/foo" {
//			t.Fatalf("bad uri: %s", r.RequestURI)
//		}
//
//		if v := r.Header.Get("foo"); v != "bar" {
//			t.Fatalf("bad header: expect foo=bar, got foo=%v", v)
//		}
//
//		body, err := ioutil.ReadAll(r.Body)
//		if err != nil {
//			t.Fatalf("err: %s", err)
//		}
//
//		expected := []byte("hello")
//		if !bytes.Equal(body, expected) {
//			t.Fatalf("bad: %v", body)
//		}
//
//		w.WriteHeader(int(atomic.LoadInt64(&code)))
//	})
//
//	listen, err := net.Listen("tcp", ":28934")
//	if err != nil {
//		t.Fatalf("err: %v", err)
//	}
//	defer listen.Close()
//
//	go http.Serve(listen, handler)
//
//	select {
//	case <-donech:
//		t.Fatalf("should retry on 500-range")
//	case <-time.After(200 * time.Millisecond):
//	}
//
//	atomic.StoreInt64(&code, 200)
//
//	select {
//	case <-donech:
//	case <-time.After(time.Second):
//		t.Fatalf("timed out")
//	}
//
//	if response.StatusCode != 200 {
//		t.Fatalf("exected 200, got: %d", response.StatusCode)
//	}
//
//	if retryCount < 0 {
//		t.Fatal("request log hook was not called")
//	}
//}
