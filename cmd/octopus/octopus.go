//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	"flag"
	. "github.com/jbrukh/goavatar/drivers"
	. "github.com/jbrukh/goavatar/socket"
	"log"
)

func init() {
	flag.Parse()
}

func main() {
	device, err := ProvideDevice()
	if err != nil {
		log.Fatalf("could not get device: %v", err)
	}

	NewOctopusSocket(device).ListenAndServe()
}
