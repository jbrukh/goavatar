//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/devices/avatar"
	. "github.com/jbrukh/goavatar/devices/mock_avatar"
	"log"
)

const (
	DefaultPort        = "/dev/tty.AvatarEEG03009-SPPDev"
	DefaultMaxFrames   = 10000
	WindowMultiple     = 10
	DefaultMockFile    = "etc/1fabece1-7a57-96ab-3de9-71da8446c52c"
)

var (
	serialPort  *string = flag.String("port", DefaultPort, "the serial port for the device")
	maxFrames   *int    = flag.Int("maxFrames", DefaultMaxFrames, "maximum frames to read before turning off")
	mockDevice  *bool   = flag.Bool("mockDevice", false, "whether to use the mock device")
	mockFile    *string = flag.String("mockFile", DefaultMockFile, "OBF file to play back in the mock device")
	mockChannels *int   = flag.Int("mockChannels", 2, "the number of channels to mock")
)

func init() {
	flag.Parse()
}

func main() {
	// set up the device
	var device Device
	if *mockDevice {
		device = NewMockDevice("", *mockFile, *mockChannels)
	} else {
		device = NewAvatarDevice(*serialPort, "")
	}

	// connect to it
	if err := device.Connect(); err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	defer device.Disconnect()

	out := device.Out()
	printFrame(out)
	log.Printf("Thank you!")
}

func printFrame(out <-chan DataFrame) {
	for i := 0; i < *maxFrames; i++ {
		df, ok := <-out
		if !ok {
			log.Printf("The data channel got closed (exiting)")
			return
		}

		b := df.Buffer()
		for b.Samples() > 0 {
			v, t := b.PopSample()
			fmt.Printf("%v, %v\n", t, v)
		}
	}
}
