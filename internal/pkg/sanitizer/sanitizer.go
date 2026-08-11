package sanitizer

import "regexp"

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)("(?:[^"\\]*_)?(?:password|token|secret|authorization|api_key|private_key|code)"\s*:\s*)("(?:\\.|[^"\\])*"|null|true|false|-?\d+(?:\.\d+)?)`),
	regexp.MustCompile(`(?i)(password\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(token\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(secret\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(api_key\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(private_key\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
}

func String(input string) string {
	output := input
	for index, pattern := range sensitivePatterns {
		replacement := `${1}***`
		if index == 0 {
			replacement = `${1}"***"`
		}
		output = pattern.ReplaceAllString(output, replacement)
	}
	return output
}
