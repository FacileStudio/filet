package fixture

import (
	"net/http"
	"os"
)

type config struct{ name string }

func load(string) (*config, error) { return &config{}, nil }

func deferBeforeCheck(c *http.Client, req *http.Request) error {
	resp, err := c.Do(req)
	defer resp.Body.Close() // want
	if err != nil {
		return err
	}
	return nil
}

func fieldBeforeCheck() (string, error) {
	cfg, err := load("x")
	name := cfg.name // want
	if err != nil {
		return "", err
	}
	return name, nil
}

func interfaceBeforeCheck() (bool, error) {
	info, err := os.Stat("x")
	dir := info.IsDir() // want
	return dir, err
}

func checkedFirst(c *http.Client, req *http.Request) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func guarded(c *http.Client, req *http.Request) error {
	resp, err := c.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}

func pointerMethod() error {
	f, err := os.Open("x")
	name := f.Name()
	_ = name
	return err
}

func closureNotRun() (func() string, error) {
	cfg, err := load("x")
	get := func() string { return cfg.name }
	return get, err
}

func reassigned() (string, error) {
	cfg, err := load("x")
	cfg = &config{}
	name := cfg.name
	return name, err
}
