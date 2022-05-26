package dockerimagesave

import "strings"

func Sanitize(s string) string {
	escapedString := strings.Replace(s, "\n", "", -1)
	escapedString = strings.Replace(escapedString, "\r", "", -1)
	return escapedString
}

func RemoveDoubleDots(s string) string {
	escapedString := strings.ReplaceAll(s, "..", ".")
	for strings.Contains(escapedString, "..") {
		escapedString = strings.ReplaceAll(escapedString, "..", ".")
	}
	return escapedString
}
