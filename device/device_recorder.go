//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"log"
)

// A real-time recorder of dataframes.
type Recorder interface {
	Start() error
	RecordFrame(DataFrame) error
	Stop() (id string, err error)
}

type DeviceRecorder struct {
	device      Device
	r           Recorder
	out         chan DataFrame
	sampleCount int
	maxSamples  int
}

func NewDeviceRecorder(device Device, r Recorder) *DeviceRecorder {
	return &DeviceRecorder{
		device: device,
		r:      r,
	}
}

func (d *DeviceRecorder) SetMaxSamples(maxSamples int) {
	d.maxSamples = maxSamples
}

func (d *DeviceRecorder) Record() (id string, err error) {
	d.out, err = d.device.Subscribe("recorder")
	if err != nil {
		return
	}

	err = d.r.Start()
	if err != nil {
		return
	}

	for {
		df, ok := <-d.out
		if !ok {
			break
		}
		d.r.RecordFrame(df)
	}

	log.Printf("recording ended")
	id, err = d.r.Stop()
	return
}

func (d *DeviceRecorder) Stop() {
	d.device.Unsubscribe("recorder")
}
