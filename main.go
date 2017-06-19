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

// customCommand represents a custom command to be executed
// contains details about the command to be executed, its arguments (if any),
// timeout, the result block and the order in which the result should be displayed
type customCommand struct {
	command string
	args    []string
	timeout time.Duration
	result  *i3bar
	order   int
}

func (c *customCommand) execute() ([]byte, error) {

	// Adding a context with timeout to handle cases of long running commands
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	customCmd := exec.CommandContext(ctx, c.command, c.args...)
	cmdStatusOutput, err := customCmd.Output()

	// If the deadline was exceeded, just output that to the status instead of failing
	if ctx.Err() == context.DeadlineExceeded {
		return []byte("Timed out"), nil
	}

	if err != nil {
		return nil, err
	}

	cmdStatusOutput = bytes.TrimSpace(cmdStatusOutput)
	return cmdStatusOutput, nil
}

func (c *customCommand) runJob(done chan int) {

	cmdStatusOutput, err := c.execute()
	if err != nil {
		fmt.Println("Cannot run command:", c.command, ":", err.Error())
		os.Exit(1)
	}

	// Here we try to parse the output as JSON with the i3bar format
	// If it fails the output will be processed as a regular string
	err = json.Unmarshal(cmdStatusOutput, c.result)

	if err != nil {
		// Not JSON, using custom fields and string output as FullText
		c.result.Name = "customCmd"
		c.result.Instance = c.command
		c.result.FullText = string(cmdStatusOutput)
	}

	// Send status out to channel, indicates both completion and order
	done <- c.order

}

func main() {

	// Defining a timeout flag to give control to the user on when to timeout long running commands
	// It will be used by the context creation command in the loop below
	timeout := flag.Duration("timeout", 5*time.Second, "timeout for custom command execution")
	flag.Parse()

	// Defining an slice to hold the list of custom commands to be executed every iteration
	cmdList := make([]*customCommand, len(flag.Args()))

	// Custom commands to be included in the output should be provided
	// as arguments to i3status-wrapper. They will be parsed by flag.Args
	for k, cmd := range flag.Args() {

		// Commands are split by a blank space to separate arguments if any
		cmdSplit := strings.Split(cmd, " ")

		cmdList[k] = &customCommand{
			command: cmdSplit[0],
			args:    cmdSplit[1:],
			timeout: *timeout,
			result:  &i3bar{},
			order:   k,
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
		fmt.Println("Cannot encode output json:", err.Error())
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
		var blocks []*i3bar

		err := stdInDec.Decode(&blocks)

		if err != nil {
			fmt.Println("Cannot decode input json:", err.Error())
			os.Exit(1)
		}

		// Creating an empty slice of i3bar blocks to be filled with custom blocks
		customBlocks := make([]*i3bar, len(cmdList), len(blocks)+len(cmdList))

		// Creating a channel to receive the status for the goroutine that runs
		// the custom commands asynchronously (contains the order of the finished command)
		done := make(chan int)

		// Execute every custom command provided as an argument to i3status-wrapper
		for _, cmd := range cmdList {
			go cmd.runJob(done)
		}

		// Process custom commands results as they become available
		for i := 0; i < len(cmdList); i++ {
			d := <-done
			// d contains the order provided during creation. It will be  used to
			// position commands correctly regardless of results availability order
			customBlocks[d] = cmdList[d].result
		}
		close(done)

		// Appending blocks from i3status to the custom blocks
		customBlocks = append(customBlocks, blocks...)

		// Enconding & Sending output back to stdout to be processed by i3bar
		err = stdOutEnc.Encode(customBlocks)
		if err != nil {
			fmt.Println("Cannot encode output json:", err.Error())
			os.Exit(1)
		}

		// A comma is required to signal another entry in the array to i3bar
		fmt.Print(",")
	}
}
