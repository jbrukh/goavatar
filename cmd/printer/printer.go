//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar/device"
	. "github.com/jbrukh/goavatar/drivers"
	"log"
)

const (
	DefaultPort      = "/dev/tty.AvatarEEG03009-SPPDev"
	DefaultMaxFrames = 10000
	WindowMultiple   = 10
)

var (
	maxFrames *int = flag.Int("maxFrames", DefaultMaxFrames, "maximum frames to read before turning off")
)

func init() {
	flag.Parse()
}

func main() {
	// set up the device
	device, err := ProvideDevice()
	if err != nil {
		log.Fatalf("could not get device: %v", err)
	}

	// connect to it
	if err := device.Engage(); err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	defer device.Disengage()

	out, err := device.Subscribe("printer")
	if err != nil {
		log.Printf("could not subscribe to device: %s", err)
		return
	}
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
