//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	. "github.com/jbrukh/goavatar/devices"
	. "github.com/jbrukh/goavatar/socket"
	"log"
)

func main() {
	device, err := ProvideDevice()
	if err != nil {
		log.Fatalf("could not get device: %v", err)
	}

	NewOctopusSocket(device).ListenAndServe()
}
