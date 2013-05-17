//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package thinkgear

import (
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/formats"
	"io"
	"log"
	"os"
)

// ----------------------------------------------------------------- //
// ThinkGear Device
// ----------------------------------------------------------------- //

type ThinkGearDevice struct {
	name       string
	repo       string
	serialPort string
	reader     io.ReadCloser
}

func NewThinkGearDevice(repo, serialPort string) Device {
	return NewDevice(&ThinkGearDevice{
		name:       "NeuroSkyDevice",
		repo:       repo,
		serialPort: serialPort,
	})
}

func (d *ThinkGearDevice) Engage() (err error) {
	d.reader, err = os.Open(d.serialPort)
	return
}

func (d *ThinkGearDevice) Disengage() (err error) {
	return d.reader.Close()
}

func (d *ThinkGearDevice) ProvideRecorder() Recorder {
	return NewOBFRecorder(d.repo)
}

func (d *ThinkGearDevice) Name() string {
	return d.name
}

func (d *ThinkGearDevice) Repo() string {
	return d.repo
}

func (d *ThinkGearDevice) Stream(c *Control) (err error) {
	return parseByteStream(d.reader, c)
}

func parseByteStream(reader io.ReadCloser, c *Control) (err error) {
	parser := NewThinkGearParser(reader)
	defer c.Close()

	c.SendInfo(&DeviceInfo{
		SampleRate: 512,
		Channels:   1,
	})

	for {
		if c.ShouldTerminate() {
			return nil
		}

		frame, err := parser.ParseRaw()
		if err != nil {
			log.Printf("error parsing frame: %v", err)
			return err
		}
		c.Send(frame)
	}
}
