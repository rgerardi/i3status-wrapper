package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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

type customCommand struct {
	command string
	args    []string
	timeout time.Duration
}

func (c *customCommand) execute() ([]byte, error) {

	// Adding a context with timeout to handle cases of long running commands
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	customCmd := exec.CommandContext(ctx, c.command, c.args...)
	cmdStatusOutput, err := customCmd.Output()

	// If the deadline was exceeded, just ouput that to the status instead of failing
	if ctx.Err() == context.DeadlineExceeded {
		return []byte("Timed out"), nil
	}

	if err != nil {
		return nil, err
	}

	cmdStatusOutput = bytes.TrimSpace(cmdStatusOutput)
	return cmdStatusOutput, nil
}

func main() {

	// Defining a timeout flag to give control to the user on when to timeout long running commands
	// It will be used by the context creation command in the loop below
	timeout := flag.Duration("timeout", 5*time.Second, "timeout for custom command execution")
	flag.Parse()

	// Defining an slice to hold the list of custom commands to be executed every iteration
	cmdList := make([]customCommand, len(flag.Args()))

	// Custom commands to be included in the output should be provided
	// as arguments to i3status-wrapper. They will be parsed by flag.Args
	for k, cmd := range flag.Args() {

		// Commands are split by a blank space to separate arguments if any
		cmdSplit := strings.Split(cmd, " ")

		cmdList[k] = customCommand{
			command: cmdSplit[0],
			args:    cmdSplit[1:],
			timeout: *timeout,
		}
	}

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

		// Creating an empty slice of i3bar blocks to be filled with custom blocks
		customBlocks := make([]i3bar, len(cmdList), len(blocks)+len(cmdList))

		// Execute every custom command provided as an argument ot i3status-wrapper
		for k, cmd := range cmdList {

			cmdStatusOutput, err := cmd.execute()
			if err != nil {
				fmt.Println("Cannnot run command:", cmd.command, ":", err.Error())
				os.Exit(1)
			}

			// Here we try to parse the output as JSON with the i3bar format
			// If it fails the output will be processed as a regular string
			var customBlock i3bar

			err = json.Unmarshal(cmdStatusOutput, &customBlock)

			if err != nil {
				// Not JSON, using custom fields and string output as FullText
				customBlock.Name = "customCmd"
				customBlock.Instance = cmd.command
				customBlock.FullText = string(cmdStatusOutput)
			}

			customBlocks[k] = customBlock
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
