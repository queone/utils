package main

// The textbelt.com website mentions below is all you really need for the Go calls:
// import (
//   "net/http"
//   "net/url"
// )

// func main() {
//   values := url.Values{
//     "phone": {"5555555555"},
//     "message": {"Hello world"},
//     "key": {"textbelt"},
//   }

//   http.PostForm("https://textbelt.com/text", values)
// }

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/vaughan0/go-ini"
)

const (
	program_name    = "sms"
	program_version = "1.2.0"
)

// Global variables
var (
	cfgfile = "" // func processConfigFile sets it to $HOME/.${ program_name} + "rc"
	svckey  = "textbelt"
	svcurl  = "https://textbelt.com/text"
)

// Print usage information
func printUsage() {
	fmt.Printf("SMS CLI utility %s\n", program_version)
	fmt.Printf("%s <CellPhoneNum> <Message>\n", program_name)
	fmt.Printf("%s -y Create skeleton ~/.%src file\n", program_name, program_name)
	fmt.Println("Visit https://textbelt.com for more info.")
	// n := utl.Whi2(program_name)
	// v := program_version
	// usageHeader := fmt.Sprintf("%s v%s\n"+
	// 	"Memorable password generator - https://github.com/queone/utils/blob/main/cmd/pgen/README.md\n"+
	// 	"%s\n"+
	// 	"  %s [option]\n\n"+
	// 	"%s\n"+
	// 	"                     Without arguments it generates a 3-word memorable password phrase\n"+
	// 	"  NUMBER             Generates a NUMBER-word memorable password phrase\n"+
	// 	"                     For example, if NUMBER is '6' it generates a 6-word phrase\n"+
	// 	"                     Minimum is 1, maximum is 9\n"+
	// 	"  -?, -h, --help     Print this usage page\n",
	// 	n, v, utl.Whi2("Usage"), n, utl.Whi2("Options"))
	// fmt.Print(usageHeader)
	os.Exit(0)
}

// Set up global variables as per values in configuration file
func processConfigFile() {
	// Read config file
	cfgfile = filepath.Join(os.Getenv("HOME"), "."+program_name+"rc")
	if _, err := os.Stat(cfgfile); os.IsNotExist(err) {
		fmt.Printf("Error. Missing '%s' file. Run '%s -y' to create a new one.\n", cfgfile, program_name)
		os.Exit(1)
	}

	f, _ := ini.LoadFile(cfgfile)
	v1, _ := f.Get("global", "svcurl")
	if v1 == "" {
		fmt.Printf("Error. svcurl not defined in '%s' file.\n", cfgfile)
		os.Exit(1)
	}
	svcurl = v1

	v2, _ := f.Get("global", "svckey")
	if v2 == "" {
		fmt.Printf("Error. svckey not defined in '%s' file.\n", cfgfile)
		os.Exit(1)
	}
	svckey = v2
}

// Create a skeleton configuration file with default hard-coded values
func createSkeletonConfigFile() {
	cfgfile := filepath.Join(os.Getenv("HOME"), "."+program_name+"rc")

	// Check if file already exists
	if _, err := os.Stat(cfgfile); err == nil {
		fmt.Printf("There's already a '%s' file.\n", cfgfile)
		return
	} else if !os.IsNotExist(err) {
		// Some unexpected error
		panic(err.Error())
	}

	// Build configuration file content
	content := "# Edit below values accordingly\n"
	content += "[global]\n"
	content += "svcurl = https://textbelt.com/text\n"
	content += "svckey = textbelt\n"

	// Create the file
	f, err := os.Create(cfgfile)
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()

	// Write the contents
	if _, err := f.Write([]byte(content)); err != nil {
		panic(err.Error())
	}
}

func main() {

	tel, msg := "", ""
	argCount := len(os.Args[1:])

	if argCount < 1 || argCount > 2 {
		printUsage()
	}
	if os.Args[1] == "-y" {
		createSkeletonConfigFile()
	}

	processConfigFile()

	tel = os.Args[1]
	msg = os.Args[2]

	values := url.Values{
		"key":     {svckey},
		"phone":   {tel},
		"message": {msg},
	}
	fmt.Printf("%s  %s  %s\n", svckey, tel, msg)
	resp, err := http.PostForm(svcurl, values)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		fmt.Printf("Error. HTTP error code = %s\n", resp.Status)
		os.Exit(1)
	}
	os.Exit(0)
}

// func main() {
// 	args := len(os.Args[1:])
// 	numWords := 3 // default

// 	if args == 1 {
// 		arg1 := os.Args[1]
// 		switch arg1 {
// 		case "-?", "-h", "--help":
// 			printUsage()
// 		default:
// 			n, err := strconv.Atoi(arg1)
// 			if err != nil || n < 1 || n > 9 {
// 				fmt.Println("NUMBER must be 1 thru 9.")
// 				os.Exit(1)
// 			}
// 			numWords = n
// 		}
// 	}

// 	// 1. Original diceware password
// 	dicewareWords := GenerateDiceware(numWords)
// 	fmt.Println(strings.Join(dicewareWords, delimiter))

// 	// 2. Strong memorable password
// 	fmt.Println(GenerateStrongMemorable(dicewareWords))

// 	// 3. Random alphanumeric password 16 chars
// 	fmt.Println(GenerateRandomAlphaNumeric(16))
// }
