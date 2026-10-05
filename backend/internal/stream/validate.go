package stream

import (
	"net"
	"net/url"
	"strings"
)

// UserError is an error whose message is safe and useful to show to the end user.
type UserError struct{ Msg string }

func (e *UserError) Error() string { return e.Msg }

// ValidateURL accepts only rtsp:// and rtsps:// URLs. Because the scheme is checked here and the URL is
// passed to FFmpeg as a single argument after -i, a caller cannot make FFmpeg read local files or
// smuggle in extra command-line options.
func ValidateURL(raw string, allowedHosts []string, blockPrivate bool) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, &UserError{"No stream URL was provided."}
	}
	if len(raw) > 2048 {
		return nil, &UserError{"The stream URL is too long."}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, &UserError{"That does not look like a valid URL."}
	}
	switch strings.ToLower(u.Scheme) {
	case "rtsp", "rtsps":
	default:
		return nil, &UserError{"Only rtsp:// and rtsps:// URLs are supported."}
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return nil, &UserError{"The URL is missing a host name."}
	}
	if len(allowedHosts) > 0 && !contains(allowedHosts, host) {
		return nil, &UserError{"This server only allows streams from approved hosts."}
	}
	if blockPrivate && resolvesToPrivate(host) {
		return nil, &UserError{"This server does not allow private or local network addresses."}
	}
	return u, nil
}

// Redact returns the URL with any credentials removed, for logs.
func Redact(u *url.URL) string {
	c := *u
	if c.User != nil {
		c.User = url.User("redacted")
	}
	return c.String()
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), v) {
			return true
		}
	}
	return false
}

func resolvesToPrivate(host string) bool {
	ips, err := net.LookupIP(host)
	if err != nil {
		return false // unresolvable hosts fail later with a clear FFmpeg error
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return true
		}
	}
	return false
}
