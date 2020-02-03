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
	retry(ctx context.Context, resp *http.Response, err error) (bool, error)
	backoff(min, max time.Duration, attemptNum int) time.Duration
}

var (
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)
	schemeErrorRe    = regexp.MustCompile(`unsupported protocol scheme`)
)

type retry struct {
	Max              int
	WaitMin, WaitMax time.Duration
}

func (retry) isRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
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

func (retry) backoff(min, max time.Duration, attemptNum int) time.Duration {
	multiply := math.Pow(2, float64(attemptNum)) * float64(min)
	sleep := time.Duration(multiply)
	if float64(sleep) != multiply || sleep > max {
		sleep = max
	}
	return sleep
}
