package parse

import "strings"

func IsComment(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "//")
}

func StartsMultilineComment(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "/*")
}

func EndsMultilineComment(line string) (bool, string) {
	exists := strings.Contains(line, "*/")
	return exists, line
}

func Consume(line, prefix string) (bool, string) {
	return false, line
}
