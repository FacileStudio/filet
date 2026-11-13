package fixture

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

func discardedRows(db *sql.DB) {
	db.Query("SELECT 1") // want
}

func blankRows(db *sql.DB) error {
	_, err := db.Query("SELECT 1") // want
	return err
}

func blankFile() error {
	_, err := os.Create("x") // want
	return err
}

func keptRows(db *sql.DB) error {
	rows, err := db.Query("SELECT 1")
	if err != nil {
		return err
	}
	return rows.Close()
}

func bodyNeverClosed(c *http.Client, req *http.Request) ([]byte, error) {
	resp, err := c.Do(req) // want
	if err != nil {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}

func bodyOnlyDecoded(c *http.Client, req *http.Request, v any) error {
	resp, err := c.Do(req) // want
	if err != nil {
		return err
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func bodyClosed(c *http.Client, req *http.Request) ([]byte, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func responseReturned(c *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func bodyReturned(c *http.Client, req *http.Request) (io.ReadCloser, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func bodyWrapped(c *http.Client, req *http.Request) (io.ReadCloser, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(resp.Body), nil
}

func bodyDrained(c *http.Client, req *http.Request) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	drainAndClose(resp.Body)
	return nil
}

func bodyDeferredHelper(c *http.Client, req *http.Request) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer release(resp.Body)
	return nil
}

func responseHandedOff(c *http.Client, req *http.Request) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	return consume(resp)
}

func namedResult(c *http.Client, req *http.Request) (resp *http.Response, err error) {
	resp, err = c.Do(req)
	return
}

func drainAndClose(rc io.ReadCloser) {
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()
}

func release(rc io.ReadCloser) { _ = rc.Close() }

func consume(resp *http.Response) error { return resp.Body.Close() }
