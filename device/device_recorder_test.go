package device

import (
	. "github.com/jbrukh/goavatar/formats"
	//"log"
	"testing"
	"time"
)

func TestRecord(t *testing.T) {
	d := newEmptyDevice()
	if err := d.Engage(); err != nil || !d.Engaged() {
		t.Errorf("could not engage")
	}

	r := NewDeviceRecorder(d, NewOBFRecorder("../var"))
	// record for 10 ms
	go func() {
		time.Sleep(time.Millisecond * 10)
		r.Stop()
	}()

	_, err := r.Record()
	if err != nil {
		t.Fatalf("failed to record")
	}
}

func TestRecord__MaxSamples(t *testing.T) {
	d := newEmptyDevice()
	if err := d.Engage(); err != nil || !d.Engaged() {
		t.Errorf("could not engage")
	}

	r := NewDeviceRecorder(d, NewOBFRecorder("../var"))
	r.SetMax(2)
	_, err := r.Record()
	if err != nil {
		t.Fatalf("failed to record")
	}
}
