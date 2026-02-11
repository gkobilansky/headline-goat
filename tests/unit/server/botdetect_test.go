package server_test

import (
	"testing"

	"github.com/gkobilansky/headline-goat/internal/server"
)

func TestIsBot_KnownBotUserAgents(t *testing.T) {
	botUAs := []struct {
		name string
		ua   string
	}{
		{"Googlebot", "Googlebot/2.1 (+http://www.google.com/bot.html)"},
		{"Bingbot", "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)"},
		{"YandexBot", "Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)"},
		{"Baiduspider", "Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)"},
		{"Facebook", "facebookexternalhit/1.1"},
		{"Twitter", "Twitterbot/1.0"},
		{"Slack", "Slackbot-LinkExpanding 1.0"},
		{"LinkedIn", "LinkedInBot/1.0"},
		{"HeadlessChrome", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36"},
		{"PhantomJS", "Mozilla/5.0 (Unknown; Linux x86_64) AppleWebKit/538.1 (KHTML, like Gecko) PhantomJS/2.1.1 Safari/538.1"},
		{"curl", "curl/7.68.0"},
		{"wget", "Wget/1.21"},
		{"Python requests", "python-requests/2.28.0"},
		{"Go http", "Go-http-client/1.1"},
		{"Scrapy", "Scrapy/2.7.1 (+https://scrapy.org)"},
		{"Semrush", "Mozilla/5.0 (compatible; SemrushBot/7~bl; +http://www.semrush.com/bot.html)"},
		{"Ahrefs", "Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)"},
	}

	for _, tc := range botUAs {
		t.Run(tc.name, func(t *testing.T) {
			if !server.IsBot(tc.ua) {
				t.Errorf("expected %q to be detected as bot", tc.ua)
			}
		})
	}
}

func TestIsBot_RealUserAgents(t *testing.T) {
	realUAs := []struct {
		name string
		ua   string
	}{
		{"Chrome Desktop", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
		{"Safari iPhone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1"},
		{"Firefox Linux", "Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0"},
		{"Edge Windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"},
		{"Samsung Browser", "Mozilla/5.0 (Linux; Android 13; SM-S908U) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115.0.0.0 Mobile Safari/537.36"},
		{"Empty UA", ""},
	}

	for _, tc := range realUAs {
		t.Run(tc.name, func(t *testing.T) {
			if server.IsBot(tc.ua) {
				t.Errorf("expected %q to NOT be detected as bot", tc.ua)
			}
		})
	}
}
