package goavatar

import (
	//"log"
	"testing"
	"time"
)

type MockRecorder struct {
}

func (r *MockRecorder) Start() (err error)                     { return }
func (r *MockRecorder) ProcessFrame(df *DataFrame) (err error) { return }
func (r *MockRecorder) Stop() (err error)                      { return }

func newEmptyDevice() *baseDevice {
	connFunc := func() error {
		return nil
	}

	disconnFunc := func() error {
		return nil // do nothing
	}

	streamFunc := func(control <-chan ControlCode, output chan<- *DataFrame) (err error) {
		for {
			select {
			case <-control:
				return
			default:
			}
			time.Sleep(time.Second)
		}
		return
	}

	recorderFunc := func(file string) Recorder {
		return &MockRecorder{}
	}

	return newBaseDevice(
		"UnitTestMockDevice",
		connFunc,
		disconnFunc,
		streamFunc,
		recorderFunc,
	)
}

func TestConnectionLogic(t *testing.T) {
	d := newEmptyDevice()
	d.Connect()
	if !d.Connected() {
		t.Errorf("didn't connect")
	}

	err := d.Connect()
	if err == nil {
		t.Errorf("failed to block second connect")
	}

	d.Disconnect()
	if d.Connected() {
		t.Errorf("failed to disconnect")
	}

	d.Disconnect()
	d.Disconnect()
	d.Disconnect()
	d.Disconnect()

	if d.Connected() {
		t.Errorf("connected now for some reason")
	}

	d.Connect()
	if !d.Connected() {
		t.Errorf("didn't connect for a second time")
	}
}

func TestCleanupLogic(t *testing.T) {
	d := newEmptyDevice()
	if d.out != nil || d.publicOut != nil {
		t.Errorf("has an out channel for some reason")
	}

	err := d.Connect()
	if err != nil {
		t.Errorf("failed to connect")
	}

	if d.out == nil || d.publicOut == nil {
		t.Errorf("didn't create out channel")
	}

	err = d.Disconnect()
	if err != nil {
		t.Errorf("failed to disconnect")
	}

	ensureClosed(t, d.out)

	// wait for worker thread to close the public out
	time.Sleep(time.Millisecond * 500)
	ensureClosed(t, d.publicOut)
}

func ensureClosed(t *testing.T, out chan *DataFrame) {
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	out <- &DataFrame{}
	t.Errorf("failed to panic when writing to this channel, hence it is still open")
}
