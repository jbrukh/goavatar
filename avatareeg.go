package goavatar

import (
	"io"
	"log"
	"os"
)

// ----------------------------------------------------------------- //
// AvatarEEG Device
// ----------------------------------------------------------------- //

type AvatarDevice struct {
	baseDevice
	serialPort string // serial port like /dev/tty.AvatarEEG03009-SPPDev
}

// NewAvatarDevice creates a new AvatarEEG connection. The user 
// can then start streaming data by calling Connect() and reading the 
// output channel.
func NewAvatarDevice(serialPort string) *AvatarDevice {
	var (
		reader io.ReadCloser
	)

	// connect to the avatar by connecting to the
	// specified serial port
	connFunc := func() (err error) {
		reader, err = os.Open(serialPort)
		return
	}

	// disconnect from the device
	disconnFunc := func() error {
		return reader.Close()
	}

	// the streaming function
	streamFunc := func(offSignal <-chan bool, out chan<- *DataFrame) {
		parseByteStream(reader, offSignal, out)
	}

	return &AvatarDevice{
		baseDevice: *newBaseDevice(connFunc, disconnFunc, streamFunc),
		serialPort: serialPort,
	}
}

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

const (
	DataBufferSize = 1024
)

// parseByteStream parses the byte stream coming out of the device and writes the output
// to the output channel parameter. It also listens on the offSignal channel for any
// data, in which case it will stop listening the device and return.
func parseByteStream(r io.ReadCloser, offSignal <-chan bool, output chan<- *DataFrame) {
	reader := newAvatarParser(r)

	for {
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

		data, err := reader.ConsumePayload(header)
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
			data:            data,
			crc:             crc,
		}
		ourCrc := reader.Crc()
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
		return true
	default:
	}
	return false
}
