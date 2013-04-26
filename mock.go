package goavatar

import (
	"time"
)

// ----------------------------------------------------------------- //
// Mock Avatar Device
// ----------------------------------------------------------------- //

type MockDevice struct {
	baseDevice
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
	streamFunc := func(offSignal <-chan bool, output chan<- *DataFrame) {
		tick := 0
		for {
			if shouldBreak(offSignal) {
				break
			}
			output <- frames[tick%len(frames)]
			tick++
			time.Sleep(time.Millisecond * 64) // 15.625 fps == 1 frame every 64 milliseconds
		}
	}

	return &MockDevice{
		baseDevice: *newBaseDevice(
			connFunc,
			disconnFunc,
			streamFunc,
		),
	}
}
