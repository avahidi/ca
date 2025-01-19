package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

const TIME_DATA_CACHE time.Duration = 14 * 24 * time.Hour
const TIME_FAILURE_CACHE time.Duration = 2 * time.Hour

var BASE_OPTIONS = []string{"--silent", "-A", "Curl via CA v0.1"}

var cacheBase, historyFile, configFile string

// config contains general app configuration
type Config struct {
	Curl string
}

func NewConfig() *Config {
	return &Config{Curl: "/usr/bin/curl"}
}

func (c *Config) Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(c)
}

func (c Config) Save(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(&c)
}

// Query is a helper class for the URL being accessed
type Query struct {
	Url    *url.URL
	HostId string
	PathId string
}

func NewQuery(urlstr string) (*Query, error) {
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

	return &Query{
		Url:    url,
		HostId: base64.URLEncoding.EncodeToString([]byte(host)),
		PathId: base64.URLEncoding.EncodeToString([]byte(path)),
	}, nil
}

func (q Query) CacheFile() string {
	return path.Join(cacheBase, q.HostId, q.PathId)
}

func (q Query) CacheFolder() string {
	return path.Join(cacheBase, q.HostId)
}

func (q Query) CacheCheck() (exists bool, isRecent bool) {
	filename := q.CacheFile()
	info, err := os.Stat(filename)
	if err != nil {
		return false, false
	}

	return true, time.Since(info.ModTime()) < TIME_DATA_CACHE
}

func (q Query) CacheRead() ([]byte, error) {
	filename := q.CacheFile()
	return ioutil.ReadFile(filename)
}

func (q Query) CacheWrite(content []byte) error {
	folder := q.CacheFolder()
	if err := os.MkdirAll(folder, 0700); err != nil {
		return err
	}

	filename := q.CacheFile()
	return ioutil.WriteFile(filename, content, 0600)
}

func (q *Query) Get(c *Config, options []string) ([]byte, error) {
	args := append(options, q.Url.String())
	cmd := exec.Command(c.Curl, args...)
	return cmd.Output() // cmd.CombinedOutput()
}

func recordHistory(q *Query, mode string) error {
	f, err := os.OpenFile(historyFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer f.Close()
	_, err = fmt.Fprintf(f, "%v - %s - %v\n", time.Now().Format(time.RFC3339), mode, q.Url)
	return err
}

func setup() (cache_dir, history_file, config_file string) {
	// set up the relevant folders and
	home, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	cache_dir = path.Join(home, ".cache/ca")
	history_file = path.Join(home, ".cache/ca/history")
	config_file = path.Join(home, ".config/ca.conf")

	if err := os.MkdirAll(cache_dir, 0700); err != nil {
		log.Panic(err)
	}
	return
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n"+
		"    %s [--prefix=<prefix>] [--o=<flag>] <query>\n"+
		"    %s --purge",
		os.Args[0], os.Args[0],
	)
	flag.PrintDefaults()
}

func parseArgs() (query string, options []string) {
	prefix_ := flag.String("prefix", "", "Optional query prefix")
	options_ := flag.String("o", "", "Additional options for curl")
	purge_ := flag.Bool("purge", false, "Purge old cache entries and exit")
	flag.Usage = usage
	flag.Parse()

	if *purge_ {
		log.Panic("purge not implemented")
	}
	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}
	return *prefix_ + flag.Args()[0], strings.Fields(*options_)
}

func main() {
	cacheBase, historyFile, configFile = setup()

	// Load configuration
	config := NewConfig()
	config.Load(configFile)
	config.Save(configFile)

	queryStr, options := parseArgs()
	options = append(options, BASE_OPTIONS...)

	query, err := NewQuery(queryStr)
	if err != nil {
		log.Fatalf("Invalid URLL '%s': %v\n", queryStr, err)
	}

	var content []byte
	var mode string
	cexist, crecent := query.CacheCheck()
	if cexist && crecent {
		mode = "cached"
		content, err = query.CacheRead()
		if err != nil {
			log.Fatalf("Failed to read content from cache: %v\n", err)
		}
	} else {
		mode = "received"
		content, err = query.Get(config, options)
		if err != nil {
			if cexist {
				mode = "cache-backup"
				log.Printf("Using old cache due to server failure: %v\n", err)
				content, err = query.CacheRead()
				if err != nil {
					log.Fatalf("Failed to read content from cache: %v\n", err)
				}
			} else {
				log.Fatalf("Failed to get content from server: %v\n", err)
			}
		} else {
			err = query.CacheWrite(content)
			if err != nil {
				log.Printf("Unable to write to cache: %v\n", err)
			}
		}
	}
	recordHistory(query, mode)
	fmt.Printf("%s", string(content))
}
