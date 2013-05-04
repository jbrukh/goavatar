//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package mock_avatar

import (
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/devices/avatar"
	"time"
)

// ----------------------------------------------------------------- //
// Mock Avatar Device
// ----------------------------------------------------------------- //

type MockDevice struct {
	BaseDevice
}

// NewDevice creates a new Device. The user can then start
// streaming data by calling Connect() and reading the 
// output channel.
func NewMockDevice() *MockDevice {

	// CONNECT
	connFunc := func() error {
		// simulate startup time
		time.Sleep(time.Second * 1)
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
			c.Send(MockAvatarFrames[tick%len(MockAvatarFrames)])
			tick++
			time.Sleep(time.Millisecond * 64) // 15.625 fps == 1 frame every 64 milliseconds
		}
		return nil
	}

	recorderProvider := func(token string) Recorder {
		return NewFileRecorder(token)
	}

	return &MockDevice{
		BaseDevice: *NewBaseDevice(
			"MockAvatarEEG",
			connFunc,
			disconnFunc,
			streamFunc,
			recorderProvider,
		),
	}
}
