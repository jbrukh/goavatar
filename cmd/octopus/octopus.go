//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	"flag"
	. "github.com/jbrukh/goavatar/drivers"
	. "github.com/jbrukh/goavatar/socket"
	"log"
	"os"
	"os/signal"
)

var c = make(chan os.Signal, 1)

func init() {
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			log.Printf("Terminating the Octopus Server (SIGINT)...\n")
			os.Exit(0)
		}
	}()
}

func main() {
	flag.Parse()
	device, err := ProvideDevice()
	if err != nil {
		log.Fatalf("could not get device: %v", err)
	}

	NewOctopusSocket(device).ListenAndServe()
}
