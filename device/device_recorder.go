//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	. "github.com/jbrukh/goavatar/datastruct"
	"log"
)

// A real-time recorder of dataframes.
type Recorder interface {
	Init() error
	RecordFrame(DataFrame) error
	Stop() (id string, err error)
}

// DeviceRecorder
type DeviceRecorder struct {
	device Device
	r      Recorder
	out    chan DataFrame
	count  int // sample count
	max    int // max samples
}

func NewDeviceRecorder(device Device, r Recorder) *DeviceRecorder {
	return &DeviceRecorder{
		device: device,
		r:      r,
	}
}

func (d *DeviceRecorder) SetMax(max int) {
	if d.max == 0 {
		d.max = max
	}
}

// Make a recording. This method will block as the recording
// proceeds until a separate thread calls Stop().
func (d *DeviceRecorder) Record() (err error) {
	d.out, err = d.device.Subscribe("recorder")
	if err != nil {
		return
	}

	err = d.r.Init()
	if err != nil {
		return
	}

	var (
		df DataFrame
		ok bool
	)
	for {
		df, ok = <-d.out
		if !ok {
			break
		}
		d.count += df.Buffer().Samples()
		if d.max > 0 && d.count >= d.max {
			if err = d.recordLast(df); err != nil {
				return
			}
			break
		}
		if err = d.r.RecordFrame(df); err != nil {
			return
		}
	}
	return
}

func (d *DeviceRecorder) recordLast(df DataFrame) (err error) {
	var (
		samples = df.Buffer().Samples()
		needed  = samples - (d.count - d.max)
	)
	if needed < samples {
		buf := df.Buffer().Slice(0, needed)
		df = NewDataFrame(buf, df.SampleRate())
	}
	log.Printf("got to record last with : %v", df.Buffer())
	if err = d.r.RecordFrame(df); err != nil {
		return
	}
	return
}

func (d *DeviceRecorder) Stop() (id string, err error) {
	d.device.Unsubscribe("recorder")
	return d.r.Stop()
}
