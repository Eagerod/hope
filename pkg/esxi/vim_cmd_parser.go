package esxi

import (
	"regexp"
	"strings"
)

var typeNameAfterKeyRegex *regexp.Regexp = regexp.MustCompile(`^(\s*\w+) = \([^\)]+\)`)
var typeNameNoKeyRegex *regexp.Regexp = regexp.MustCompile(`^\s*\([^\)]+\)`)
var keyNameRegex *regexp.Regexp = regexp.MustCompile(`^(\s+)(\w+) =`)

// Super sketchy function that attempts to convert the vim-cmd structured
// output into a JSON-looking string.
// From there, it can be fed into json.Unmarshal.
func VimCmdParseOutput(input string) string {
	outputLines := []string{}
	inputLines := strings.Split(input, "\n")

	outputStarted := false
	for _, l := range inputLines {
		// Some command outputs start with a descriptive line, some don't, so
		//   try to find the first actual line out of output before appending
		//   to the output.
		// Actual output always seems to start with a type name in ().
		if !outputStarted {
			if strings.HasPrefix(l, "(") {
				outputStarted = true
			} else {
				continue
			}
		}

		// Remove type names from the output.
		l = typeNameAfterKeyRegex.ReplaceAllString(l, "$1 = ")
		l = typeNameNoKeyRegex.ReplaceAllString(l, "")
		l = keyNameRegex.ReplaceAllString(l, "$1\"$2\":")
		l = strings.Replace(l, " <unset>", " null", 1)

		outputLines = append(outputLines, l)
	}

	return strings.Join(outputLines, "\n")
}
