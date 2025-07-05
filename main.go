package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
)

const (
	// TimeDataCache is the default age for a cached entry
	TimeDataCache = 60 * 12

	// We need to pretend be curl sometimes
	FakeCurlAgent = "curl/7.54.1"
)

// Config contains general app configuration
type Config struct {
	filename  string
	Templates []string
}

// NewConfig create a new configuration with default parameters
func NewConfig(filename string) *Config {
	return &Config{
		filename:  filename,
		Templates: []string{},
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

// Params contains the parameters used for this call
type Params struct {
	Query      *Query
	Params     []string
	UserAgent  string
	CacheRead  bool
	CacheWrite bool
	MaxAge     int
	Verbose    bool
}

func (p Params) URL() (string, error) {
	return p.Query.Build(p.Params)
}

// parseArgs  will parse command line arguments and build the parameters.
// If a template is used, it will try to get it from the list of templates
func parseArgs(templates []*Template) (*Params, error) {
	agent_ := flag.String("A", FakeCurlAgent, "User Agent, if you don't want to be curl")
	noread_ := flag.Bool("f", false, "Force download (do not read from cache)")
	nowrite_ := flag.Bool("n", false, "Do not write to cache")
	verbose_ := flag.Bool("v", false, "Be verbose")
	maxage_ := flag.Int("age", TimeDataCache, "Max cache age in minutes")

	flag.Usage = usage
	flag.Parse()

	p := &Params{
		UserAgent:  *agent_,
		Verbose:    *verbose_,
		MaxAge:     *maxage_,
		CacheRead:  !*noread_,
		CacheWrite: !*nowrite_,
	}

	args := flag.Args()
	if len(args) == 0 {
		return nil, fmt.Errorf("no query was given")
	}

	p.Params = args[1:]
	base := args[0]
	if base[0] == '@' {
		if err := ApplyTemplate(templates, p, base[1:], len(p.Params)); err != nil {
			return nil, err
		}
	} else {
		var err error
		p.Query, err = QueryFromString(args[0])
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n"+
		"    %s [OPTIONS] <query>\n"+
		"    %s @<builtin> [optional parameter]\n"+
		"Where OPTIONS are:\n",
		os.Args[0], os.Args[0])
	flag.PrintDefaults()

	fmt.Fprintf(os.Stderr, "Example:\n"+
		"    %s https://cht.sh/python/lambda\n"+
		"    %s -age 30 http://ip-api.com\n"+
		"    %s @weather berlin\n",
		os.Args[0], os.Args[0], os.Args[0])
}

func get(params *Params, cache *Cache, request *Request) ([]byte, string, error) {
	exists := false
	recent := false

	if params.CacheRead {
		exists, recent = cache.Check(request, params.MaxAge)
		if exists && recent {
			content, err := cache.Read(request)
			return content, "cached", err
		}
	}

	content, err := request.Download()
	if err != nil {
		// we couldn't download but maybe we happen to have an old copy in the cache?
		if exists {
			log.Printf("Using old cache due to server failure: %v\n", err)
			content, err = cache.Read(request)
			return content, "cache-backup", err
		}
		return nil, "failed", err
	}

	if params.CacheWrite {
		if err := cache.Write(request, content); err != nil {
			log.Printf("Unable to write to cache: %v\n", err)
		}
	}

	return content, "received", nil
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Cannot find our home directory: %v", err)
	}

	// prepare cache
	cache := NewCache(path.Join(home, ".cache/ca"))
	if err := os.MkdirAll(cache.base, 0700); err != nil {
		log.Fatalf("Cannot create cache folder: %v", err)
	}

	// Load configuration
	config := NewConfig(path.Join(home, ".config/ca.conf"))
	if err := config.Load(); err != nil {
		log.Printf("WARNING: failed to load config: %v", err)
	}

	// empty config? lets fill it
	if len(config.Templates) == 0 {
		config.Templates = defaultTemplates
		config.Save()
	}

	// load our templates
	templates := ParseTemplates(config.Templates)

	// get params from command line arguments + templates
	params, err := parseArgs(templates)
	if err != nil {
		fmt.Printf("%v\n", err)
		usage()
		os.Exit(20)
	}

	urlstr, err := params.URL()
	if err != nil {
		log.Fatalf("Could not get URL: %v\n", err)
	}

	request, err := NewRequest(urlstr, params.UserAgent)
	if err != nil {
		log.Fatalf("Cannot build request: %v\n", err)
	}

	content, mode, err := get(params, cache, request)
	if err != nil {
		log.Fatalf("Failed to get content: %v\n", err)
	}

	// print outcome
	fmt.Printf("%s\n", string(content))

	// print mode and cache read/write state
	if params.Verbose {
		fmt.Printf("INFO: mode=%s R=%v W=%v\n", mode, params.CacheRead, params.CacheWrite)
	}

}
