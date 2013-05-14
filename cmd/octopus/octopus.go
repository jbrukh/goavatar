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
	. "github.com/jbrukh/goavatar/socket"
	"log"
	"net/http"
)

const (
	DefaultSerialPort = "/dev/tty.AvatarEEG03009-SPPDev"
	DefaultRepo       = "var"
	DefaultListenPort = 8000
	ControlEndpoint   = "/control"
	DataEndpoint      = "/device"
	DefaultMockFile   = "etc/ee6d09f8-1df6-5bac-deee-c18a28407190"
)

var (
	serialPort *string = flag.String("serialPort", DefaultSerialPort, "the serial port for the device")
	mockDevice *bool   = flag.Bool("mockDevice", false, "whether to use the mock device")
	listenPort *int    = flag.Int("listenPort", DefaultListenPort, "the websocket port on which to listen")
	verbose    *bool   = flag.Bool("verbose", false, "whether the socket is verbose (shows outgoing data)")
	repo       *string = flag.String("repo", DefaultRepo, "directory where recordings are stored")
	mockFile   *string = flag.String("mockFile", DefaultMockFile, "OBF file to play back in the mock device")
)

func main() {
	flag.Parse()
	var device Device

	// get the device
	if *mockDevice {
		device = NewMockDevice(*repo, *mockFile)
	} else {
		device = NewAvatarDevice(*serialPort, *repo)
	}
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
