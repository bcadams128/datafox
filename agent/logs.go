package main

import "path/filepath"

type LogBatch struct {
	Lines    []string          `msgpack:"lines"`
	Metadata map[string]string `msgpack:"metadata"`
}

type LogLine struct {
	Log    string
	Source string
}

func discover(globs []string) ([]string, error) {
	seen := make(map[string]struct{})
	var files []string

	for _, g := range globs {
		matches, err := filepath.Glob(g)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			files = append(files, m)
		}
	}
	return files, nil
}
