package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar"
	"github.com/jbrukh/goavatar/socket"
	"log"
	"net/http"
)

const (
	DefaultSerialPort = "/dev/tty.AvatarEEG03009-SPPDev"
	DefaultListenPort = 8000
)

var (
	serialPort *string = flag.String("serialPort", DefaultPort, "the serial port for the device")
	mockDevice *bool   = flag.Bool("mockDevice", false, "whether to use the mock device")
	listenPort int     = flag.Bool("listenPort", DefaultListenPort, "the websocket port on which to listen")
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

	log.Printf("starting server on port %d", *listenPort)
	port := fmt.Sprintf(":%d", *listenPort)
	h := socket.Handler(device)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
