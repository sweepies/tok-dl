package util

import (
	"regexp"
)

func StringNotEmptyCoalesce(args ...string) string {
	for _, elem := range args {
		if len(elem) > 0 {
			return elem
		}
	}
	return ""
}

func SanitizeFileName(name string) string {
	// Replace invalid Windows characters with underscores
	re := regexp.MustCompile(`[\/\?<>\\:\*\|"]`)
	return re.ReplaceAllString(name, "_")
}
