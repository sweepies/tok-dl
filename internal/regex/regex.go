package regex

import "regexp"

var (
	// TiktokURL matches valid TikTok URLs from various domains
	TiktokURL = regexp.MustCompile(`^https?:\/\/(?:(?:www|vm|vt|m)\.)?tiktokv?\.com\/.+$`)
)