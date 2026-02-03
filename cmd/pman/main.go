package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
	program_name    = "pman"
	program_version = "2.0.0"
)

func printUsage() {
	fmt.Printf("%s Azure REST API Caller v%s\n", program_name, program_version)
	fmt.Println("  This utility relies on the 'azm' command-line utility being installed,")
	fmt.Println("  authenticated, and properly configured to obtain Azure access tokens.")
	fmt.Println("  It uses azm to retrieve Microsoft Graph or Azure Resource Manager tokens")
	fmt.Println("  based on the target endpoint.")
	fmt.Println()
	fmt.Println("  Usage Examples:")
	fmt.Printf("    %s GET \"https://graph.microsoft.com/v1.0/me\"\n", program_name)
	fmt.Printf("    %s GET \"https://management.azure.com/subscriptions?api-version=2022-04-01\"\n", program_name)
	fmt.Printf("    %s GET \"https://graph.microsoft.com/v1.0/applications/<id>\"\n", program_name)
	os.Exit(1)
}

func checkBinary(name string) {
	if _, err := exec.LookPath(name); err != nil {
		fmt.Fprintf(os.Stderr, "Missing '%s' binary!\n", name)
		os.Exit(1)
	}
}

func getToken(url string) string {
	var cmd *exec.Cmd

	switch {
	case strings.Contains(url, "https://graph.microsoft.com"):
		cmd = exec.Command("azm", "-tmg")
	case strings.Contains(url, "https://management.azure.com"):
		cmd = exec.Command("azm", "-taz")
	default:
		printUsage()
	}

	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to obtain token: %v\n", err)
		os.Exit(1)
	}

	token := strings.TrimSpace(string(out))

	if !strings.HasPrefix(token, "eyJ") {
		fmt.Fprintln(os.Stderr) // blank line
		fmt.Fprintln(os.Stderr, "WARNING: Token string is invalid!")
		fmt.Fprintln(os.Stderr) // blank line
	}

	return token
}

func main() {
	checkBinary("azm")

	if len(os.Args) < 3 {
		printUsage()
	}

	method := strings.ToUpper(os.Args[1])
	url := os.Args[2]
	extraArgs := os.Args[3:]

	token := getToken(url)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Request creation failed: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Handle optional curl-style flags minimally:
	// Only supports: -d / --data
	for i := 0; i < len(extraArgs); i++ {
		if extraArgs[i] == "-d" || extraArgs[i] == "--data" {
			if i+1 >= len(extraArgs) {
				fmt.Fprintln(os.Stderr, "Missing value for -d/--data")
				os.Exit(1)
			}
			req.Body = io.NopCloser(bytes.NewBufferString(extraArgs[i+1]))
			i++
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HTTP request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Print(string(body))
}
