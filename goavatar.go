package goavatar

import (
	"fmt"
	"io"
	"log"
	"os"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

const (
	DataBufferSize = 1024
)

type AvatarChannel int

// AvatarEEG channels
const (
	AvatarChannelTrigger AvatarChannel = iota
	AvatarChannel1
	AvatarChannel2
	AvatarChannel3
	AvatarChannel4
	AvatarChannel5
	AvatarChannel6
	AvatarChannel9
	AvatarChannel8
)

// ----------------------------------------------------------------- //
// AvatarEEG Device
// ----------------------------------------------------------------- //

// Device represents an AvatarEEG device on a particular port.
type Device struct {
	serialPort string          // serial port like /dev/tty.AvatarEEG03009-SPPDev
	offSignal  chan bool       // send a value to disconnect the device
	reader     io.ReadCloser   // the reader of the serial port
	output     chan *DataFrame // channel that delivers raw Avatar output
}

// NewDevice creates a new Device. The user can then start
// streaming data by calling Connect() and reading the 
// output channel.
func NewDevice(serialPort string) *Device {
	return &Device{
		serialPort: serialPort,
		offSignal:  make(chan bool, 1),
		output:     make(chan *DataFrame, DataBufferSize),
	}
}

// Connect to the device.
func (d *Device) Connect() (output <-chan *DataFrame, err error) {
	// connect to the reader for the port; this will
	// fail if we are already reading from this port
	reader, err := os.Open(d.serialPort)
	if err != nil {
		return nil, fmt.Errorf("Cannot connect: %v", err)
	}

	// remember the reader and begin streaming data
	// on a separate thread
	d.reader = reader
	go func() {
		parseByteStream(d.reader, d.offSignal, d.output)
	}()
	return d.output, nil
}

// Disconnect from the device.
func (d *Device) Disconnect() {
	// send the off signal; will block until the
	// offSignal is processed on the output thread
	d.offSignal <- true

	// close the reader
	if err := d.reader.Close(); err != nil {
		log.Printf("Error closing the reader: %v", err)
	}

	// close the output channel
	close(d.output)
}

// parseByteStream parses the byte stream coming out of the device and writes the output
// to the output channel parameter. It also listens on the offSignal channel for any
// data, in which case it will stop listening the device and return.
func parseByteStream(r io.ReadCloser, offSignal <-chan bool, output chan<- *DataFrame) {
	defer r.Close()
	reader := newAvatarParser(r)

	for {
		log.Printf("new loop")
		// break the loop if 
		// there is an off signal
		if shouldBreak(offSignal) {
			break
		}

		// read the frame
		err := reader.ConsumeSync()
		if err != nil {
			log.Printf("Error: %v", err)
			break // since the underlying reader must be hosed
		}

		header, err := reader.ConsumeHeader()
		if err != nil {
			log.Printf("Error: %v", err)
			continue // will break on next loop if reader hosed
		}

		err = reader.ConsumePayload(header)
		if err != nil {
			log.Printf("Error: %v", err)
			continue // will break on next loop if reader hosed
		}

		crc, err := reader.ConsumeCrc()
		if err != nil {
			log.Printf("Error: %v", err)
			continue // will break on next loop if reader hosed
		}

		// collect the frame
		frame := &DataFrame{
			DataFrameHeader: *header,
			crc:             crc,
		}
		ourCrc := reader.Crc()
		//log.Printf("Frame: %+v, Crc: %v", *frame, ourCrc)
		if ourCrc != crc {
			log.Printf("Bad CRC: %+v (expected: %d)", *frame, ourCrc)
			continue
		}
		output <- frame
	}
}

func shouldBreak(offSignal <-chan bool) bool {
	select {
	case <-offSignal:
		log.Printf("Received off signal...")
		return true
	default:
	}
	return false
}
