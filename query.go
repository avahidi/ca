package main

import (
	"fmt"
	"strings"
)

// Query represents extracted data from a query string in the format "https://example.com/<param1>/dir/<param2>/..."
type Query struct {
	Source     string
	Components []string
	Params     []string
}

func QueryFromString(source string) (*Query, error) {
	parts := strings.Split(source, "<")
	components := []string{parts[0]}
	params := []string{}

	for _, part := range parts[1:] {
		subparts := strings.SplitN(part, ">", 2)
		if len(subparts) != 2 {
			return nil, fmt.Errorf("missing '>' in '%s'", source)
		}
		params = append(params, subparts[0])
		components = append(components, subparts[1])
	}

	return &Query{Source: source, Components: components, Params: params}, nil
}

func (q Query) Build(params []string) (string, error) {
	if len(params) != len(q.Params) {
		return "", fmt.Errorf("Exteced %d parameters, got %d for '%s'",
			len(q.Params), len(params), q.Source)
	}

	var sb strings.Builder
	for i, _ := range q.Params {
		sb.WriteString(q.Components[i])
		sb.WriteString(params[i])
	}
	sb.WriteString(q.Components[len(q.Components)-1])
	return sb.String(), nil
}
