# utils
A collection of small CLI utilities written in Go.

- [`days`](cmd/days/README.md): A CLI calendar days calculator.
- [`fr`](cmd/fr/README.md): A simple find/replace utility.
- [`tree`](cmd/tree/README.md): A lightweight directory tree printing utility.

## Why?
The Go language and its tool chain are an ideal way to maintain a commonly used set of CLI utilities because they can be quickly compiled and installed whether the OS is Windows, macOS, or Linux. This provides a unified and portable solution to many a scripting needs. With this setup, Go essentially turns into a package manager for these utilities.

## Getting Started
To compile the entire collection, you obviously need to have GoLang installed and properly setup in your system, with `$GOPATH` set up correctly (typically at `$HOME/go`). Also setup `$GOPATH/bin/` in your `$PATH`, since that is where all executable binaries will be placed.

From a `bash` shell do: 

```bash
git clone https://github.com/queone/utils
cd utils
go mod init utils
go mod tidy
./build
```

To build in Windows you have to have a BASH shell such as [GitBASH](https://www.git-scm.com/download/win). To build from a regular Windows Command Prompt, you may have to tweak the `build` script a bit, to have it run the right `go build ...` command. 

