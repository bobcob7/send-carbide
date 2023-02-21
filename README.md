# Send Carbide

Send a GCode file to [Carbide Motion](https://carbide3d.com/carbidemotion) over a network.

I got tired to shuffling files between my design machine and my CNC PC and I didn't want to setup a networked file syncing system.
I noticed that the remote access feature was recently released and wasn't very useful unless you used Carbide Create.

I personally use Fusion360, this is a first step towards making a add-in to integrate this further.

This program requires that know how to use a CLI on your OS.
Tested on a MacOS 13.1 and Carbide Motion Build 578

## Installation

### From Release

1. Download the binary for your operating system from the [release page](https://github.com/bobcob7/send-carbide/releases).
2. If you're using linux or MacOS, be sure to make the binary executable using `chmod +x send-carbide`
3. Move binary to a location in you path. Ex. `cp send-carbide /usr/local/bin/`

### From Source

1. Install [Go development environment](https://go.dev/doc/install). Remember to add `$GOPATH/bin` to your path.
2. Run `go install github.com/bobcob7/send-carbide`

## Usage

Run the program while specifying your GCode file and the address of your CNC PC.

```bash
send-carbide -address 127.0.0.1 -file test-file.gcode
```

You should immediately see output and a progress bar should begin in Carbide Motion.
