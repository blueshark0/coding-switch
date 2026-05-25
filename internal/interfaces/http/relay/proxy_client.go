package relay

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func newInsecureProxyClient(rawProxyURL string) (*http.Client, error) {
	proxyURL, err := url.Parse(rawProxyURL)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy: relayProxyWithNoProxy(proxyURL),
			// Temporary compatibility for local intercepting proxies.
			// #nosec G402 -- explicitly requested for proxied relay requests.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}, nil
}

func relayProxyWithNoProxy(proxyURL *url.URL) func(*http.Request) (*url.URL, error) {
	noProxy := relayNoProxyList()
	return func(req *http.Request) (*url.URL, error) {
		if relayShouldBypassProxy(req.URL.Hostname(), noProxy) {
			return nil, nil
		}
		return proxyURL, nil
	}
}

func relayNoProxyList() []string {
	merged := os.Getenv("NO_PROXY")
	if lower := os.Getenv("no_proxy"); lower != "" {
		if merged == "" {
			merged = lower
		} else {
			merged += "," + lower
		}
	}
	if merged == "" {
		return nil
	}
	parts := strings.Split(merged, ",")
	list := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			list = append(list, part)
		}
	}
	return list
}

func relayShouldBypassProxy(host string, noProxyList []string) bool {
	if host == "" || len(noProxyList) == 0 {
		return false
	}
	for _, rule := range noProxyList {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		if rule == "*" {
			return true
		}
		if idx := strings.IndexByte(rule, ':'); idx >= 0 {
			rule = rule[:idx]
		}
		rule = strings.TrimPrefix(rule, ".")
		if strings.EqualFold(host, rule) {
			return true
		}
		if len(host) > len(rule) && strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(rule)) {
			return true
		}
	}
	return false
}
