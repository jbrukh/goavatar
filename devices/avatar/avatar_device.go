//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package avatar

import (
	"errors"
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/formats"
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

// AvatarEEG; implements DeviceImpl
type AvatarDevice struct {
	serialPort string // serial port like /dev/tty.AvatarEEG03009-SPPDev
	reader     io.ReadCloser
	repo       string
	name       string
}

// NewAvatarDevice creates a new AvatarEEG connection. The user
// can then start streaming data by calling Connect() and reading the
// output channel.
func NewAvatarDevice(serialPort, repo string) Device {
	return NewDevice(&AvatarDevice{
		serialPort: serialPort,
		name:       "AvatarEEG",
		repo:       repo,
	})

}

// Engaging the AvatarEEG means opening the serial
// port to the device, at which point it immediately
// begins streaming.
func (ad *AvatarDevice) Engage() (err error) {
	ad.reader, err = os.Open(ad.serialPort)
	return
}

// Disengage by closing the serial port.
func (ad *AvatarDevice) Disengage() (err error) {
	return ad.reader.Close()
}

// Process the stream.
func (ad *AvatarDevice) Stream(c *Control) (err error) {
	return parseByteStream(ad.reader, c)
}

// Provide a recorder.
func (ad *AvatarDevice) ProvideRecorder() Recorder {
	return NewOBFRecorder(ad.repo)
}

// The name of the device: AvatarEEG.
func (ad *AvatarDevice) Name() string {
	return ad.name
}

// The repo to which recordings are written.
func (ad *AvatarDevice) Repo() string {
	return ad.repo
}

// parseByteStream parses the byte stream coming out of the device and writes the output
// to the output channel parameter. It also listens on the Control in order to
// know when to terminate. Note that this function must strictly obey ShouldTerminate()
// and call Close() upon exiting.
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
}
