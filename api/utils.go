package api

import (
	"net"
	"net/http"
	"strings"
)

// getClientIP returns the client IP address from X-Forwarded-For when
// present, or falls back to the remote address on the request.
func getClientIP(r *http.Request) string {
	// check behind-proxy IP
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// check direct IP
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
