//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package avatar

import (
	"errors"
	. "github.com/jbrukh/goavatar/device"
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
	repo       *Repository
	name       string
}

// NewAvatarDevice creates a new AvatarEEG connection. The user
// can then start streaming data by calling Connect() and reading the
// output channel.
func NewAvatarDevice(basedir, serialPort string) Device {
	return NewDevice(&AvatarDevice{
		serialPort: serialPort,
		name:       "AvatarEEG",
		repo:       NewRepositoryOrPanic(basedir),
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
func (ad *AvatarDevice) Repo() *Repository {
	return ad.repo
}

// parseByteStream parses the byte stream coming out of the device and writes the output
// to the output channel parameter. It also listens on the Control in order to
// know when to terminate. Note that this function must strictly obey ShouldTerminate()
// and call Close() upon exiting.
func parseByteStream(r io.ReadCloser, c *Control) (err error) {
	parser := NewAvatarParser(r)

	// first send the device info; the Avatar keeps its
	// info on its frames, so we will parse the first frame
	frame, err := parser.ParseFrame()
	if err != nil {
		if IsCrcErr(err) || IsSizeErr(err) {
			log.Printf("skippable error: %v", err)
		} else {
			log.Printf("error parsing frame: %v", err)
			return err
		}
	}
	info := &DeviceInfo{
		Channels:   frame.Channels(),
		SampleRate: frame.SampleRate(),
	}
	c.SendInfo(info)

	for {
		if c.ShouldTerminate() {
			return nil
		}

		frame, err = parser.ParseFrame()
		if err != nil {
			if IsCrcErr(err) || IsSizeErr(err) {
				log.Printf("skippable error: %v", err)
				continue
			} else {
				log.Printf("error parsing frame: %v", err)
				return err
			}
		}

		c.Send(frame)
	}
}
