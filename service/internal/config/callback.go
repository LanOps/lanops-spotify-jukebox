package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateAuthCallbackUrl checks a redirect URI against Spotify's rules, which
// Spotify enforces for all apps since 27 November 2025: HTTPS is required,
// except for loopback IP literals (http://127.0.0.1:PORT or http://[::1]:PORT).
// "localhost" is rejected by Spotify.
// See https://developer.spotify.com/documentation/web-api/concepts/redirect_uri
func ValidateAuthCallbackUrl(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("missing host")
	}
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("%q uses localhost, which Spotify no longer allows; use http://127.0.0.1:<port>%s instead (and register that exact URL in the Spotify dashboard)", raw, u.Path)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return nil
		}
		return fmt.Errorf("%q uses http; Spotify requires https unless the host is a loopback IP (127.0.0.1 or [::1])", raw)
	default:
		return fmt.Errorf("%q must use https (or http with a loopback IP)", raw)
	}
}
