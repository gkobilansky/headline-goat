package server

import (
	"strings"
)

// botPatterns contains lowercase substrings found in known bot/crawler User-Agent strings.
var botPatterns = []string{
	"bot", "crawl", "spider", "slurp",
	"facebookexternalhit", "twitterbot", "linkedinbot", "slackbot",
	"whatsapp", "telegrambot", "discordbot",
	"headlesschrome", "phantomjs",
	"curl/", "wget/", "python-requests/", "go-http-client/",
	"scrapy/", "httpie/", "postman",
	"semrush", "ahrefs", "mj12bot", "dotbot", "petalbot",
	"yeti/", "baiduspider",
	"archive.org_bot", "ia_archiver",
}

// IsBot returns true if the given User-Agent string matches known bot patterns.
// Empty User-Agent strings are allowed through (sendBeacon may not set UA).
func IsBot(ua string) bool {
	if ua == "" {
		return false
	}
	lower := strings.ToLower(ua)
	for _, pattern := range botPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}
