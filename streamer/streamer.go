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
	err, output := device.Connect()
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	for i := 0; i < 100; i++ {
		df, ok := <-output
		if !ok {
			log.Printf("The data channel got closed (exiting)")
			return
		}
		log.Printf("Got df: %v\n", df.String())
	}
	log.Printf("Finished... closing.")
	device.Disconnect()
}
