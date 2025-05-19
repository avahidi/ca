// rak is the opposite of curl
//
// I can already see this on HN front page:
// 1. "Show HN: I implemented curl in 10 lines of Go, I bet Daniel is kicking himself now..."

package main

import (
	"io"
	"net/http"
)

// download returns the query data.
func download(q *Query) ([]byte, error) {
	client := &http.Client{}


	req, err := http.NewRequest("GET", q.Url(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", q.UserAgent())
	req.Header.Add("Accept", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	resp.Body.Close() // do this instead of defer
	return data, err
}
