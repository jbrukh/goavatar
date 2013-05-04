//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package avatar

import (
	"errors"
	. "github.com/jbrukh/goavatar"
	"io"
	"log"
	"os"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

const (
	DataBufferSize   = 1024
	DiagnosticFrames = 10
)

var BadCrcErr = errors.New("frame had bad crc")

// ----------------------------------------------------------------- //
// AvatarEEG Device
// ----------------------------------------------------------------- //

type AvatarDevice struct {
	BaseDevice
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
	streamFunc := func(c *Control) error {
		return parseByteStream(reader, c)
	}

	recorderProvider := func(token string) Recorder {
		return NewFileRecorder(token)
	}

	return &AvatarDevice{
		BaseDevice: *NewBaseDevice("AvatarEEG", connFunc, disconnFunc, streamFunc, recorderProvider),
		serialPort: serialPort,
	}
}

// parseByteStream parses the byte stream coming out of the device and writes the output
// to the output channel parameter. It also listens on the offSignal channel for any
// data, in which case it will stop listening the device and return.
func parseByteStream(r io.ReadCloser, c *Control) (err error) {
	parser := NewAvatarParser(r)
	defer c.Close()

	for {
		if c.ShouldTerminate() {
			return nil
		}

		frame, err := parser.ParseFrame()
		if err != nil {
			log.Printf("error parsing frame: %v", err)
			return err
		}

		c.Send(frame)
	}
	return nil
}
