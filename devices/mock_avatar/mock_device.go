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
	BaseDevice
}

// Mock AvatarEEG device that plays pre-recorded frames on
// repeat. The frames are specified as an OBF file.
func NewMockDevice(repo string, obfFile string) *MockDevice {
	var frames []DataFrame
	// CONNECT
	connFunc := func() (err error) {
		log.Printf("loading up mock data from: %s", obfFile)
		frames, err = MockDataFrames(obfFile)
		if err != nil {
			return err
		}
		return nil
	}

	// DISCONNECT
	disconnFunc := func() error {
		return nil // do nothing
	}

	// STREAM
	streamFunc := func(c *Control) error {
		defer c.Close()
		tick := 0
		for {
			if c.ShouldTerminate() {
				return nil
			}

			// overwrite timestamps so the
			// test recording doesn't repeat timestamps
			var (
				now   = time.Now().UnixNano()
				frame = frames[tick%len(frames)]
				δ     = time.Millisecond * 4
			)
			frame.Buffer().TransformTs(func(s int, ts int64) int64 {
				return InterpolateTs(now, s, δ)
			})

			c.Send(frame)
			tick++
			time.Sleep(time.Millisecond * 64) // 15.625 fps == 1 frame every 64 milliseconds
		}
		return nil
	}

	recorderProvider := func() Recorder {
		return NewOBFRecorder(repo)
	}

	return &MockDevice{
		BaseDevice: *NewBaseDevice(
			"MockAvatarEEG",
			connFunc,
			disconnFunc,
			streamFunc,
			recorderProvider,
			repo,
		),
	}
}
