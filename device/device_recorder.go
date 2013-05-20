//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

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

// func (d *DeviceRecorder) SetMaxSamples(maxSamples int) {
// 	d.maxSamples = maxSamples
// }

// func (d *DeviceRecorder) Start() (err error) {
// 	d.out, err = d.device.Subscribe("recorder")
// 	if err != nil {
// 		return err
// 	}
// 	return d.r.Start()
// }

// func (d *DeviceRecorder) RecordFrame(df DataFrame) (err error) {
// 	return d.r.RecordFrame(df)
// }

// func (d *DeviceRecorder) Stop() (id string, err error) {
// 	id, err = d.r.Stop()
// }
