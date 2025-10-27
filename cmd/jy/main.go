package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	goyaml "github.com/goccy/go-yaml"
	"github.com/gookit/color"
	"github.com/mattn/go-isatty"
	"github.com/queone/utl"
	"gopkg.in/yaml.v3"
)

const (
	program_name    = "jy"
	program_version = "1.4.6"
)

func printUsage() {
	n := utl.Whi2(program_name)
	v := program_version
	usage := fmt.Sprintf("%s v%s\n"+
		"JSON / YAML converter - https://github.com/queone/utils/blob/main/cmd/jy/README.md\n"+
		"%s\n"+
		"  %s [options] [file]\n"+
		"\n"+
		"  Options can be specified in any order. The file can be piped into the utility, or it\n"+
		"  can be referenced as an argument. If the file is YAML, the output will be JSON, or\n"+
		"  vice versa.\n"+
		"\n"+
		"%s\n"+
		"  -c                     Colorize the output for the specified file.\n"+
		"  -d                     Decolorize the output for piped input or file.\n"+
		"  -?, --help, -h         Show this help message and exit.\n"+
		"\n"+
		"%s\n"+
		"  cat file | %s\n"+
		"  %s /path/to/file\n"+
		"  %s /path/to/file -d\n"+
		"  %s file.yaml -c        Prints a colorized version of the file. Does not convert.\n"+
		"  %s -h\n",
		n, v, utl.Whi2("Usage"), n, utl.Whi2("Options"), utl.Whi2("Examples"), n, n, n, n, n)
	fmt.Print(usage)
	os.Exit(0)
}

func isGitBashOnWindows() bool {
	return runtime.GOOS == "windows" && strings.HasPrefix(os.Getenv("MSYSTEM"), "MINGW")
}

func hasPipedInput() bool {
	stat, _ := os.Stdin.Stat() // Check if anything was piped in
	if isGitBashOnWindows() {
		// Git Bash on Windows handles input redirection differently than other shells. When a program
		// is run without any input or arguments, it still treats the input as if it were piped from an
		// empty stream, causing the program to consider it as piped input and hang. This works around that.
		if !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
			return true
		}
	} else {
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			return true
		}
	}
	return false
}

func printOut(rawBytes []byte, option string) {
	// Check if raw bytes are either a JSON or YAML object
	// JSON must be checked first because it is a subset of the YAML standard
	var rawObject interface{}
	_ = json.Unmarshal(rawBytes, &rawObject) // Is it JSON?
	if rawObject == nil {
		// Is it YAML?
		_ = yaml.Unmarshal(rawBytes, &rawObject)
		if rawObject == nil {
			utl.Die("Not JSON nor YAML\n")
		}
		// It is YAML, print in JSON
		jsonBytes, _ := goyaml.YAMLToJSON(rawBytes)
		jsonBytes2, _ := utl.JsonBytesReindent(jsonBytes, 2) // Two space indent
		if option == "decolor_output" {
			jsonObj, _ := utl.JsonBytesToJsonObj(jsonBytes2)
			utl.PrintJson(jsonObj)
		} else {
			utl.PrintJsonBytesColor(jsonBytes2)
		}
	} else {
		// It is JSON, print in YAML
		if option == "decolor_output" {
			utl.PrintYaml(rawObject)
		} else {
			utl.PrintYamlColor(rawObject)
		}
	}
}

func processPipedInput(option string) {
	// Read piped input and convert to decolorized raw bytes
	rawBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading from stdin:", err)
	}

	// Remove color codes in piped input
	stringSansColor := color.ClearCode(string(rawBytes))
	rawBytes = []byte(stringSansColor)

	printOut(rawBytes, option)
}

func processFileInput(filePath, option string) {
	// Read file input and convert to decolorized raw bytes
	if !utl.FileUsable(filePath) {
		utl.Die("File is unusable\n")
	}

	rawBytes, err := utl.LoadFileText(filePath)
	if err != nil {
		utl.Die("Couln't read file.\n")
	}

	// Remove color codes in file
	stringSansColor := color.ClearCode(string(rawBytes))
	rawBytes = []byte(stringSansColor)

	printOut(rawBytes, option)
}

func printInColor(filePath string) {
	// Print given JSON or YAML file in color
	// JSON must be checked first because it is a subset of the YAML standard
	jsonBytes, err := utl.LoadFileYamlBytes(filePath)
	if err == nil {
		utl.PrintJsonBytesColor(jsonBytes) // Print colorized JSON
	} else {
		// Load as raw YAML byte slice that can include comments
		yamlBytes, err := utl.LoadFileYamlBytes(filePath)
		if err == nil {
			utl.PrintYamlBytesColor(yamlBytes) // Print colorized YAML
		} else {
			utl.Die("File is neither JSON nor YAML\n")
		}
	}
}

func main() {
	var filePath string
	var decolorize bool
	var colorize bool

	args := os.Args[1:] // Get all command-line arguments excluding the program name
	if len(args) > 0 {
		for _, arg := range args {
			switch arg {
			case "-d":
				decolorize = true // Set the decolorize flag
			case "-c":
				colorize = true // Set the colorize flag
			case "-?", "--help", "-h":
				printUsage()
				return
			default:
				// Treat any other argument as a file path
				filePath = arg
			}
		}
	}

	// If a file path was provided, process it
	if filePath != "" {
		if colorize {
			// Print the file in its current format with colorization
			printInColor(filePath)
			return
		} else if decolorize {
			processFileInput(filePath, "decolor_output") // Process with decolorization
			return
		} else {
			processFileInput(filePath, "") // Process normally (convert format)
			return
		}
	} else if hasPipedInput() {
		// If there is piped input, check if we need to decolorize or colorize
		if decolorize {
			processPipedInput("decolor_output") // Process piped input with decolorization
		} else {
			processPipedInput("") // Process piped input normally
		}
	} else {
		printUsage()
	}
}
