//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package avatar

import (
	//"fmt"
	"github.com/jbrukh/goavatar/etc"
	"testing"
)

func TestSampleRateVersion(t *testing.T) {
	h := &AvatarHeader{}

	// test 250
	h.FieldSampleRateVersion = 0x03
	if rate := h.SampleRate(); rate != 250 || h.Version() != 3 {
		t.Errorf("Wrong sample rate: %v", rate)
	}

	// test 500
	h.FieldSampleRateVersion = 0x43
	if rate := h.SampleRate(); rate != 500 || h.Version() != 3 {
		t.Errorf("Wrong sample rate: %v", rate)
	}

	// test 1000
	h.FieldSampleRateVersion = 0x83
	if rate := h.SampleRate(); rate != 1000 || h.Version() != 3 {
		t.Errorf("Wrong sample rate: %v", rate)
	}
}

func TestFrameSize(t *testing.T) {
	h := &AvatarHeader{}

	size := h.FrameSize()
	if size != 0 {
		t.Errorf("wrong number of bytes for empty frame: %d", size)
	}

	h.FieldFrameSize = uint16(0x1234)
	size = h.FrameSize()
	if size != 4660 {
		t.Errorf("wrong frame size: %d", size)
	}
}

func TestFrameType(t *testing.T) {
	h := &AvatarHeader{}

	h.FieldFrameType = 0x01
	typ := h.FrameType()
	if typ != 1 {
		t.Errorf("wrong type: %d", typ)
	}
}

func TestFrameCount(t *testing.T) {
	h := &AvatarHeader{}

	h.FieldFrameCount = uint32(0x12345678)
	count := h.FrameCount()
	if count != 305419896 {
		t.Errorf("wrong count: %d", count)
	}
}

func TestChannels(t *testing.T) {
	h := &AvatarHeader{}

	// test 250
	h.FieldChannels = 0x03
	if channels := h.Channels(); channels != 3 || h.HasTriggerChannel() {
		t.Errorf("error parsing channels: %v", channels)
	}

	h.FieldChannels = 0x83
	if channels := h.Channels(); channels != 3 || !h.HasTriggerChannel() {
		t.Errorf("error parsing channels: %v", channels)
	}
}

func TestTimestamps(t *testing.T) {
	df := etc.MockAvatarFrames[0]
	if df.SampleRate() != 250 {
		t.Errorf("expecting frame to have 250 sample rate, but not the case")
	}
	expDiff := int64(4000000) // 4 ms
	ts := df.Timestamps()
	tLast := ts[0]
	for _, v := range ts[1:] {
		d := v - tLast
		if d != expDiff {
			t.Errorf("unexpected time diff: %d", d)
		}
		tLast = v
	}
}
