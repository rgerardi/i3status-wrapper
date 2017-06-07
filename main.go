package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// i3bar struct represents a block in the i3bar protocol
// http://i3wm.org/docs/i3bar-protocol.html
type i3bar struct {
	Name                string `json:"name,omitempty"`
	Instance            string `json:"instance,omitempty"`
	Markup              string `json:"markup,omitempty"`
	FullText            string `json:"full_text,omitempty"`
	Color               string `json:"color,omitempty"`
	ShortText           string `json:"short_text,omitempty"`
	Background          string `json:"background,omitempty"`
	Border              string `json:"border,omitempty"`
	MinWidth            int    `json:"min_width,omitempty"`
	Align               string `json:"align,omitempty"`
	Urgent              bool   `json:"urgent,omitempty"`
	Separator           bool   `json:"separator,omitempty"`
	SeparatorBlockWidth int    `json:"separator_block_width,omitempty"`
}

func main() {

	stdInDec := json.NewDecoder(os.Stdin)
	stdOutEnc := json.NewEncoder(os.Stdout)

	// The first line is a header indicating to i3bar that JSON will be used
	i3barHeader := struct {
		Version int `json:"version"`
	}{}

	err := stdInDec.Decode(&i3barHeader)
	if err != nil {
		fmt.Println("Cannot read input:", err.Error())
		os.Exit(1)
	}

	err = stdOutEnc.Encode(i3barHeader)
	if err != nil {
		fmt.Println("Cannnot encode output json:", err.Error())
		os.Exit(1)
	}

	// The second line is just the start of the endless array '['
	t, err := stdInDec.Token()
	if err != nil {
		fmt.Println("Cannot read input:", err.Error())
		os.Exit(1)
	}

	fmt.Println(t)

	// Start of the main i3status loop (endless array)
	for stdInDec.More() {

		// For every iteration of the loop we capture the blocks provided by i3status
		// and append custom blocks to it before sending it to i3bar
		var blocks []i3bar

		err := stdInDec.Decode(&blocks)

		if err != nil {
			fmt.Println("Cannnot decode input json:", err.Error())
			os.Exit(1)
		}

		// Creating an empty array of i3bar blocks to be filled with custom blocks
		customBlocks := []i3bar{}

		// Custom commands to be included in the output should be provided
		// as arguments to i3status-wrapper
		customCommands := os.Args[1:]

		for _, cmd := range customCommands {

			// Commands are split by a blank space to separate arguments if any
			cmdSplit := strings.Split(cmd, " ")

			customCmd := exec.Command(cmdSplit[0], cmdSplit[1:]...)
			cmdStatusOutput, err := customCmd.Output()

			if err != nil {
				fmt.Println("Cannnot run command:", cmd, ":", err.Error())
				os.Exit(1)
			}

			cmdStatusOutput = bytes.TrimSpace(cmdStatusOutput)

			// Here we try to parse the output as JSON with the i3bar format
			// If it fails the output will be processed as a regular string
			var customBlock i3bar

			err = json.Unmarshal(cmdStatusOutput, &customBlock)

			if err != nil {
				// Not JSON, using custom fields and string output as FullText
				customBlock.Name = "customCmd"
				customBlock.Instance = cmdSplit[0]
				customBlock.FullText = string(cmdStatusOutput)
			}

			customBlocks = append(customBlocks, customBlock)
		}

		customBlocks = append(customBlocks, blocks...)

		err = stdOutEnc.Encode(customBlocks)
		if err != nil {
			fmt.Println("Cannnot encode output json:", err.Error())
			os.Exit(1)
		}

		// A comma is required to signal another entry in the array to i3bar
		fmt.Print(",")
	}
}
