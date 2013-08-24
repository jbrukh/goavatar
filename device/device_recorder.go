//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	"log"
	"sync"
)

const RecorderName = "recorder"

// A real-time recorder of dataframes. This recorder
// should support calling the given methods in the
// given order: Init, RecordFrame (multiple times),
// and finally stop.
type Recorder interface {
	Init() error
	RecordFrame(DataFrame) error
	Stop() (id string, err error)

	// return the recording length, since
	// the last time Init() was called
	Stats() (ms uint32)
}

type RecordingInfo struct {
	ResourceId string
	DurationMs uint32
}

// DeviceRecorder -- a thread-safe recorder that
// operates on a device and a Recorder implementation.
type DeviceRecorder struct {
	sync.Mutex
	device    Device
	r         Recorder
	cerr      chan error
	recording bool
	max       int // max samples
}

// Create a new DeviceRecorder.
func NewDeviceRecorder(device Device, r Recorder) *DeviceRecorder {
	return &DeviceRecorder{
		device: device,
		r:      r,
	}
}

// Set the maximum number of samples that the recorder will
// read. If this number is set to 0 (default), the recorder
// will record indefinitely until such time that Stop() is
// called.
func (d *DeviceRecorder) SetMax(max int) {
	d.Lock()
	defer d.Unlock()
	if max > 0 {
		d.max = max
	}
}

// Recording returns true if and only if this
// device is currently recording.
func (d *DeviceRecorder) Recording() bool {
	d.Lock()
	defer d.Unlock()
	return d.recording
}

// RecordingTimed returns true if and only if this
// device is currently recording a fixed-time
// recording.
func (d *DeviceRecorder) RecordingTimed() bool {
	d.Lock()
	defer d.Unlock()
	return d.recording && d.max > 0
}

// RecordAsync will subscribe to its device and begin to record
// asynchronously. An error is returned if the device
// cannot be subscribed to. If the subscription is closed (for
// instance, if the device is turned off) then the
// asynchronous worker will exit.
func (d *DeviceRecorder) RecordAsync() (err error) {
	d.Lock()
	defer d.Unlock()

	// already recording?
	if d.recording {
		return fmt.Errorf("already recording")
	}

	// subscribe to the device
	out, err := d.device.Subscribe(RecorderName)
	if err != nil {
		return
	}

	// initialize the underlying recorder
	err = d.r.Init()
	if err != nil {
		return
	}

	// record asynchronously
	d.cerr = make(chan error, 1)
	go worker(d.r, out, d.cerr, d.max)
	d.recording = true
	return
}

// worker will read the frames one by one and write them
// to the Recorder; if we have reached max frames, he will
// stop.
func worker(r Recorder, out chan DataFrame, cerr chan error, max int) {
	defer close(cerr)
	var (
		df      DataFrame
		ok      bool
		count   int
		samples int
	)
	for {
		// take a data frame from the device
		df, ok = <-out
		if !ok {
			return
		}

		// count the samples
		samples = df.Buffer().Samples()
		count += samples

		// respect max samples
		frame, proceed := nextFrame(df, max, count, samples)

		// record the frame
		if err := r.RecordFrame(frame); err != nil {
			cerr <- err
			return
		}

		if !proceed {
			return
		}
	}
}

// nextFrame will decide if we need to proceed writing frames
// with respect to the max frames
func nextFrame(df DataFrame, max, count, samples int) (DataFrame, bool) {
	if max > 0 && count >= max {
		if needed := (samples - count + max); needed < samples {
			buf := df.Buffer().Slice(0, needed)
			df = NewDataFrame(buf, df.SampleRate())
		}
		return df, false
	}
	return df, true
}

// Wait will block until the recorder has finished recording. If
// the recorder is not recording, an error is returned. This method
// is only appropriate when SetMax() has been set, as otherwise
// the device will never stop recording; the proper way to stop
// an indefinitely recording recorder is to call Stop(). If the
// device is recording indefinitely and this method is called
// followed by Stop(), then recording will stop, this method will
// succeed, but the Stop() call will fail with an error.
func (d *DeviceRecorder) Wait() (info *RecordingInfo, err error) {
	// you can only wait on recording
	// devices
	d.Lock()
	recording := d.recording
	d.Unlock()
	if !recording {
		return nil, fmt.Errorf("not recording")
	}

	// wait for the worker
	err, _ = <-d.cerr
	if err != nil {
		log.Printf("wait err: %v", err)
		return
	}

	d.Lock()
	d.recording = false
	defer d.Unlock()

	// stop
	id, err := d.r.Stop()
	if err != nil {
		return nil, err
	}

	// get the recording duration
	// from the last timestamp
	ms := d.r.Stats()

	return &RecordingInfo{
		ResourceId: id,
		DurationMs: ms,
	}, nil
}

// Release the worker.
func (d *DeviceRecorder) Release() {
	d.device.Unsubscribe(RecorderName)
}

// Stop will stop recording and return the details of the
// recorder file. If the device is not recording, then this
// operation will fail.
func (d *DeviceRecorder) Stop() (info *RecordingInfo, err error) {
	// this will cause the worker to exit on the next iteration
	d.Release()
	return d.Wait()
}
