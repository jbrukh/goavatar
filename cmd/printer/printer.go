//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	. "github.com/jbrukh/goavatar/device"
	. "github.com/jbrukh/goavatar/drivers"
	. "github.com/jbrukh/goavatar/formats"
	"log"
)

const (
	DefaultPort      = "/dev/tty.AvatarEEG03009-SPPDev"
	DefaultMaxFrames = 10000
	WindowMultiple   = 10
)

var (
	maxFrames *int = flag.Int("maxFrames", DefaultMaxFrames, "maximum frames to read before turning off")
	rec       *int = flag.Int("rec", 0, "frames to record")
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

	if *rec > 0 {
		log.Printf("going to record...")
		r := NewDeviceRecorder(device, NewOBFRecorder(device.Repo()))
		r.SetMax(*rec)
		if err = r.RecordAsync(map[string]string{"subdir": "printer"}); err != nil {
			log.Printf("Error: %v", err)
			return
		}
		if id, err := r.Wait(); err != nil {
			log.Printf("could not stop")
		} else {
			log.Printf("Recorded result to: %s", id)
		}

	}

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
		samples := b.Samples()
		for s := 0; s < samples; s++ {
			v, t := b.Sample(s)
			fmt.Printf("%v, %v\n", t, v)
		}
	}
}
