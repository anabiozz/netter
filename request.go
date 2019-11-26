package netter

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// ReaderFunc ..
type ReaderFunc func() (io.ReadCloser, error)

// Request ..
type Request struct {
	body ReaderFunc
	*http.Request
}

type lenner interface {
	Len() int
}

// NewRequest ..
func NewRequest(method, url string, rawBody interface{}) (*Request, error) {
	bodyReader, contentLength, err := getBodyReader(rawBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.ContentLength = contentLength

	return &Request{bodyReader, httpReq}, nil
}

func getBodyReader(body interface{}) (bodyReader ReaderFunc, contentLength int64, err error) {

	if body != nil {
		switch bodyType := body.(type) {
		case ReaderFunc:
			bodyReader = bodyType
			tmp, err := bodyType()
			if err != nil {
				return nil, 0, err
			}
			if lr, ok := tmp.(lenner); ok {
				contentLength = int64(lr.Len())
			}
			if c, ok := tmp.(io.Closer); ok {
				c.Close()
			}

		case *bytes.Reader:
			bodyReader = func() (io.ReadCloser, error) {
				return ioutil.NopCloser(bodyType), nil
			}
			contentLength = int64(bodyType.Len())

		case *bytes.Buffer:
			buf := bodyType.Bytes()
			bodyReader = func() (io.ReadCloser, error) {
				r := bytes.NewReader(buf)
				return ioutil.NopCloser(r), nil
			}
			contentLength = int64(bodyType.Len())

		case *strings.Reader:
			bodyReader = func() (io.ReadCloser, error) {
				return ioutil.NopCloser(bodyType), nil
			}
			contentLength = int64(bodyType.Len())

		default:
			return nil, 0, fmt.Errorf("cannot handle type %T", bodyType)
		}
	}
	return bodyReader, contentLength, nil
}
