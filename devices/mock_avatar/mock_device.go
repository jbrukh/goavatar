//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package mock_avatar

import (
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/etc"
	. "github.com/jbrukh/goavatar/formats"
	"log"
	"time"
)

// ----------------------------------------------------------------- //
// Mock Avatar Device
// ----------------------------------------------------------------- //

type MockDevice struct {
	frames   []DataFrame
	name     string
	repo     string
	obfFile  string
	channels int
}

// Mock AvatarEEG device that plays pre-recorded frames on
// repeat. The frames are specified as an OBF file.
func NewMockDevice(repo string, obfFile string, channels int) Device {
	if channels < 1 {
		log.Printf("Resetting channels to: 1")
		channels = 1
	}
	return NewDevice(&MockDevice{
		name:     "MockAvatarEEG",
		repo:     repo,
		obfFile:  obfFile,
		channels: channels,
	})
}

func (ad *MockDevice) Engage() (err error) {
	ad.frames, err = MockDataFrames(ad.obfFile)
	if err != nil {
		return err
	}
	return nil
}

func (ad *MockDevice) Disengage() (err error) {
	return nil
}

func (ad *MockDevice) Stream(c *Control) (err error) {
	defer c.Close()
	tick := 0
	for {
		if c.ShouldTerminate() {
			return nil
		}
		frame := ad.getFrame(tick)
		//arr, _ := frame.Buffer().Arrays()
		//log.Printf("sending frame %d: %v", tick, arr)
		c.Send(frame)
		tick = (tick + 1) % len(ad.frames)
		time.Sleep(time.Millisecond * 64) // 15.625 fps == 1 frame every 64 milliseconds
	}
}

func (ad *MockDevice) ProvideRecorder() Recorder {
	return NewOBFRecorder(ad.repo)
}

func (ad *MockDevice) Name() string {
	return ad.name
}

func (ad *MockDevice) Repo() string {
	return ad.repo
}

// Obtain the forward-facing DataFrame. Since the mock
// device works on pre-recorded data, it must overwrite
// the timestamps to maintain a continuous stream of times.
// Further more, it may opt to add (duplicate) some channels
// to satisfy the channels parameter.
func (ad *MockDevice) getFrame(tick int) DataFrame {
	// overwrite timestamps so the
	// test recording doesn't repeat timestamps
	var (
		now   = time.Now().UnixNano()
		frame = ad.frames[tick]
		b     = frame.Buffer()
		δ     = time.Millisecond * 4
	)
	bb := ad.transformBuffer(b)
	bb.TransformTs(func(s int, ts int64) int64 {
		return InterpolateTs(now, s, δ)
	})
	return NewDataFrame(bb, frame.SampleRate())
}

// Appends (or reduces) some channels to the BlockBuffer depending
// on the channels parameter. TODO: create AppendChannel
// method in BlockBuffer.
func (ad *MockDevice) transformBuffer(b *BlockBuffer) (bb *BlockBuffer) {
	bb = b
	if ad.channels != b.Channels() {
		bb = NewBlockBuffer(ad.channels, b.Samples())
		samples := b.Samples()
		for s := 0; s < samples; s++ {
			v, t := b.Sample(s)
			vv := make([]float64, ad.channels)
			for i := range vv {
				vv[i] = v[i%len(v)]
			}
			bb.AppendSample(vv, t)
		}
	}
	return
}
