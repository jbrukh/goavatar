package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

const DEFAULT_PORT = "/dev/tty.AvatarEEG03009-SPPDev"

var port *string = flag.String("port", DEFAULT_PORT, "the serial port for the device")

func init() {
	flag.Parse()
}

func main() {
	const ITER = 100
	// connect to the port
	device, err := os.Open(*port)
	if err != nil {
		log.Fatalf("Could not connect to the port: %v", err)
	}
	defer device.Close()
	log.Printf("Connected to the device on port: %s", *port)

	// try to read chunks of bytes
	readSomeBytes(device, 1024)
	readSomeBytes(device, 2048)
	readSomeBytes(device, 4096)
	readSomeBytes(device, 8192)
	readSomeBytes(device, 16384)
	readSomeBytes(device, 32768)
	readSomeBytes(device, 65536)
	readSomeBytes(device, 131072)
	readSomeBytes(device, 262144)
	readSomeBytes(device, 524288)
	readSomeBytes(device, 1048576)

	// try to read 128 a bunch of times
	for i := 0; i < ITER; i++ {
		readSomeBytes(device, 64)
	}

}

func readSomeBytes(device *os.File, n int) {
	bytes := make([]byte, n)
	t0 := time.Now()

	nRead, err := device.Read(bytes)

	dt := float64(time.Since(t0).Nanoseconds()) / 1000000
	fmt.Printf("%d bytes: %.3f ms | ", n, dt)
	if nRead != n {
		fmt.Printf("Only read %d bytes", nRead)
	}
	if err != nil {
		log.Printf("Error occurred: %v", err)
	}
	fmt.Println("")

}

func readSomeBytesWithBuffer(device *os.File, n int, bufSize int) {

}
