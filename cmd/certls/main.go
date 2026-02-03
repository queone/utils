package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	program_name    = "certls"
	program_version = "2.0.0"
)

func init() {
	_ = program_name
	_ = program_version
}

func usage() {
	fmt.Println("Print SSL certificate details for given FQDN:Port.")
	fmt.Println()
	fmt.Printf("Usage: %s FQDN[:PORT]\n", filepath.Base(os.Args[0]))
	fmt.Println("  Examples:")
	fmt.Printf("    %s microsoft.com     Uses 443 by default\n", filepath.Base(os.Args[0]))
	fmt.Printf("    %s mysite.com:1473   Uses port 1473\n", filepath.Base(os.Args[0]))
	os.Exit(1)
}

func parseTarget(arg string) (string, string) {
	if strings.Count(arg, ":") > 1 {
		usage()
	}

	host, port, err := net.SplitHostPort(arg)
	if err == nil {
		return host, port
	}

	// No port specified
	if strings.Contains(arg, ":") {
		usage()
	}
	return arg, "443"
}

func main() {
	if len(os.Args) != 2 {
		usage()
	}

	host, port := parseTarget(os.Args[1])
	if host == "" {
		usage()
	}

	address := net.JoinHostPort(host, port)

	conn, err := tls.Dial("tcp", address, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "TLS connection failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		fmt.Fprintln(os.Stderr, "No certificates presented by server")
		os.Exit(1)
	}

	cert := state.PeerCertificates[0]

	fmt.Printf("==> Go TLS version %s\n", runtimeVersion())
	fmt.Printf("==> FQDN:Port %s:%s\n", host, port)
	fmt.Printf("==> EXPIRY: NotBefore=%s NotAfter=%s\n",
		cert.NotBefore.Format(time.RFC3339),
		cert.NotAfter.Format(time.RFC3339),
	)

	fmt.Println("==> LIST")
	for _, dns := range cert.DNSNames {
		fmt.Println(dns)
	}
}

func runtimeVersion() string {
	if v := os.Getenv("GOVERSION"); v != "" {
		return v
	}
	return strings.TrimPrefix(runtime.Version(), "go")
}
