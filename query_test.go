package main

import (
	"reflect"
	"testing"
)

func TestNewQuery(t *testing.T) {
	testCases := []struct {
		input      string
		components []string
		params     []string
		fail       bool
	}{
		{"plain", []string{"plain"}, []string{}, false},
		{"before <param1> between <param2> end",
			[]string{"before ", " between ", " end"},
			[]string{"param1", "param2"},
			false,
		},
		{"<param1> between <param2>", []string{"", " between ", ""}, []string{"param1", "param2"}, false},
		{"before< badparam", nil, nil, true},
	}

	for _, tc := range testCases {
		query, err := QueryFromString(tc.input)

		if tc.fail {
			if err == nil {
				t.Fatalf("Expected an error, but got none")
			}
		} else if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		} else {
			if !reflect.DeepEqual(query.Components, tc.components) {
				t.Errorf("Expected components %+v, got %+v", tc.components, query.Components)
			}
			if !reflect.DeepEqual(query.Params, tc.params) {
				t.Errorf("Expected params %+v, got %+v", tc.params, query.Params)
			}
		}
	}
}

func TestBuildQuery(t *testing.T) {
	testCases := []struct {
		source string
		params []string
		output string
		fail   bool
	}{
		{"plain", []string{}, "plain", false},
		{"<p1>xyz<p2>", []string{"AAA", "BBB"}, "AAAxyzBBB", false},
		{"<p1><p2>", []string{"AAA", "BBB"}, "AAABBB", false},
		{"<p1><p2><p3>", []string{"AAA", "BBB"}, "AAABBB", true},
		{"<p1><p2>", []string{"AAA"}, "AAA", true},
	}
	for _, tc := range testCases {
		query, err := QueryFromString(tc.source)
		if err != nil {
			t.Fatalf("Unable to create query '%s'", tc.source)
		}
		text, err := query.Build(tc.params)
		if tc.fail {
			if err == nil {
				t.Fatalf("Expected an error, but got none")
			}
		} else if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		} else {
			if tc.output != text {
				t.Errorf("Expected build '%s' got %s", tc.output, text)
			}
		}
	}
}
