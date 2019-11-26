package netter

import (
	"context"
	"crypto/x509"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type retryer interface {
	Retry(ctx context.Context, resp *http.Response, err error) (bool, error)
	Backoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration
}

var (
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)
	schemeErrorRe    = regexp.MustCompile(`unsupported protocol scheme`)
)

// Retry ..
type Retry struct {
	Max              int
	WaitMin, WaitMax time.Duration
}

// Retry ..
func (Retry) isRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		if v, ok := err.(*url.Error); ok {
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, nil
			}
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, nil
			}
			if schemeErrorRe.MatchString(v.Error()) {
				return false, nil
			}
		}
		return true, nil
	}
	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, nil
	}
	return false, nil
}

// Backoff ..
func (Retry) Backoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	mult := math.Pow(2, float64(attemptNum)) * float64(min)
	sleep := time.Duration(mult)
	if float64(sleep) != mult || sleep > max {
		sleep = max
	}
	return sleep
}
