// Package main provides a simple command-line tool for making HTTP requests
// with caching capabilities.
package main

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
)

// Request is a helper class for the URL being accessed
// It also implements CachedLocation so we can put each request in a cache
type Request struct {
	userAgent string
	url       string
	hostID    string
	pathID    string
}

func (r Request) Location() (string, string) {
	return r.hostID, r.pathID
}

// Download returns the query data.
func (q Request) Download() ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", q.url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", q.userAgent)
	req.Header.Add("Accept", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	resp.Body.Close() // do this instead of defer
	return data, err
}

// NewRequest creates a new Request from an url
func NewRequest(urlstr, userAgent string) (*Request, error) {
	url, err := url.Parse(urlstr)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "" {
		url.Scheme = "https"
	}
	host := url.Scheme + "//" + url.Host
	if url.Port() != "" {
		host += ":" + url.Port()
	}
	path := url.Path
	if path == "" {
		path = "/"
	}

	return &Request{
		url:       url.String(),
		userAgent: userAgent,
		hostID:    base64.URLEncoding.EncodeToString([]byte(host)),
		pathID:    base64.URLEncoding.EncodeToString([]byte(path)),
	}, nil
}
