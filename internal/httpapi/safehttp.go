package httpapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// errBlockedAddress is returned by the dial control when a connection target
// resolves to a disallowed (private/loopback/link-local/metadata) address.
var errBlockedAddress = fmt.Errorf("connection to non-public address blocked")

// validateImportURL enforces an https-only scheme on a user-supplied import
// source URL. Per-IP filtering happens later in the dial control, after DNS
// resolution, so that a hostname cannot smuggle a private target past us.
func validateImportURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("only https URLs are allowed (got %q)", u.Scheme)
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("URL must include a host")
	}
	return u, nil
}

// isDisallowedIP reports whether an IP must not be dialed by the server-side
// fetcher. It blocks loopback, link-local (incl. the cloud metadata range
// 169.254.0.0/16), private RFC1918/ULA, unspecified, and multicast addresses.
func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast()
}

// safeDialControl is a net.Dialer Control hook that runs after DNS resolution
// with the concrete IP:port about to be connected. It rejects any address that
// resolves to a non-public IP, defeating DNS-rebinding and metadata-endpoint
// SSRF (e.g. http://169.254.169.254/, http://localhost, internal services).
func safeDialControl(_ string, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if isDisallowedIP(ip) {
		return errBlockedAddress
	}
	return nil
}

// newSafeHTTPClient builds an http.Client suitable for fetching untrusted,
// user-supplied URLs. It blocks non-public targets at dial time and refuses to
// follow redirects (a redirect could otherwise point at an internal address
// after the initial allow check).
func newSafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: safeDialControl,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		// No proxy from environment; the request target is the only egress.
		Proxy:                 nil,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
