package fixture

import (
	"net/http"
	"time"
)

var shared = &http.Client{} // want

var bounded = &http.Client{Timeout: 10 * time.Second}

var streaming = &http.Client{Timeout: 0}

func literalWithoutTimeout() *http.Client {
	return &http.Client{Transport: http.DefaultTransport} // want
}

func newClient() *http.Client {
	return new(http.Client) // want
}

func zeroValue() *http.Client {
	var c http.Client // want
	return &c
}

func zeroValueTimedLater() *http.Client {
	var c http.Client
	c.Timeout = time.Second
	return &c
}

func timeoutSetLater() *http.Client {
	c := &http.Client{}
	c.Timeout = 5 * time.Second
	return c
}

func packageGet(url string) (*http.Response, error) {
	return http.Get(url) // want
}

func defaultClient(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req) // want
}

func ownClient(c *http.Client, req *http.Request) (*http.Response, error) {
	return c.Do(req)
}
