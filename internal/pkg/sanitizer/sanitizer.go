package sanitizer

import "regexp"

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(token\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(secret\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(api_key\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)(private_key\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`),
}

func String(input string) string {
	output := input
	for _, pattern := range sensitivePatterns {
		output = pattern.ReplaceAllString(output, `${1}***`)
	}
	return output
}
