// main.go

// TODO: Check back on 03:14:08 UTC on 19 January 2038, to confirm we're good ;-)
//       https://en.wikipedia.org/wiki/Year_2038_problem

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/queone/governa-color"
)

const (
	// Global constants
	programName    = "days"
	programVersion = "1.0.8"
)

func printUsage() {
	n := color.Whi2(programName)
	v := programVersion
	usage := fmt.Sprintf("%s v%s\n"+
		"Calendar days calculator — https://github.com/queone/utils/blob/main/cmd/days/README.md\n"+
		"%s\n"+
		"  This utility works with calendar dates expressed as YYYY-MM-DD (or the equivalent\n"+
		"  YYYY-MMM-DD format), and reports the relationship between today's date and the supplied\n"+
		"  argument(s). Supported invocations are:\n"+
		"\n"+
		"    days -v, --version            Prints this information screen.\n"+
		"    days -N                       Prints the calendar date N days ago (e.g. -11).\n"+
		"    days +N                       Prints the calendar date N days in the future (e.g. +6 or just 6).\n"+
		"    days YYYY-MM-DD               Prints the number of days between today and the given date (positive\n"+
		"                                  if the date is in the future, negative if it is in the past).\n"+
		"    days YYYY-MM-DD YYYY-MM-DD    Prints the number of days between the two supplied dates.\n",
		n, v, color.Whi2("Overview"))
	fmt.Print(usage)
	os.Exit(0)
}

func main() {
	numberOfArguments := len(os.Args[1:]) // Not including the program itself
	if numberOfArguments < 1 || numberOfArguments > 2 {
		// Don't accept less than 1 or more than 2 arguments
		printUsage()
	}

	// Process given arguments
	switch numberOfArguments {
	case 1:
		arg1 := os.Args[1]
		if arg1 == "-v" || arg1 == "--version" {
			printUsage()
		} else if validDate(arg1, "2006-01-02") {
			days := getDaysSinceOrTo(arg1)
			printDays(days)
		} else if arg1[0:1] == "+" || arg1[0:1] == "-" {
			dateStr := getDateInDays(arg1)
			fmt.Println(dateStr.Format("2006-01-02"))
		} else if _, err := strconv.Atoi(arg1); err == nil { // Check if arg1 is a valid number
			arg1 = "+" + arg1
			dateStr := getDateInDays(arg1)
			fmt.Println(dateStr.Format("2006-01-02"))
		}
	case 2:
		arg1 := os.Args[1]
		arg2 := os.Args[2]
		if validDate(arg1, "2006-01-02") && validDate(arg2, "2006-01-02") {
			days := getDaysBetween(arg1, arg2)
			printDays(days)
		}
	default:
		printUsage()
	}
}
