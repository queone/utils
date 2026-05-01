package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Prints JSON object, flushing the output buffer
func printJson(jsonObject any) {
	pretty, err := prettify(jsonObject)
	if err != nil {
		fmt.Printf("Prettify() error\n")
	} else {
		fmt.Println(pretty)
	}
	os.Stdout.Sync()
}

// Convert JSON interface object to byte slice, with option to indent spacing
func jsonBytesReindent(jsonBytes []byte, indent int) (jsonBytes2 []byte, err error) {
	var prettyJson bytes.Buffer
	indentStr := strings.Repeat(" ", indent)
	err = json.Indent(&prettyJson, jsonBytes, "", indentStr)
	if err != nil {
		return nil, err
	}
	jsonBytes2 = prettyJson.Bytes()
	return jsonBytes2, nil
}

// Convert JSON byte slice to JSON interface object, with default 2-space indentation
func jsonBytesToJsonObj(jsonBytes []byte) (jsonObject any, err error) {
	err = json.Unmarshal(jsonBytes, &jsonObject)
	return jsonObject, err
}

// NOTE: To be replaced by jsonToBytes()
func prettify(jsonObject any) (pretty string, err error) {
	j, err := json.MarshalIndent(jsonObject, "", "  ")
	return string(j), err
}

// Prints JSON byte slice in color. Just an alias of yaml.go:printYamlBytesColor().
func printJsonBytesColor(jsonBytes []byte) {
	printYamlBytesColor(jsonBytes)
}
