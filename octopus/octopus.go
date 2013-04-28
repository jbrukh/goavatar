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
	integers   *bool   = flag.Bool("integers", false, "whether to return integral data")
)

func main() {
	flag.Parse()
	var device Device

	// get the device
	if *mockDevice {
		device = NewMockDevice()
	} else {
		device = NewAvatarDevice(*serialPort)
	}
	log.Printf("Device:\t%v", device.Name())

	log.Printf("Control:\thttp://localhost:%d%s", *listenPort, ControlEndpoint)
	log.Printf("Data:\thttp://localhost:%d%s", *listenPort, DataEndpoint)
	port := fmt.Sprintf(":%d", *listenPort)

	http.Handle(ControlEndpoint, ControlHandler(device, *verbose))
	http.Handle(DataEndpoint, DataHandler(device, *verbose, *integers))

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
