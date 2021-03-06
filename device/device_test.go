//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	. "github.com/jbrukh/goavatar/repo"

	"testing"
	"time"
)

type MockRecorder struct {
	started   bool
	processed bool
	stopped   bool
}

func (r *MockRecorder) Init() (err error) {
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
	repo     *Repository
	errProne bool // will produce errors in stream (for testing)
}

func (ed *emptyDevice) Name() string {
	return ed.name
}

func (ed *emptyDevice) Repo() *Repository {
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
		Channels:   1,
		SampleRate: 250,
	})
	if ed.errProne {
		return fmt.Errorf("errProne device is error prone")
	}
	for !c.ShouldTerminate() {
		time.Sleep(time.Millisecond * 1)
		b := NewBlockBuffer(1, 1)
		b.AppendSample([]float64{42}, time.Now().UnixNano())
		c.Send(&MockFrame{buf: b})
	}
	return
}

func newEmptyDevice() Device {
	return NewDevice(&emptyDevice{
		name: "EmptyDevice",
		repo: NewRepositoryOrPanic("../var/unit-tests/empty-device"),
	})
}

// Returns a device whose stream always has errors.
func newErrorProneDevice() Device {
	return NewDevice(&emptyDevice{
		name:     "ErrorProneDevice",
		repo:     NewRepositoryOrPanic("../var/unit-tests/error-prone-device"),
		errProne: true,
	})
}

type MockFrame struct {
	buf *BlockBuffer
}

func (f *MockFrame) Buffer() (data *BlockBuffer) {
	return f.buf
}

func (f *MockFrame) SampleRate() (r int) {
	return 250
}

func TestEngageLogic(t *testing.T) {
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

	bd.Subscribe("test")

	err = d.Disengage()
	if err != nil {
		t.Errorf("failed to disconnect")
	}

}

func TestDevice__Subscription(t *testing.T) {
	d := newEmptyDevice()
	err := d.Engage()
	if err != nil || !d.Engaged() {
		t.Errorf("failed to engage device")
	}

	out, err := d.Subscribe("test")
	if err != nil {
		t.Errorf("failed to subscribe")
	}

	// suck out 5
	for i := 0; i < 5; i++ {
		fmt.Printf("got: %v\n", <-out)
	}

	d.Unsubscribe("test")
	// check the channel is closed
	if _, ok := <-out; ok {
		t.Errorf("channel is still open")
	}
	fmt.Printf("still %d items on the channel (ok)\n", len(out))

	// now test 2
	out1, err1 := d.Subscribe("1")
	out2, err2 := d.Subscribe("2")
	if err1 != nil || err2 != nil {
		t.Errorf("could not subscribe 2")
	}

	for i := 0; i < 5; i++ {
		fmt.Printf("got: %v %v\n", <-out1, <-out2)
	}
	d.Unsubscribe("1")
	d.Unsubscribe("2")

	if _, ok := <-out1; ok {
		t.Errorf("channel is still open")
	}
	if _, ok := <-out2; ok {
		t.Errorf("channel is still open")
	}

	err = d.Disengage()
	if err != nil {
		t.Fatalf("failed to disengage")
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
