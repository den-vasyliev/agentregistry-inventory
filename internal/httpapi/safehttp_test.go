package httpapi

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateImportURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://registry.example.com/servers.json", false},
		{"http rejected", "http://registry.example.com/servers.json", true},
		{"file scheme rejected", "file:///etc/passwd", true},
		{"missing host", "https://", true},
		{"garbage", "://not a url", true},
		{"metadata over http rejected by scheme", "http://169.254.169.254/", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateImportURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsDisallowedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1",       // loopback
		"::1",             // loopback v6
		"169.254.169.254", // cloud metadata / link-local
		"10.0.0.5",        // private
		"172.16.0.1",      // private
		"192.168.1.1",     // private
		"0.0.0.0",         // unspecified
		"224.0.0.1",       // multicast
		"fc00::1",         // unique-local v6 (private)
		"fe80::1",         // link-local v6
	}
	for _, s := range blocked {
		assert.True(t, isDisallowedIP(net.ParseIP(s)), "expected %s to be blocked", s)
	}
	assert.True(t, isDisallowedIP(nil), "nil IP must be blocked")

	allowed := []string{
		"8.8.8.8",
		"1.1.1.1",
		"93.184.216.34", // example.com
		"2606:2800:220:1:248:1893:25c8:1946",
	}
	for _, s := range allowed {
		assert.False(t, isDisallowedIP(net.ParseIP(s)), "expected %s to be allowed", s)
	}
}
