package parse

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	singleIndex = regexp.MustCompile(`^\s*([a-zA-Z0-9-_]+)\s*$`)
	multiIndex  = regexp.MustCompile(`^\s*([a-zA-Z0-9-_]+)(,\s*[a-zA-Z0-9-_]+)+\s*$`)

	singleIndexWithTable = regexp.MustCompile(`^\s*([a-zA-Z0-9-_]+)\s+([a-zA-Z0-9-_]+)\s*$`)
	multiIndexWithTable  = regexp.MustCompile(`^\s*([a-zA-Z0-9-_]+)(,\s[a-zA-Z0-9-_]+)+\s+([a-zA-Z0-9-_]+)\s*$`)
)

func ParseIndexes(lines []string) (string, error) {
	results := []string{strings.Replace(lines[0], "index", "var _ = doc.RegisterIndexes", 1)}
	lines = lines[1:]

	started := false
	for !started {
		if singleIndexWithTable.MatchString(lines[0]) {
			matches := singleIndexWithTable.FindStringSubmatch(lines[0])[1:]
			results = append(results, matches[1]+"{},")
			results = append(results, fmt.Sprintf("[]string{\"%s\"},", matches[0]))
			started = true
		} else if multiIndexWithTable.MatchString(lines[0]) {
			matches := multiIndexWithTable.FindAllStringSubmatch(lines[0], -1)[0][1:]
			results = append(results, matches[len(matches)-1]+"{},")
			matches = matches[:len(matches)-1]
			cols := "[]string{"
			for i, match := range matches {
				if i < len(matches)-1 {
					cols += "\"" + strings.TrimPrefix(match, ", ") + "\", "
				} else {
					cols += "\"" + strings.TrimPrefix(match, ", ") + "\"},"
				}
			}
			results = append(results, cols)
			started = true
		} else {
			results = append(results, lines[0])
		}
		lines = lines[1:]
	}
	for len(lines) > 0 {
		if singleIndex.MatchString(lines[0]) {
			matches := singleIndex.FindStringSubmatch(lines[0])[1:]
			results = append(results, fmt.Sprintf("[]string{\"%s\"},", matches[0]))
		} else if multiIndex.MatchString(lines[0]) {
			matches := multiIndex.FindAllStringSubmatch(lines[0], -1)[0][1:]
			cols := "[]string{"
			for i, match := range matches {
				if i < len(matches)-1 {
					cols += "\"" + strings.TrimPrefix(match, ", ") + "\", "
				} else {
					cols += "\"" + strings.TrimPrefix(match, ", ") + "\"},"
				}
			}
			results = append(results, cols)
		} else {
			results = append(results, lines[0])
		}
		lines = lines[1:]
	}

	return strings.Join(results, "\n"), nil
}
