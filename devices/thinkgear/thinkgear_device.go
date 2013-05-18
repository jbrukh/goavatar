//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package thinkgear

import (
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/formats"
)

// ----------------------------------------------------------------- //
// ThinkGear Device
// ----------------------------------------------------------------- //

type ThinkGearDevice struct {
	name       string
	repo       string
	serialPort string
}

func NewThinkGearDevice(serialPort, repo string) Device {
	return NewDevice(&ThinkGearDevice{
		name:       "NeuroSkyDevice",
		repo:       repo,
		serialPort: serialPort,
	})
}

func (d *ThinkGearDevice) Engage() (err error) {
	return nil
}

func (d *ThinkGearDevice) Disengage() (err error) {
	return nil
}

func (d *ThinkGearDevice) Stream(c *Control) (err error) {
	return nil
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
