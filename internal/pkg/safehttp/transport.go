package safehttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// SafeTransport rejects connections to private or loopback IP ranges to reduce SSRF risk.
var SafeTransport = &http.Transport{
	DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		ip := net.ParseIP(host)
		if ip == nil {
			conn.Close()
			return nil, fmt.Errorf("failed to parse remote IP for %q", addr)
		}

		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			conn.Close()
			return nil, fmt.Errorf("access to private IP %s is denied", ip)
		}

		return conn, nil
	},
}
