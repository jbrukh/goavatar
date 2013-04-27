package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar"
	"log"
	"net/http"
)

const (
	DefaultSerialPort = "/dev/tty.AvatarEEG03009-SPPDev"
	DefaultListenPort = 8000
	ControlEndpoint   = "/control"
	DataEndpoint      = "/device"
)

var (
	serialPort *string = flag.String("serialPort", DefaultSerialPort, "the serial port for the device")
	mockDevice *bool   = flag.Bool("mockDevice", false, "whether to use the mock device")
	listenPort *int    = flag.Int("listenPort", DefaultListenPort, "the websocket port on which to listen")
	verbose    *bool   = flag.Bool("verbose", false, "whether the socket is verbose (shows outgoing data)")
)

func main() {
	flag.Parse()
	var device Device

	// get the device
	if *mockDevice {
		log.Printf("using the MockDevice")
		device = NewMockDevice()
	} else {
		log.Printf("using the AvatarEEG on port %s", *serialPort)
		device = NewAvatarDevice(*serialPort)
	}

	log.Printf("starting server, ControlEndpoint: http://localhost:%d%s; DataEndpoint: http://localhost:%d%s", *listenPort,
		ControlEndpoint, *listenPort, DataEndpoint)
	port := fmt.Sprintf(":%d", *listenPort)

	http.Handle(ControlEndpoint, ControlHandler(device, *verbose))
	http.Handle(DataEndpoint, DataHandler(device, *verbose))

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
