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
	"time"
)

const TIME_DATA_CACHE time.Duration = 14 * 24 * time.Hour

var cacheBase, configFile string

// MultiFlag is a flag value that can be added multiple times
type MultiFlag []string

// Set implementation for flag.Value interface
func (m *MultiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

// String implementation for flag.Value interface
func (m MultiFlag) String() string {
	return "[?]"
}

// Config contains general app configuration
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

func (q Query) CacheCheck(maxAge int) (exists bool, isRecent bool) {
	filename := q.CacheFile()
	info, err := os.Stat(filename)
	if err != nil {
		return false, false
	}

	age := time.Since(info.ModTime()).Minutes()
	return true, age <= float64(maxAge)
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
	return cmd.Output()
}

func setup() (cache_dir, config_file string) {
	// set up the relevant folders and
	home, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	cache_dir = path.Join(home, ".cache/ca")
	config_file = path.Join(home, ".config/ca.conf")

	if err := os.MkdirAll(cache_dir, 0700); err != nil {
		log.Panic(err)
	}
	return
}

// Params contains the parameters used for this call
type Params struct {
	Options    []string
	Query      string
	CacheRead  bool
	CacheWrite bool
	MaxAge     int
	Verbose    bool
}

// parseParams will parse command line arguments and build the parameters
func parseParams() *Params {
	var options MultiFlag

	flag.Var(&options, "o", "Additional options for curl")
	prefix_ := flag.String("prefix", "", "Optional query prefix")
	agent_ := flag.String("A", "CA-via-Curl/0.1", "User Agent. Empty -> use Curl UA")
	noread_ := flag.Bool("f", false, "Force download, do not read from cache")
	nowrite_ := flag.Bool("no-write", false, "Do not write to cache")
	verbose_ := flag.Bool("v", false, "be verbose")
	maxage_ := flag.Int("age", 3*24, "max cache age in minutes")

	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	// add base options to user options
	options = append(options, "--silent")
	if *agent_ != "" {
		options = append(options, "-A", *agent_)
	}

	return &Params{
		Options:    options,
		Query:      *prefix_ + flag.Args()[0],
		Verbose:    *verbose_,
		MaxAge:     *maxage_,
		CacheRead:  !*noread_,
		CacheWrite: !*nowrite_,
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n"+
		"    %s [OPTIONS] <query>\n"+
		"Where OPTIONS are:\n",
		os.Args[0])
	flag.PrintDefaults()
}

func main() {
	cacheBase, configFile = setup()

	// Load configuration
	config := NewConfig()
	config.Load(configFile)
	config.Save(configFile)

	params := parseParams()

	query, err := NewQuery(params.Query)
	if err != nil {
		log.Fatalf("Invalid URLL '%s': %v\n", params.Query, err)
	}

	// try get data from cache, if possible and allowed
	var cexist, crecent bool
	if params.CacheRead {
		cexist, crecent = query.CacheCheck(params.MaxAge)
	}

	var content []byte
	var mode string

	if cexist && crecent {
		mode = "cached"
		content, err = query.CacheRead()
		if err != nil {
			log.Fatalf("Failed to read content from cache: %v\n", err)
		}
	} else {
		mode = "received"
		content, err = query.Get(config, params.Options)
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
		} else if params.CacheWrite {
			err = query.CacheWrite(content)
			if err != nil {
				log.Printf("Unable to write to cache: %v\n", err)
			}
		}
	}

	// update mode with cache read/write state
	if params.Verbose {
		fmt.Printf("%s R=%v W=%v", mode, params.CacheRead, params.CacheWrite)
	}

	// print outcome
	fmt.Printf("%s\n", string(content))
}
