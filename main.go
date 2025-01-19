package main

import (
	"encoding/json"
	"log"
	"encoding/base64"
	"net/url"
	"flag"
	"fmt"
	"os"
	"path"
	"os/exec"
	"strings"
	"time"
	"io/ioutil"
)

const TIME_DATA_CACHE time.Duration = 14 * 24 * time.Hour
const TIME_FAILURE_CACHE time.Duration = 2 * time.Hour

var BASE_OPTIONS = []string { "-A", "Curl via CA v0.1"}

// config contains general app configuration
type Config struct {
	Curl string
	CacheBase string
	ConfigFile string
	HistoryFile string
}

func NewConfig(cache_dir, history_file, config_file string) *Config {
	return &Config {
		Curl: "/usr/bin/curl",
			CacheBase: cache_dir,
			ConfigFile: config_file,
			HistoryFile: history_file,
		}
}

func (c *Config) Load(filename string) error {
	if filename != "" {
		c.ConfigFile = filename
	}

	file, err := os.Open(c.ConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(c)
}

func (c Config) Save(filename string) error {
	if filename != "" {
		c.ConfigFile = filename
	}

	file, err := os.Create(c.ConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(&c)
}


// Query is a helper class for the URL being accessed
type Query struct {
	Url *url.URL
	HostId string
	PathId string
}

func NewQuery(urlstr string) (*Query, error) {
	url, err := url.Parse(urlstr)
	if err != nil {
		return nil, err
	}

	host := url.Scheme + "//" + url.Host
	if url.Port() != "" {
		host += ":" + url.Port()
	}
	path := url.Path
	if path == "" {
		path = "/"
	}

	return &Query {
		Url: url,
			HostId: base64.URLEncoding.EncodeToString([]byte(host)),
			PathId: base64.URLEncoding.EncodeToString([]byte(path)),
		}, nil
}

func (q Query)CacheFile(c *Config) string {
	return path.Join(c.CacheBase, q.HostId, q.PathId)
}

func (q Query)CacheFolder(c *Config) string {
	return path.Join(c.CacheBase, q.HostId)
}

func (q Query)CacheCheck(c *Config) (exists bool, isRecent bool)  {
	filename := q.CacheFile(c)
	info, err := os.Stat(filename)
	if err != nil {
		return false, false
	}

	return true, time.Since(info.ModTime()) > TIME_DATA_CACHE
}

func (q Query)CacheRead(c *Config) ([]byte, error)  {
	filename := q.CacheFile(c)
	return ioutil.ReadFile(filename)
}

func (q Query)CacheWrite(content []byte, c *Config) error {
	folder := q.CacheFolder(c)
	if err := os.MkdirAll(folder, 0700); err != nil {
		return err
	}

	filename := q.CacheFile(c)
	return ioutil.WriteFile(filename, content, 0600)
}


type HistoryEntry struct {
	Time  string
	Query string
}

func Get(q *Query, c *Config, options []string) ([]byte, error) {
	args := append(options, q.Url.String())
	cmd := exec.Command(c.Curl, args...)
	//return cmd.CombinedOutput()
	return cmd.Output()

}


func setup() (cache_dir, history_file, config_file string){
	// set up the relevant folders and
	home, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	cache_dir = path.Join(home, ".cache/ca")
	history_file  = path.Join(home, ".cache/ca/history")
	config_file = path.Join(home, ".config/ca.conf")

	if err := os.MkdirAll(cache_dir, 0700); err != nil {
		log.Panic(err)
	}
	return
}

func usage() {
	fmt.Fprintf(os.Stderr,"Usage:\n" +
		"    %s [--prefix=<prefix>] [--o=<flag>] <query>\n" +
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
	cache_dir, history_file, config_file := setup()

	// Load configuration
	config := NewConfig(cache_dir, history_file, config_file)
	config.Load("")
	config.Save("")

	queryStr, options := parseArgs()
	options = append(options, BASE_OPTIONS...)

	query, err := NewQuery(queryStr)
	if err != nil {
		log.Fatalf("Invalid URLL '%s': %v\n", queryStr, err)
	}

	var content []byte
	cexist, crecent := query.CacheCheck(config)

	if cexist && crecent {
		content, err = query.CacheRead(config)
		if err != nil {
			log.Fatalf("Failed to read content from cache: %v\n", err)
		}
	} else {
		content, err = Get(query, config, options)
		if err != nil {
			if cexist {
				log.Printf("Using old cache due to server failure: %v\n", err)
				content, err = query.CacheRead(config)
				if err != nil {
					log.Fatalf("Failed to read content from cache: %v\n", err)
				}
			} else {
				log.Fatalf("Failed to get content from server: %v\n", err)
			}
		} else {
			err = query.CacheWrite(content, config)
			if err != nil {
				log.Printf("Unable to write to cache: %v\n", err)
			}
		}
	}

	fmt.Printf("%s", string(content))
}
