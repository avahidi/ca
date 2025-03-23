package main

import (
	"bufio"
	"embed"
	"fmt"
	"log"
	"os"
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

// loadEntries will read and convert entries, and filter them by name (if provided)
// file entries look like this:
// name,optional parameter, component 1, ...
func loadEntries(builtins []string, name string) [][]string {
	var ret [][]string

	// Read the file line by line
	for _, line := range builtins {
		line := strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		items := splitAndTrim(line, ",")
		if len(items) < 3 {
			log.Printf("WATNING: invalid line in builtins: '%s'", line)
		} else if name == "" || items[0] == name {
			ret = append(ret, items)
		}
	}
	return ret
}

// given 0 or more items, select the one with the correct number of parameters
func selectOne(itemlist [][]string, count int) ([]string, error) {
	var filtered [][]string
	for _, items := range itemlist {
		if (count == 1 && items[1] == "") || (count == 2 && items[1] != "") {
			filtered = append(filtered, items)
		}
	}

	// see what we got, try to return useful error messages
	switch len(filtered) {
	case 0:
		if len(itemlist) == 0 {
			return nil, fmt.Errorf("no matching builtins found")
		} else if count == 1 {
			return nil, fmt.Errorf("no matching builtins found (one with a parameter exists)")
		} else {
			return nil, fmt.Errorf("no matching builtins found (one without a parameter exist)")
		}
	case 1:
		return filtered[0], nil
	default:
		return nil, fmt.Errorf("multiple matching builtins found")
	}
}

// itemsToArgs convert items in a builtin to commandline args
func itemsToArgs(items []string, extraArgs []string) ([]string, error) {
	all := append(items[2:], extraArgs...)
	flags, query := all[:len(all)-1], all[len(all)-1]

	var args []string
	for _, flag := range flags {
		if strings.HasPrefix(flag, "M=") {
			args = append(args, "-age", flag[2:])
		} else if strings.HasPrefix(flag, "P=") {
			args = append(args, "-prefix", flag[2:])
		} else if strings.HasPrefix(flag, "A=") {
			args = append(args, "-A", flag[2:])
		} else if flag == "F" {
			args = append(args, "-f")
		} else {
			return nil, fmt.Errorf("unknown parameter in builtin: %s", flag)
		}
	}

	// add query to end of our flags and return it as the new command line
	return append(args, query), nil
}

// showBuiltins will dump all builtins
func showBuiltins(builtins []string) {
	itemlist := loadEntries(builtins, "")

	fmt.Printf("Valid items are:\n ")
	for _, item := range itemlist {
		fmt.Printf("\t@%-30s  %v\n", fmt.Sprintf("%s %s", item[0], item[1]), item[2:])
	}
}

// loadBuiltin will load the builtin matching the name and optional arg in params, and return corresponding command line params
func loadBuiltin(params, builtins []string) ([]string, error) {
	name := params[0][1:]

	// help is a special case
	if name == "help" || name == "?" {
		showBuiltins(builtins)
		os.Exit(0)
	}

	itemlist := loadEntries(builtins, name)

	selected, err := selectOne(itemlist, len(params))
	if err != nil {
		return nil, err
	}

	args, err := itemsToArgs(selected, params[1:])
	return args, err
}

func RewriteArgsFromBuiltin(builtins []string) error {
	args, err := loadBuiltin(os.Args[1:], builtins)
	if err != nil {
		return err
	}
	// rewrite arguments so we can parse it like before
	os.Args = append(os.Args[:1], args...)

	return nil
}

// returns true if the command line is of type 'progm @name [optional parameter]'
func ArgsIsBuiltin() bool {
	count := len(os.Args) - 1
	return (count == 1 || count == 2) && os.Args[1][0] == '@'
}
