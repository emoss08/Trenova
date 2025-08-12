package grpcutils

import (
	"net"
	"strings"
)

func NormalizeDialTarget(listenAddr string) string {
	if strings.HasPrefix(listenAddr, ":") {
		return "127.0.0.1" + listenAddr
	}
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		// if split fails, return as-is
		return listenAddr
	}
	h := strings.TrimSpace(host)
	if h == "" || h == "0.0.0.0" || h == "::" || h == "[::]" {
		return net.JoinHostPort("127.0.0.1", port)
	}
	return listenAddr
}
