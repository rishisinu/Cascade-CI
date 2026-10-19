package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"


)

const (
	maxReadLines = 400
	maxOutputBytes = 20000
    maxGrepMatches = 100
    maxGrepFile    = 1 << 20 // skip files over 1 MiB, they're almost never source
)

var skipdirs = []string{".git","node_modules","vendor"}

type readFile struct{repo string}

func (readFile) Spec() Spec {
	return Spec{
		Name: "read_file",
		Description: "Read a file from the repository, with line numbers. Path is relative to the repo root." +
						"Reads at most 400 lines per call ; use start_kline to page through longer files",
		Properties: map[string]any{
			"path": map[string]any{"type": "string"},
			"start_line": "start_line": map[string]any{"type": "integer", "description": "First line to return, 1-based. Default 1."},
			"end_line": "end_line":   map[string]any{"type": "integer", "description": "Last line to return, inclusive."},
			Required: []string{"path"},



		},
	}
}
//The reason we name the reciever here (the self) is so we can access the obj opposed to the one above, where we just call (readfile)
func (t readFile) Run(_ context.Context, input json.RawMessage) (string, error){
	//Create anon struct just for this func
	var in struct {
                Path      string `json:"path"`
                StartLine int    `json:"start_line"`
                EndLine   int    `json:"end_line"`
    }
    if err := json.Unmarshal(input, &in); err != nil{
    	return "", err
    }
    path, err := resolvePath(t.repo, in.Path)
    if err != nil{
    	return "", err
    }

    data, err := os.ReadFile(path)

    if err != nil{
    return "", fmt.Errorf("couldnt read file: %w", err)
    }
    //Turn raw bytes reeturned from readfile to strings
    lines := strings.Split(string(data), "\n")
    if lines[len(lines)-1] == ""{
    	lines = lines[:len(lines)-1]
    }
    start := in.StartLine
    end := in.EndLine
    if end == 0 || end > len(lines){
    	end = len(lines)
    }
    end = min(end, start + maxReadLines - 1)
    if start > len(lines){
    return "", fmt.Errorf("Start(%d)was after the max length of the file, it was len: %d lines", start, len(lines))
    }

    var b strings.Builder
    for i:= start; i<= end; i++ {
    	fmt.Fprintf(&b, "%5d| %s\n", i, lines[i-1])
    }
    if end < len(lines){
    	fmt.Fprintf(&b, "...[showing lines %d-%d of %d; call read_file with start_line=%d for more]\n", start, end, len(lines), end+1)
    }
    return b.String(), nil

}



type grep struct{repo string}

func (grep) Spec() Spec {
	return Spec{
		Name: "grep"
		Description: "allows you to search for a certain pattern/regex in a file. Returns file:line: text " +
																													"this works for up to 100 matches, so you can use to find function callers etc",
	    Properties: map[string]any{
						"pattern": map[string]any{"type":"string"}
						"path": map[string]any{"type":"string", Description: "Directory to search, relative to the repo root. Default: whole repo"}
					},
		Required: []string{"pattern"},
	}
}
