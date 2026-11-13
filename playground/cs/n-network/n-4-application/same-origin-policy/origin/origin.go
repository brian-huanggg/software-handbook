// Package origin computes the web origin of a URL, the (scheme, host, port)
// triple that browsers use to decide whether two pages trust each other.
package origin

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Origin is a tuple origin. Two origins are the same only when all three
// fields are equal, so plain struct comparison (==) is the right test.
type Origin struct {
	Scheme string
	Host   string
	Port   string
}

// ErrOpaque reports a URL whose scheme has no tuple origin (file:, data:,
// about:...). Browsers give such documents an opaque origin, serialized as
// "null", that is never the same as any other origin, not even itself.
var ErrOpaque = errors.New("origin: opaque origin")

// Browsers fill in the default port, so https://a.test and https://a.test:443
// are the same origin. We only model http and https to keep the example small.
var defaultPorts = map[string]string{"http": "80", "https": "443"}

// Parse returns the origin of rawURL. The path, query and fragment are
// dropped because the browser ignores them when comparing origins.
func Parse(rawURL string) (Origin, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return Origin{}, fmt.Errorf("origin: parse %q: %w", rawURL, err)
	}
	scheme := strings.ToLower(u.Scheme)
	def, ok := defaultPorts[scheme]
	if !ok {
		return Origin{}, fmt.Errorf("%w: scheme %q in %q", ErrOpaque, u.Scheme, rawURL)
	}
	// Host names are case-insensitive, so SHO.RT and sho.rt must compare equal.
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return Origin{}, fmt.Errorf("origin: no host in %q", rawURL)
	}
	port := u.Port()
	if port == "" {
		port = def
	}
	return Origin{Scheme: scheme, Host: host, Port: port}, nil
}

// SameOrigin reports whether o and other are the same origin.
func (o Origin) SameOrigin(other Origin) bool {
	return o == other
}

// String serializes the origin the way browsers send it in the Origin header:
// the default port is omitted and IPv6 addresses keep their brackets.
func (o Origin) String() string {
	host := o.Host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if o.Port == defaultPorts[o.Scheme] {
		return o.Scheme + "://" + host
	}
	return o.Scheme + "://" + host + ":" + o.Port
}
