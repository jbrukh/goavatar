//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	//"log"
	. "github.com/jbrukh/goavatar/obf/recorder"
	"testing"
	"time"
)

func TestRecord(t *testing.T) {
	d := newEmptyDevice()
	if err := d.Engage(); err != nil || !d.Engaged() {
		t.Errorf("could not engage")
	}
	defer d.Disengage()

	r := NewDeviceRecorder(d, NewObfRecorder(d.Repo()))

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

	r := NewDeviceRecorder(d, NewObfRecorder(d.Repo()))
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

func TestRecord__WaitFail(t *testing.T) {
	d := newEmptyDevice()
	if err := d.Engage(); err != nil || !d.Engaged() {
		t.Errorf("could not engage")
	}
	defer d.Disengage()
	r := NewDeviceRecorder(d, NewObfRecorder(d.Repo()))

	// at this point wait should fail because
	// we are not recording: Stop() and Wait()
	if _, err := r.Stop(); err == nil {
		t.Errorf("should have failed")
	}

	if _, err := r.Wait(); err == nil {
		t.Errorf("should have failed")
	}

	err := r.RecordAsync()
	if err != nil {
		t.Fatalf("failed to record")
	}

	go func() {
		if _, err := r.Wait(); err != nil {
			t.Fatalf("this should have succeeded")
		}
	}()

	// let that thread wait
	time.Sleep(time.Millisecond * 1)
	if _, err := r.Stop(); err == nil {
		t.Fatalf("this should have failed")
	}
}
