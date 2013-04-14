package main

import (
	"flag"
	"github.com/jbrukh/goavatar"
	"log"
)

const DEFAULT_PORT = "/dev/tty.AvatarEEG03009-SPPDev"

var serialPort *string = flag.String("port", DEFAULT_PORT, "the serial port for the device")

func init() {
	flag.Parse()
}

func main() {
	device := goavatar.NewDevice(DEFAULT_PORT)
	out, err := device.Connect()
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	for i := 0; i < 1000; i++ {
		df, ok := <-out
		if !ok {
			log.Printf("The data channel got closed (exiting)")
			return
		}
		log.Printf("Got df: %v", df.String())
	}
	log.Printf("Finished... closing.")
	device.Disconnect()
}
