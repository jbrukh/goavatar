//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package mock_avatar

import (
	. "github.com/jbrukh/goavatar/datastruct"
	. "github.com/jbrukh/goavatar/device"
	. "github.com/jbrukh/goavatar/etc"
	. "github.com/jbrukh/goavatar/formats"
	. "github.com/jbrukh/goavatar/util"
	"log"
	"time"
)

// ----------------------------------------------------------------- //
// Mock Avatar Device
// ----------------------------------------------------------------- //

type MockDevice struct {
	frames   []DataFrame
	name     string
	repo     *Repository
	obfFile  string
	channels int
}

// Mock AvatarEEG device that plays pre-recorded frames on
// repeat. The frames are specified as an OBF file.
func NewMockDevice(basedir string, obfFile string, channels int) Device {
	if channels < 1 {
		log.Printf("Resetting channels to: 1")
		channels = 1
	}
	return NewDevice(&MockDevice{
		name:     "MockAvatarEEG",
		repo:     NewRepository(basedir),
		obfFile:  obfFile,
		channels: channels,
	})
}

func (d *MockDevice) Engage() (err error) {
	d.frames, err = MockDataFrames(d.obfFile)
	if err != nil {
		return err
	}
	return nil
}

func (d *MockDevice) Disengage() (err error) {
	return nil
}

func (d *MockDevice) Stream(c *Control) (err error) {
	tick := 0

	// send device info
	c.SendInfo(&DeviceInfo{
		Channels:   d.channels,
		SampleRate: 250,
	})

	for {
		if c.ShouldTerminate() {
			return nil
		}
		frame := d.getFrame(tick)
		//arr, _ := frame.Buffer().Arrays()
		//log.Printf("sending frame %d: %v", tick, arr)
		c.Send(frame)
		tick = (tick + 1) % len(d.frames)
		time.Sleep(time.Millisecond * 64) // 15.625 fps == 1 frame every 64 milliseconds
	}
}

func (d *MockDevice) ProvideRecorder() Recorder {
	return NewOBFRecorder(d.repo)
}

func (d *MockDevice) Name() string {
	return d.name
}

func (d *MockDevice) Repo() string {
	return d.repo
}

// Obtain the forward-facing DataFrame. Since the mock
// device works on pre-recorded data, it must overwrite
// the timestamps to maintain a continuous stream of times.
// Further more, it may opt to dd (duplicate) some channels
// to satisfy the channels parameter.
func (d *MockDevice) getFrame(tick int) DataFrame {
	// overwrite timestamps so the
	// test recording doesn't repeat timestamps
	var (
		now   = time.Now().UnixNano()
		frame = d.frames[tick]
		b     = frame.Buffer()
		δ     = time.Millisecond * 4
	)
	bb := d.transformBuffer(b)
	bb.TransformTs(func(s int, ts int64) int64 {
		return InterpolateTs(now, s, δ)
	})
	return NewDataFrame(bb, frame.SampleRate())
}

// Appends (or reduces) some channels to the BlockBuffer depending
// on the channels parameter. TODO: create AppendChannel
// method in BlockBuffer.
func (d *MockDevice) transformBuffer(b *BlockBuffer) (bb *BlockBuffer) {
	bb = b
	if d.channels != b.Channels() {
		bb = NewBlockBuffer(d.channels, b.Samples())
		samples := b.Samples()
		for s := 0; s < samples; s++ {
			v, t := b.Sample(s)
			vv := make([]float64, d.channels)
			for i := range vv {
				vv[i] = v[i%len(v)]
			}
			bb.AppendSample(vv, t)
		}
	}
	return
}
