//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar/devices"
	. "github.com/jbrukh/goavatar/socket"
	"log"
	"net/http"
)

const (
	DefaultListenPort = 8000
	ControlEndpoint   = "/control"
	DataEndpoint      = "/device"
)

var (
	listenPort *int  = flag.Int("listenPort", DefaultListenPort, "the websocket port on which to listen")
	verbose    *bool = flag.Bool("verbose", false, "whether the socket is verbose (shows outgoing data)")
)

func main() {
	flag.Parse()
	device := ProvideDevice()

	fmt.Printf("Device:   %v\n", device.Name())
	fmt.Printf("Control:  http://localhost:%d%s\n", *listenPort, ControlEndpoint)
	fmt.Printf("Data:     http://localhost:%d%s\n", *listenPort, DataEndpoint)
	port := fmt.Sprintf(":%d", *listenPort)

	http.Handle(ControlEndpoint, ControlHandler(device, *verbose))
	http.Handle(DataEndpoint, DataHandler(device, *verbose))

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
