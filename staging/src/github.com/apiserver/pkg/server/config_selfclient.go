package server

import (
	"fmt"
	"net"
)

// LoopbackHostPort returns the host and port loopback REST clients should use
// to contact the server.
func LoopbackHostPort(bindAddress string) (string, string, error) {
	host, port, err := net.SplitHostPort(bindAddress)
	if err != nil {
		// should never happen
		return "", "", fmt.Errorf("invalid server bind address: %q", bindAddress)
	}

	// Value is expected to be an IP or DNS name, not "0.0.0.0".
	if host == "0.0.0.0" {
		host = "localhost"
		// Get ip of local interface, but fall back to "localhost".
		// Note that "localhost" is resolved with the external nameserver first with Go's stdlib.
		// So if localhost.<yoursearchdomain> resolves, we don't get a 127.0.0.1 as expected.
		addrs, err := net.InterfaceAddrs()
		if err == nil {
			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && ipnet.IP.IsLoopback() {
					host = ipnet.IP.String()
					break
				}
			}
		}
	}
	return host, port, nil
}
