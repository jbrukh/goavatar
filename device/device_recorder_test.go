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
	defer d.Disengage()

	r := NewDeviceRecorder(d, NewOBFRecorder("../var"))

	err := r.RecordAsync()
	if err != nil {
		t.Fatalf("failed to record")
	}

	if !r.Recording() {
		t.Errorf("did not set recording flag")
	}

	// record for 10 ms
	time.Sleep(time.Millisecond * 10)
	r.Stop()

	if r.Recording() {
		t.Errorf("did not set recording flag")
	}

}

func TestRecord__MaxSamples(t *testing.T) {
	d := newEmptyDevice()
	if err := d.Engage(); err != nil || !d.Engaged() {
		t.Errorf("could not engage")
	}
	defer d.Disengage()

	r := NewDeviceRecorder(d, NewOBFRecorder("../var"))
	r.SetMax(2)
	err := r.RecordAsync()
	if err != nil {
		t.Fatalf("failed to record")
	}

	if !r.Recording() {
		t.Errorf("did not set recording flag")
	}

	r.Wait()

	if r.Recording() {
		t.Errorf("did not set recording flag")
	}
}
