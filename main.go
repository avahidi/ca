package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"time"
)

// TIME_DATA_CACHE is the default age for a cached entry
const TIME_DATA_CACHE time.Duration = 14 * 24 * time.Hour

// Config contains general app configuration
type Config struct {
	filename string
	Curl     string
	Builtins []string
}

// NewConfig create a new configuration with default parameters
func NewConfig(filename string) *Config {
	return &Config{
		filename: filename,
		Curl:     "/usr/bin/curl",
		Builtins: defaultBuiltins(),
	}
}

// Load will try read configuration from disk
func (c *Config) Load() error {
	file, err := os.Open(c.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(c)
}

// Save will write configuration to disk
func (c Config) Save() error {
	bs, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.filename, bs, 0700)
}

// Query is a helper class for the URL being accessed
type Query struct {
	params *Params
	Url    *url.URL
	HostId string
	PathId string
}

// NewQuery creates a new Query from an url
func NewQuery(p *Params) (*Query, error) {
	url, err := url.Parse(p.Prefix + p.Query)
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
		params: p,
		Url:    url,
		HostId: base64.URLEncoding.EncodeToString([]byte(host)),
		PathId: base64.URLEncoding.EncodeToString([]byte(path)),
	}, nil
}

func (q Query) Get(c *Config) ([]byte, error) {
	args := []string{"--silent"}
	if q.params.UserAgent != "" {
		args = append(args, "-A", q.params.UserAgent)
	}
	args = append(args, q.Url.String())
	if q.params.Verbose {
		fmt.Printf("INFO: calling curl with '%v'\n", args)
	}

	cmd := exec.Command(c.Curl, args...)
	return cmd.Output()
}

// Cache represents our cache system
type Cache struct {
	base string
}

func NewCache(base string) *Cache {
	return &Cache{base: base}
}

func (c Cache) file(q Query) string {
	return path.Join(c.base, q.HostId, q.PathId)
}

func (c Cache) folder(q Query) string {
	return path.Join(c.base, q.HostId)
}

func (c Cache) Check(q Query, maxAge int) (exists bool, isRecent bool) {
	info, err := os.Stat(c.file(q))
	if err != nil {
		return false, false
	}

	age := time.Since(info.ModTime()).Minutes()
	return true, age <= float64(maxAge)
}

func (c Cache) Read(q Query) ([]byte, error) {
	return os.ReadFile(c.file(q))
}

func (c Cache) Write(q Query, content []byte) error {
	folder := c.folder(q)
	if err := os.MkdirAll(folder, 0700); err != nil {
		return err
	}

	filename := c.file(q)
	return os.WriteFile(filename, content, 0600)
}

// Application represents the application contect
type Application struct {
	Cache  *Cache
	Config *Config
}

func NewApplication() *Application {
	// set up the relevant folders and
	home, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	a := &Application{
		Cache:  NewCache(path.Join(home, ".cache/ca")),
		Config: NewConfig(path.Join(home, ".config/ca.conf")),
	}

	if err := os.MkdirAll(a.Cache.base, 0700); err != nil {
		log.Panic(err)
	}
	return a
}

// Params contains the parameters used for this call
type Params struct {
	UserAgent  string
	Query      string
	Prefix     string
	CacheRead  bool
	CacheWrite bool
	MaxAge     int
	Verbose    bool
}

// parseParams will parse command line arguments and build the parameters
// this function will first look for a @builtin and if found rewrite the parameters
func parseParams(builtins []string) *Params {
	prefix_ := flag.String("prefix", "", "Optional query prefix")
	agent_ := flag.String("A", "CA-via-Curl/0.1", "User Agent, set \"\" to use Curl UA")
	noread_ := flag.Bool("f", false, "Force download (do not read from cache)")
	nowrite_ := flag.Bool("n", false, "Do not write to cache")
	verbose_ := flag.Bool("v", false, "Be verbose")
	maxage_ := flag.Int("age", 3*24, "Max cache age in minutes")

	flag.Usage = usage
	flag.Parse()

	p := &Params{
		UserAgent:  *agent_,
		Prefix:     *prefix_,
		Verbose:    *verbose_,
		MaxAge:     *maxage_,
		CacheRead:  !*noread_,
		CacheWrite: !*nowrite_,
	}

	args := flag.Args()
	if ArgsIsBuiltin(args) {
		if err := LoadFromBuiltin(p, builtins, args); err != nil {
			fmt.Printf("%v\n", err)
			flag.Usage()
			os.Exit(1)
		}
	} else {
		if len(args) != 1 {
			flag.Usage()
			os.Exit(1)
		}
		p.Query = args[0]
	}
	return p
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n"+
		"    %s [OPTIONS] <query>\n"+
		"    %s @<builtin> [optional parameter]\n"+
		"Where OPTIONS are:\n",
		os.Args[0], os.Args[0])
	flag.PrintDefaults()
}

func main() {
	app := NewApplication()

	// Load configuration
	config := app.Config
	if err := config.Load(); err != nil {
		log.Printf("WARNING: failed to load config: %v", err)
		config.Save() // config was just created, save it!
	}

	params := parseParams(config.Builtins)

	query, err := NewQuery(params)
	if err != nil {
		log.Fatalf("Cannot build query: %v\n", err)
	}

	// try get data from cache, if possible and allowed
	cache := app.Cache
	var cexist, crecent bool
	if params.CacheRead {
		cexist, crecent = cache.Check(*query, params.MaxAge)
	}

	var content []byte
	var mode string

	if cexist && crecent {
		mode = "cached"
		content, err = cache.Read(*query)
		if err != nil {
			log.Fatalf("Failed to read content from cache: %v\n", err)
		}
	} else {
		mode = "received"
		content, err = query.Get(config)
		if err != nil {
			if cexist {
				mode = "cache-backup"
				log.Printf("Using old cache due to server failure: %v\n", err)
				content, err = cache.Read(*query)
				if err != nil {
					log.Fatalf("Failed to read content from cache: %v\n", err)
				}
			} else {
				log.Fatalf("Failed to get content from server: %v\n", err)
			}
		} else if params.CacheWrite {
			err = cache.Write(*query, content)
			if err != nil {
				log.Printf("Unable to write to cache: %v\n", err)
			}
		}
	}

	// update mode with cache read/write state
	if params.Verbose {
		fmt.Printf("INFO: %s R=%v W=%v\n", mode, params.CacheRead, params.CacheWrite)
	}

	// print outcome
	fmt.Printf("%s\n", string(content))
}
