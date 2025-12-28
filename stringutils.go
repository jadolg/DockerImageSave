package dockerimagesave

import "strings"

func Sanitize(s string) string {
	escapedString := strings.ReplaceAll(s, "\n", "")
	escapedString = strings.ReplaceAll(escapedString, "\r", "")
	return RemoveDoubleDots(escapedString)
}

func RemoveDoubleDots(s string) string {
	escapedString := strings.ReplaceAll(s, "..", ".")
	for strings.Contains(escapedString, "..") {
		escapedString = strings.ReplaceAll(escapedString, "..", ".")
	}
	return escapedString
}
