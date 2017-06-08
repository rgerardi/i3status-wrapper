# i3status-wrapper

i3status-wrapper is a wrapper for the i3status command written in Go with no external dependencies (std lib only).

i3status is a small program used to generate system information to be displayed by i3bar (and others). It supports several modules like network and disk  information for example, but it does not provide any way to execute custom programs or script if one wants to customize the output.

For simple tasks, a simple bash script (like the example provided in the man page) should be enough. However, if you would like to keep colors and additional formatting, a wrapper that supports i3bar JSON format is required.

This program provides a wrapper for the i3status command with support for i3bar protocol and the ability to run your own scripts and programs, adding their output to the i3status output.

Since it supports the i3bar protocol, it is intended to be used only with i3bar, usually running on [i3 wm] (https://i3wm.org/)

## Features:
* Support for the [i3bar] (https://i3wm.org/docs/i3bar-protocol.html) protocol in JSON
* Support for colors, markup and Unicode characters
* Support for execution of custom commands, adding the output to i3status results
* Support for JSON output of custom commands

## How to install:
```
go get github.com/rgerardi/i3status-wrapper
```

## How to use it:
Simply pipe the output for i3status to i3status-wrapper and execute that instead of i3status, for example in the i3 config file. i3status must be configured to output results in the i3bar JSON format (see example below).

```
i3status | i3status-wrapper
```

i3status-wrapper will run custom commands provided as arguments and add their output before the i3status output in order:

```
i3status | i3status-wrapper custom-script1.sh custom-script2.sh
```

If your command requires arguments, then the command and arguments should be wrapped in double quotes:

```
i3status | i3status-wrapper "custom-script1.sh arg1" custom-script2.sh
```

Finally, i3status must be configured to output status using the i3bar JSON protocol:

```
#### Example i3status config file:

general {
        colors = true
        interval = 5
	output_format = "i3bar"

}
...

``` 
