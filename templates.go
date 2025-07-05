package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// defaultTemplates contains the built-in targets
var defaultTemplates []string = []string{
	"go,https://cht.sh/go/<entry>",
	"rust,https://cht.sh/rust/<entry>",
	"news,M=30,http://getnews.tech",
	"ip,F=1,https://ifconfig.me",
	"city,ifconfig.co/city",
	"weather,M=120,wttr.in/",
	"weather,M=120,wttr.in/<city>",
	"eth,M=60,rate.sx/ETH",
	"btc,M=60,rate.sx/BTC",
	"whois,F=1,ipinfo.io/<what>",
	"qrcode,M=99999,qrenco.de/<item>",
}

type Template struct {
	Name   string
	Input  string
	Query  *Query
	Config map[byte]string
}

func TemplateFromString(source string) (*Template, error) {
	validConfigs := "FMA"

	// split and trim
	parts := strings.Split(source, ",")
	for i, str := range parts {
		parts[i] = strings.TrimSpace(str)
	}

	if len(parts) < 2 {
		return nil, fmt.Errorf("Not a valid template: '%s'", source)
	}

	query, err := QueryFromString(parts[len(parts)-1])
	if err != nil {
		return nil, err
	}

	t := &Template{
		Name:   parts[0],
		Input:  parts[len(parts)-1],
		Query:  query,
		Config: make(map[byte]string),
	}

	for _, part := range parts[1 : len(parts)-1] {
		if len(part) < 3 || part[1] != '=' || strings.Index(validConfigs, part[:1]) == -1 {
			return nil, fmt.Errorf("Unknown configuration '%s' in template'%s'", part, source)
		}
		t.Config[part[0]] = part[2:] // <config> = <value>
	}

	return t, nil
}

func (t Template) Apply(p *Params) error {
	p.Query = t.Query

	for k, v := range t.Config {
		switch k {
		case 'A':
			p.UserAgent = v
		case 'F':
			p.CacheRead = (v == "1" || v == "true")
		case 'M':
			n, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				return err
			}
			p.MaxAge = int(n)

		default: // this should not happen as we check configs when creating Template...
			return fmt.Errorf("Unknown template configuration in '%s'", t.Input)
		}
	}
	return nil
}

// ApplyTemplate finds the template with given name and possibly param
func ApplyTemplate(templates []*Template, p *Params, name string, paramCount int) error {
	if name == "help" || name == "?" {
		ShowTemplates(templates)
		os.Exit(0)
	}

	var partialMatch *Template
	for _, t := range templates {
		if t.Name == name {
			if len(t.Query.Params) == paramCount {
				return t.Apply(p)
			}
			partialMatch = t
		}
	}

	if partialMatch != nil {
		return fmt.Errorf("Template '%s' found but requires parameters %+v", name, partialMatch.Query.Params)
	}
	return fmt.Errorf("Template '%s' not found", name)
}

func ParseTemplates(templates []string) []*Template {
	var ret []*Template
	for _, line := range templates {
		t, err := TemplateFromString(line)
		if err != nil {
			log.Printf("WARNING - invalid template: '%s'", line)
		} else {
			ret = append(ret, t)
		}
	}
	return ret
}

// showBuiltins will dump all builtins
func ShowTemplates(templates []*Template) {
	fmt.Printf("Valid templates are:\n ")
	for _, t := range templates {
		fmt.Printf("\t@%-30s  %+v\n", t.Name, t.Query.Params)
	}
}
