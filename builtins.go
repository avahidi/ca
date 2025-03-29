package main

import (
	"bufio"
	"embed"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

//go:embed builtins.txt
var builtins embed.FS

// defaultBuiltins will return the default list of builtins
func defaultBuiltins() []string {
	var ret []string

	file, err := builtins.Open("builtins.txt")
	if err != nil {
		log.Fatalf("INTERNAL ERROR: failed when reading buildints: %v\n", err)
	}
	defer file.Close()

	// Create a new scanner for the file
	scanner := bufio.NewScanner(file)

	// Read the file line by line
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		ret = append(ret, line)
	}
	return ret
}

// split function that also times the result
func splitAndTrim(str, sep string) []string {
	strs := strings.Split(str, sep)
	for i, str := range strs {
		strs[i] = strings.TrimSpace(str)
	}
	return strs
}

// parseTemplates will read and convert entries. Each entry looks like this
// name,optional parameter, component 1, ...
func parseTemplates(builtins []string) [][]string {
	var ret [][]string

	// Read the file line by line
	for _, line := range builtins {
		line := strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		items := splitAndTrim(line, ",")
		if len(items) < 3 {
			log.Printf("WARNING: invalid line in builtins: '%s'", line)
		} else {
			ret = append(ret, items)
		}
	}
	return ret
}

// findTemplate finds the template with given name and possibly param
func findTemplate(itemlist [][]string, name string, hasParam bool) ([]string, error) {
	// 1. first finds the ones that match this target
	var filterName [][]string
	for _, items := range itemlist {
		if items[0] == name {
			filterName = append(filterName, items)
		}
	}

	if len(filterName) == 0 {
		return nil, fmt.Errorf("no matching builtins found")
	}

	// 2. see if any of those has the correct number of args
	var filterParam [][]string
	for _, items := range filterName {
		if hasParam == (items[1] != "") {
			filterParam = append(filterParam, items)
		}
	}

	// see what we got, try to return useful error messages
	switch len(filterParam) {
	case 0:
		if hasParam {
			return nil, fmt.Errorf("no matching builtins found (one without a parameter exist)")
		} else {
			return nil, fmt.Errorf("no matching builtins found (one with a parameter exists)")
		}
	case 1:
		return filterParam[0], nil
	default:
		return nil, fmt.Errorf("multiple matching builtins found")
	}
}

func applyTemplateItem(p *Params, item string) error {
	if strings.HasPrefix(item, "M=") {
		n, err := strconv.ParseInt(item[2:], 10, 32)
		if err != nil {
			return err
		}
		p.MaxAge = int(n)
	} else if strings.HasPrefix(item, "P=") {
		p.Prefix = item[2:]
	} else if strings.HasPrefix(item, "A=") {
		p.UserAgent = item[2:]
	} else if item == "F" {
		p.CacheRead = false
	} else {
		return fmt.Errorf("unknown parameter in builtin: %s", item)
	}
	return nil
}

// showBuiltins will dump all builtins
func showBuiltins(builtins []string) {
	itemlist := parseTemplates(builtins)

	fmt.Printf("Valid items are:\n ")
	for _, item := range itemlist {
		fmt.Printf("\t@%-30s  %v\n", fmt.Sprintf("%s %s", item[0], item[1]), item[2:])
	}
}

func LoadFromBuiltin(p *Params, builtins, args []string) error {
	name := args[0][1:]

	// help is a special case
	if name == "help" || name == "?" {
		showBuiltins(builtins)
		os.Exit(0)
	}

	itemlist := parseTemplates(builtins)
	template, err := findTemplate(itemlist, name, len(args) != 1)
	if err != nil {
		return err
	}

	if len(args) == 2 {
		p.Query = args[1]
	} else {
		p.Query = template[len(template)-1]
		template = template[:len(template)-1]
	}
	for _, s := range template[2:] {
		if err := applyTemplateItem(p, s); err != nil {
			return err
		}
	}

	return nil
}

// returns true if the command line is of type 'progm @name [optional parameter]'
func ArgsIsBuiltin(args []string) bool {
	count := len(args)
	return (count == 1 || count == 2) && args[0][0] == '@'
}
