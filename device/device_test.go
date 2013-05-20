//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	"testing"
	"time"
)

type MockRecorder struct {
	started   bool
	processed bool
	stopped   bool
}

func (r *MockRecorder) Start() (err error) {
	r.started = true
	return
}
func (r *MockRecorder) ProcessFrame(df DataFrame) (err error) {
	r.processed = true
	return
}
func (r *MockRecorder) Stop() (outFile string, err error) {
	r.stopped = true
	return "somefile", nil
}

func (r *MockRecorder) Reset() {
	r.started = false
	r.processed = false
	r.stopped = false
	return
}

type emptyDevice struct {
	name     string
	repo     string
	errProne bool // will produce errors in stream (for testing)
}

func (ed *emptyDevice) Name() string {
	return ed.name
}

func (ed *emptyDevice) Repo() string {
	return ed.repo
}

func (ed *emptyDevice) Engage() error {
	return nil
}

func (ed *emptyDevice) Disengage() error {
	return nil
}

func (ed *emptyDevice) Stream(c *Control) (err error) {
	c.SendInfo(&DeviceInfo{
		Channels:   2,
		SampleRate: 250,
	})
	if ed.errProne {
		return fmt.Errorf("errProne device is error prone")
	}
	for !c.ShouldTerminate() {

		time.Sleep(time.Millisecond * 1)
		c.Send(&MockFrame{})
	}
	return
}

func newEmptyDevice() Device {
	return NewDevice(&emptyDevice{
		name: "EmptyDevice",
		repo: "var",
	})
}

// Returns a device whose stream always has errors.
func newErrorProneDevice() Device {
	return NewDevice(&emptyDevice{
		name:     "ErrorProneDevice",
		repo:     "var",
		errProne: true,
	})
}

type MockFrame struct {
}

func (f *MockFrame) Buffer() (data *BlockBuffer) {
	return
}

func (f *MockFrame) Channels() (c int) {
	return
}

func (f *MockFrame) Samples() (s int) {
	return
}

func (f *MockFrame) SampleRate() (r int) {
	return
}

func (f *MockFrame) Received() (t time.Time) {
	return
}

func (f *MockFrame) Generated() (t time.Time) {
	return
}

func (f *MockFrame) Timestamps() (ts []int64) {
	return
}

func TestEngageionLogic(t *testing.T) {
	d := newEmptyDevice()
	d.Engage()
	if !d.Engaged() {
		t.Errorf("didn't connect")
	}

	err := d.Engage()
	if err == nil {
		t.Errorf("failed to block second connect")
	}

	d.Disengage()
	if d.Engaged() {
		t.Errorf("failed to disconnect")
	}

	d.Disengage()
	d.Disengage()
	d.Disengage()
	d.Disengage()

	if d.Engaged() {
		t.Errorf("connected now for some reason")
	}

	d.Engage()
	if !d.Engaged() {
		t.Errorf("didn't connect for a second time")
	}
}

func TestControl(t *testing.T) {
	d := newEmptyDevice()
	c := newControl(d.(*BaseDevice))

	go func() {
		time.Sleep(time.Second)
		for !c.ShouldTerminate() {
			time.Sleep(5 * time.Second)
		}
	}()
	c.done <- true
}

func TestCleanupLogic(t *testing.T) {
	d := newEmptyDevice()
	bd := d.(*BaseDevice)
	if bd.control != nil {
		t.Errorf("has recorder/control for some reason")
	}

	err := d.Engage()
	if err != nil {
		t.Errorf("failed to connect")
	}

	if len(bd.subs) != 0 {
		t.Errorf("already has subscriptions?")
	}

	bd.Subscribe("test")

	if len(bd.subs) != 1 {
		t.Errorf("should have 1 subscription")
	}

	err = d.Disengage()
	if err != nil {
		t.Errorf("failed to disconnect")
	}

	if len(bd.subs) != 0 {
		t.Errorf("didn't unsubscribe properly")
	}
}

// func TestRecord(t *testing.T) {
// 	d := newEmptyDevice()
// 	err := d.Engage()
// 	if err != nil || !d.Engaged() {
// 		t.Errorf("failed to connect")
// 	}

// 	err = d.Record()
// 	if err != nil || !d.Recording() {
// 		t.Errorf("failed to start recording, or wrong status")
// 	}

// 	r := d.(*BaseDevice).recorder.(*MockRecorder)
// 	if !r.started {
// 		t.Errorf("mock recorder didn't start")
// 	}

// 	// wait for a single data frame to go through
// 	<-d.Out()

// 	if !r.processed {
// 		t.Errorf("mock recorder didn't process")
// 	}

// 	d.Stop()
// 	if err != nil || d.Recording() {
// 		t.Errorf("recorder failed to stop")
// 	}

// 	if !r.stopped {
// 		t.Errorf("mock recorder didn't hit stop")
// 	}

// 	err = d.Disengage()
// 	if err != nil || d.Engaged() {
// 		t.Errorf("couldn't disconnect: %v", err)
// 	}
// }

// func TestRecordWhenOff(t *testing.T) {
// 	d := newEmptyDevice()
// 	err := d.Record()
// 	if err == nil {
// 		t.Errorf("should have failed, the device is not connected")
// 	}
// }

// func TestMultipleRecording(t *testing.T) {
// 	d := newEmptyDevice()
// 	err := d.Engage()
// 	if err != nil || !d.Engaged() {
// 		t.Errorf("failed to connect")
// 	}

// 	err = d.Record()
// 	if err != nil || !d.Recording() {
// 		t.Errorf("failed to start recording, or wrong status")
// 	}

// 	err = d.Record()
// 	if err == nil {
// 		t.Errorf("should have failed, device is already recording")
// 	}

// 	err = d.Disengage()
// 	if err != nil || d.Engaged() {
// 		t.Errorf("couldn't disconnect: %v", err)
// 	}

// 	if d.Recording() {
// 		t.Errorf("recording didn't stop")
// 	}
// }

func TestErrorProneStream(t *testing.T) {
	d := newErrorProneDevice()
	d.Engage()
	time.Sleep(time.Millisecond * 100) // wait for device to fail
	if d.Engaged() {
		t.Errorf("device should have disconnected, probably")
	}
}

func ensureClosed(t *testing.T, out chan DataFrame) {
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	out <- &MockFrame{}
	t.Errorf("failed to panic when writing to this channel, hence it is still open")
}
